/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package retriever

import (
	"context"
	"sync"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/common/privdata"
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	supportapi "github.com/hyperledger/fabric/extensions/collections/api/support"
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	"github.com/hyperledger/fabric/gossip/comm"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	cb "github.com/hyperledger/fabric/protos/common"
	gproto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/pkg/errors"
	olapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/dissemination"
	"github.com/trustbloc/fabric-peer-ext/pkg/common"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/discovery"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/multirequest"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/requestmgr"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

var logger = flogging.MustGetLogger("ext_offledger")

type support interface {
	Config(channelID, ns, coll string) (*cb.StaticCollectionConfig, error)
	Policy(channel, ns, collection string) (privdata.CollectionAccessPolicy, error)
	BlockPublisher(channelID string) gossipapi.BlockPublisher
}

// Validator is a key/value validator
type Validator func(txID, ns, coll, key string, value []byte) error

// Provider is a collection data data provider.
type Provider struct {
	support
	storeForChannel func(channelID string) olapi.Store
	gossipAdapter   func() supportapi.GossipAdapter
	validators      map[cb.CollectionType]Validator
}

// Option is a provider option
type Option func(p *Provider)

// WithValidator sets the key/value validator
func WithValidator(collType cb.CollectionType, validator Validator) Option {
	return func(p *Provider) {
		p.validators[collType] = validator
	}
}

// NewProvider returns a new collection data provider
func NewProvider(storeProvider func(channelID string) olapi.Store, support support, gossipProvider func() supportapi.GossipAdapter, opts ...Option) olapi.Provider {
	p := &Provider{
		support:         support,
		storeForChannel: storeProvider,
		gossipAdapter:   gossipProvider,
		validators:      make(map[cb.CollectionType]Validator),
	}

	// Apply options
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// RetrieverForChannel returns the collection data retriever for the given channel
func (p *Provider) RetrieverForChannel(channelID string) olapi.Retriever {
	r := &retriever{
		support:       p.support,
		gossipAdapter: p.gossipAdapter(),
		store:         p.storeForChannel(channelID),
		channelID:     channelID,
		reqMgr:        requestmgr.Get(channelID),
		resolvers:     make(map[collKey]resolver),
		validators:    p.validators,
	}

	// Add a handler so that we can remove the resolver for a chaincode that has been upgraded
	p.support.BlockPublisher(channelID).AddCCUpgradeHandler(func(blockNum uint64, txID string, chaincodeID string) error {
		logger.Infof("[%s] Chaincode [%s] has been upgraded. Clearing resolver cache for chaincode.", channelID, chaincodeID)
		r.removeResolvers(chaincodeID)
		return nil
	})

	return r
}

type resolver interface {
	// ResolvePeersForRetrieval resolves to a set of peers from which data should be retrieved
	ResolvePeersForRetrieval() discovery.PeerGroup
}

type collKey struct {
	ns   string
	coll string
}

func newCollKey(ns, coll string) collKey {
	return collKey{ns: ns, coll: coll}
}

type retriever struct {
	support
	gossipAdapter supportapi.GossipAdapter
	channelID     string
	store         olapi.Store
	resolvers     map[collKey]resolver
	lock          sync.RWMutex
	reqMgr        requestmgr.RequestMgr
	validators    map[cb.CollectionType]Validator
}

// GetData gets the values for the data item
func (r *retriever) GetData(ctxt context.Context, key *storeapi.Key) (*storeapi.ExpiringValue, error) {
	values, err := r.GetDataMultipleKeys(ctxt, storeapi.NewMultiKey(key.EndorsedAtTxID, key.Namespace, key.Collection, key.Key))
	if err != nil {
		return nil, err
	}

	if values.Values().IsEmpty() {
		return nil, nil
	}

	return values[0], nil
}

// GetDataMultipleKeys gets the values for the multiple data items in a single call
func (r *retriever) GetDataMultipleKeys(ctxt context.Context, key *storeapi.MultiKey) (storeapi.ExpiringValues, error) {
	authorized, err := r.isAuthorized(key.Namespace, key.Collection)
	if err != nil {
		return nil, err
	}
	if !authorized {
		logger.Infof("[%s] This peer does not have access to the collection [%s:%s]", r.channelID, key.Namespace, key.Collection)
		return nil, nil
	}

	localValues, err := r.getMultipleKeysFromLocal(key)
	if err != nil {
		return nil, err
	}
	if localValues.Values().AllSet() {
		err = r.validateValues(key, localValues)
		if err != nil {
			logger.Warningf(err.Error())
			return nil, err
		}
		return localValues, nil
	}

	res, err := r.getResolver(key.Namespace, key.Collection)
	if err != nil {
		return nil, errors.WithMessagef(err, "unable to get resolver for channel [%s] and [%s:%s]", r.channelID, key.Namespace, key.Collection)
	}

	// Retrieve from the remote peers
	cReq := multirequest.New()
	for _, peer := range res.ResolvePeersForRetrieval() {
		logger.Debugf("Adding request to get data for [%s] from [%s] ...", key, peer)
		cReq.Add(peer.String(), r.getDataFromPeer(key, peer))
	}

	response := cReq.Execute(ctxt)

	// Merge the values with the values received locally
	values := asExpiringValues(localValues.Values().Merge(response.Values))

	if err := r.validateValues(key, values); err != nil {
		logger.Warningf(err.Error())
		return nil, err
	}

	if roles.IsCommitter() {
		r.persistMissingKeys(key, localValues, values)
	}

	return values, nil
}

func (r *retriever) getMultipleKeysFromLocal(key *storeapi.MultiKey) (storeapi.ExpiringValues, error) {
	localValues := make(storeapi.ExpiringValues, len(key.Keys))
	for i, k := range key.Keys {
		value, retrieveErr := r.store.GetData(storeapi.NewKey(key.EndorsedAtTxID, key.Namespace, key.Collection, k))
		if retrieveErr != nil {
			logger.Warningf("[%s] Error getting data from local store for [%s]: %s", r.channelID, key, retrieveErr)
			return nil, errors.WithMessagef(retrieveErr, "unable to get data for channel [%s] and [%s]", r.channelID, key)
		}
		localValues[i] = value
	}
	return localValues, nil
}

func getMissingKeyIndexes(values []*storeapi.ExpiringValue) []int {
	var missingIndexes []int
	for i, v := range values {
		if v == nil {
			missingIndexes = append(missingIndexes, i)
		}
	}
	return missingIndexes
}

// persistMissingKeys persists the keys that were missing from the local store
func (r *retriever) persistMissingKeys(key *storeapi.MultiKey, localValues, values storeapi.ExpiringValues) {
	for _, i := range getMissingKeyIndexes(localValues) {
		v := values[i]
		if v != nil {
			r.persistMissingKey(
				storeapi.NewKey(key.EndorsedAtTxID, key.Namespace, key.Collection, key.Keys[i]),
				&storeapi.ExpiringValue{Value: v.Value, Expiry: v.Expiry},
			)
		}
	}
}

func (r *retriever) persistMissingKey(k *storeapi.Key, val *storeapi.ExpiringValue) {
	logger.Debugf("Persisting key [%s] that was missing from the local store", k)
	collConfig, err := r.Config(r.channelID, k.Namespace, k.Collection)
	if err != nil {
		logger.Warningf("Error persisting key [%s] that was missing from the local store: %s", k, err)
	}
	if err := r.store.PutData(collConfig, k, val); err != nil {
		logger.Warningf("Error persisting key [%s] that was missing from the local store: %s", k, err)
	}
}

func (r *retriever) validateValues(key *storeapi.MultiKey, values storeapi.ExpiringValues) error {
	config, err := r.Config(r.channelID, key.Namespace, key.Collection)
	if err != nil {
		return err
	}

	validate, ok := r.validators[config.Type]
	if !ok {
		// No validator for config
		return nil
	}

	for i, v := range values {
		if v != nil && v.Value != nil {
			err := validate(key.EndorsedAtTxID, key.Namespace, key.Collection, key.Keys[i], v.Value)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *retriever) getDataFromPeer(key *storeapi.MultiKey, endorser *discovery.Member) multirequest.Request {
	return func(ctxt context.Context) (common.Values, error) {
		logger.Debugf("Getting data for [%s] from [%s] ...", key, endorser)

		values, err := r.getData(ctxt, key, endorser)
		if err != nil {
			if err == context.Canceled {
				logger.Debugf("[%s] Request to get data from [%s] for [%s] was cancelled", r.channelID, endorser, key)
			} else {
				logger.Debugf("[%s] Error getting data from [%s] for [%s]: %s", r.channelID, endorser, key, err)
			}
			return nil, err
		}

		return values.Values(), nil
	}
}

func (r *retriever) getResolver(ns, coll string) (resolver, error) {
	key := newCollKey(ns, coll)

	r.lock.RLock()
	resolver, ok := r.resolvers[key]
	r.lock.RUnlock()

	if ok {
		return resolver, nil
	}

	return r.getOrCreateResolver(key)
}

func (r *retriever) getOrCreateResolver(key collKey) (resolver, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	resolver, ok := r.resolvers[key]
	if ok {
		return resolver, nil
	}

	policy, err := r.Policy(r.channelID, key.ns, key.coll)
	if err != nil {
		return nil, err
	}

	resolver = dissemination.New(r.channelID, key.ns, key.coll, policy, r.gossipAdapter)

	r.resolvers[key] = resolver

	return resolver, nil
}

func (r *retriever) removeResolvers(ns string) {
	r.lock.Lock()
	defer r.lock.Unlock()

	for key := range r.resolvers {
		if key.ns == ns {
			logger.Debugf("[%s] Removing resolver [%s:%s] from cache", r.channelID, key.ns, key.coll)
			delete(r.resolvers, key)
		}
	}
}

func (r *retriever) getData(ctxt context.Context, key *storeapi.MultiKey, peers ...*discovery.Member) (storeapi.ExpiringValues, error) {
	logger.Debugf("[%s] Sending Gossip request to %s for data for [%s]", r.channelID, peers, key)

	req := r.reqMgr.NewRequest()

	logger.Debugf("[%s] Creating Gossip request %d for data for [%s]", r.channelID, req.ID(), key)
	msg := r.createCollDataRequestMsg(req, key)

	logger.Debugf("[%s] Sending Gossip request %d for data for [%s]", r.channelID, req.ID(), key)
	r.gossipAdapter.Send(msg, asRemotePeers(peers)...)

	logger.Debugf("[%s] Waiting for response for %d for data for [%s]", r.channelID, req.ID(), key)
	res, err := req.GetResponse(ctxt)
	if err != nil {
		return nil, err
	}

	logger.Debugf("[%s] Got response for %d for data for [%s]", r.channelID, req.ID(), key)

	data := make(storeapi.ExpiringValues, len(key.Keys))
	for i, k := range key.Keys {
		d, ok := res.Data.Get(key.Namespace, key.Collection, k)
		if !ok {
			return nil, errors.Errorf("the response does not contain a value for key [%s:%s:%s]", key.Namespace, key.Collection, k)
		}
		if d.Value == nil {
			data[i] = nil
		} else {
			data[i] = &storeapi.ExpiringValue{Value: d.Value, Expiry: d.Expiry}
		}
	}

	return data, nil
}

// isAuthorized returns true if the local peer has access to the given collection
func (r *retriever) isAuthorized(ns, coll string) (bool, error) {
	policy, err := r.Policy(r.channelID, ns, coll)
	if err != nil {
		return false, errors.WithMessagef(err, "unable to get policy for [%s:%s]", ns, coll)
	}

	localMSPID, err := getLocalMSPID()
	if err != nil {
		return false, errors.WithMessagef(err, "unable to get local MSP ID")
	}

	for _, mspID := range policy.MemberOrgs() {
		if mspID == localMSPID {
			return true, nil
		}
	}

	return false, nil
}

func (r *retriever) createCollDataRequestMsg(req requestmgr.Request, key *storeapi.MultiKey) *gproto.GossipMessage {
	var digests []*gproto.CollDataDigest
	for _, k := range key.Keys {
		digests = append(digests, &gproto.CollDataDigest{
			Namespace:      key.Namespace,
			Collection:     key.Collection,
			Key:            k,
			EndorsedAtTxID: key.EndorsedAtTxID,
		})
	}

	return &gproto.GossipMessage{
		Tag:     gproto.GossipMessage_CHAN_ONLY,
		Channel: []byte(r.channelID),
		Content: &gproto.GossipMessage_CollDataReq{
			CollDataReq: &gproto.RemoteCollDataRequest{
				Nonce:   req.ID(),
				Digests: digests,
			},
		},
	}
}

func asRemotePeers(members []*discovery.Member) []*comm.RemotePeer {
	var peers []*comm.RemotePeer
	for _, m := range members {
		peers = append(peers, &comm.RemotePeer{
			Endpoint: m.Endpoint,
			PKIID:    m.PKIid,
		})
	}
	return peers
}

// getLocalMSPID returns the MSP ID of the local peer. This variable may be overridden by unit tests.
var getLocalMSPID = func() (string, error) {
	return mspmgmt.GetLocalMSP().GetIdentifier()
}

func asExpiringValues(cv common.Values) storeapi.ExpiringValues {
	vals := make(storeapi.ExpiringValues, len(cv))
	for i, v := range cv {
		if common.IsNil(v) {
			vals[i] = nil
		} else {
			vals[i] = v.(*storeapi.ExpiringValue)
		}
	}
	return vals
}
