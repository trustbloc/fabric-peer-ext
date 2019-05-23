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
	gproto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/pkg/errors"
	tdataapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/dissemination"
	"github.com/trustbloc/fabric-peer-ext/pkg/common"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/discovery"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/multirequest"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/requestmgr"
)

var logger = flogging.MustGetLogger("transientdata")

type support interface {
	Policy(channel, ns, collection string) (privdata.CollectionAccessPolicy, error)
	BlockPublisher(channelID string) gossipapi.BlockPublisher
}

// Provider is a transient data provider.
type Provider struct {
	support
	storeForChannel func(channelID string) tdataapi.Store
	gossipAdapter   func() supportapi.GossipAdapter
}

// NewProvider returns a new transient data provider
func NewProvider(storeProvider func(channelID string) tdataapi.Store, support support, gossipProvider func() supportapi.GossipAdapter) tdataapi.Provider {
	return &Provider{
		support:         support,
		storeForChannel: storeProvider,
		gossipAdapter:   gossipProvider,
	}
}

// RetrieverForChannel returns the transient data dataRetriever for the given channel
func (p *Provider) RetrieverForChannel(channelID string) tdataapi.Retriever {
	r := &retriever{
		support:       p.support,
		gossipAdapter: p.gossipAdapter(),
		store:         p.storeForChannel(channelID),
		channelID:     channelID,
		reqMgr:        requestmgr.Get(channelID),
		resolvers:     make(map[collKey]resolver),
	}

	// Add a handler so that we can remove the resolver for a chaincode that has been upgraded
	p.support.BlockPublisher(channelID).AddCCUpgradeHandler(func(blockNum uint64, txID string, chaincodeID string) error {
		logger.Infof("[%s] Chaincode [%s] has been upgraded. Clearing resolver cache for chaincode.", channelID, chaincodeID)
		r.removeResolvers(chaincodeID)
		return nil
	})

	return r
}

// ResolveEndorsers resolves to a set of endorsers to which transient data should be disseminated
type resolver interface {
	ResolveEndorsers(key string) (discovery.PeerGroup, error)
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
	channelID     string
	gossipAdapter supportapi.GossipAdapter
	store         tdataapi.Store
	resolvers     map[collKey]resolver
	lock          sync.RWMutex
	reqMgr        requestmgr.RequestMgr
}

func (r *retriever) GetTransientData(ctxt context.Context, key *storeapi.Key) (*storeapi.ExpiringValue, error) {
	authorized, err := r.isAuthorized(key.Namespace, key.Collection)
	if err != nil {
		return nil, err
	}
	if !authorized {
		logger.Infof("[%s] This peer does not have access to the collection [%s:%s]", r.channelID, key.Namespace, key.Collection)
		return nil, nil
	}

	endorsers, err := r.resolveEndorsers(key)
	if err != nil {
		return nil, errors.WithMessagef(err, "unable to get resolve endorsers for channel [%s] and [%s]", r.channelID, key)
	}

	logger.Debugf("[%s] Endorsers for [%s]: %s", r.channelID, key, endorsers)

	if endorsers.ContainsLocal() {
		value, ok, err := r.getTransientDataFromLocal(key)
		if err != nil {
			return nil, err
		}
		if ok {
			return value, nil
		}
	}

	return r.getTransientDataFromRemote(ctxt, key, endorsers.Remote())
}

type valueResp struct {
	value *storeapi.ExpiringValue
	err   error
}

// GetTransientDataMultipleKeys gets the values for the multiple transient data items in a single call
func (r *retriever) GetTransientDataMultipleKeys(ctxt context.Context, key *storeapi.MultiKey) (storeapi.ExpiringValues, error) {
	if len(key.Keys) == 0 {
		return nil, errors.New("at least one key must be specified")
	}

	var wg sync.WaitGroup
	wg.Add(len(key.Keys))
	var mutex sync.Mutex

	// TODO: This can be optimized by sending one request to endorsers that have multiple keys, as opposed to one request per key.
	responses := make(map[string]*valueResp)
	for _, k := range key.Keys {
		go func(key *storeapi.Key) {
			cctxt, cancel := context.WithCancel(ctxt)
			defer cancel()

			value, err := r.GetTransientData(cctxt, key)
			mutex.Lock()
			responses[key.Key] = &valueResp{value: value, err: err}
			logger.Debugf("Got response for [%s]: %s, Err: %s", key.Key, value, err)
			mutex.Unlock()
			wg.Done()
		}(storeapi.NewKey(key.EndorsedAtTxID, key.Namespace, key.Collection, k))
	}

	wg.Wait()

	// Return the responses in the order of the requested keys
	values := make(storeapi.ExpiringValues, len(key.Keys))
	for i, k := range key.Keys {
		r, ok := responses[k]
		if !ok {
			return nil, errors.Errorf("no response for key [%s:%s:%s]", key.Namespace, key.Collection, k)
		}
		if r.err != nil {
			return nil, r.err
		}
		values[i] = r.value
	}

	return values, nil
}

// resolveEndorsers returns the endorsers that (should) have the transient data
func (r *retriever) resolveEndorsers(key *storeapi.Key) (discovery.PeerGroup, error) {
	res, err := r.getResolver(key.Namespace, key.Collection)
	if err != nil {
		return nil, errors.WithMessagef(err, "unable to get resolver for channel [%s] and [%s:%s]", r.channelID, key.Namespace, key.Collection)
	}
	return res.ResolveEndorsers(key.Key)
}

func (r *retriever) getTransientDataFromLocal(key *storeapi.Key) (*storeapi.ExpiringValue, bool, error) {
	value, retrieveErr := r.store.GetTransientData(key)
	if retrieveErr != nil {
		logger.Debugf("[%s] Error getting transient data from local store for [%s]: %s", r.channelID, key, retrieveErr)
		return nil, false, errors.WithMessagef(retrieveErr, "unable to get transient data for channel [%s] and [%s]", r.channelID, key)
	}

	if value != nil && len(value.Value) > 0 {
		logger.Debugf("[%s] Got transient data from local store for [%s]", r.channelID, key)
		return value, true, nil
	}

	logger.Debugf("[%s] nil transient data in local store for [%s]. Will try to pull from remote peer(s).", r.channelID, key)
	return nil, false, nil
}

func (r *retriever) getTransientDataFromRemote(ctxt context.Context, key *storeapi.Key, endorsers discovery.PeerGroup) (*storeapi.ExpiringValue, error) {
	cReq := multirequest.New()
	for _, endorser := range endorsers {
		logger.Debugf("Adding request to get transient data for [%s] from [%s] ...", key, endorser)
		cReq.Add(endorser.String(), r.getTransientDataRequest(key, endorser))
	}

	response := cReq.Execute(ctxt)

	if response.Values.IsEmpty() {
		logger.Debugf("Got empty transient data response for [%s] ...", key)
		return nil, nil
	}

	logger.Debugf("Got non-nil transient data response for [%s] ...", key)
	return response.Values[0].(*storeapi.ExpiringValue), nil
}

func (r *retriever) getTransientDataRequest(key *storeapi.Key, endorser *discovery.Member) func(ctxt context.Context) (common.Values, error) {
	return func(ctxt context.Context) (common.Values, error) {
		return r.getTransientDataFromEndorser(ctxt, key, endorser)
	}
}

func (r *retriever) getTransientDataFromEndorser(ctxt context.Context, key *storeapi.Key, endorser *discovery.Member) (common.Values, error) {
	logger.Debugf("Getting transient data for [%s] from [%s] ...", key, endorser)

	value, err := r.getTransientData(ctxt, key, endorser)
	if err != nil {
		if err == context.Canceled {
			logger.Debugf("[%s] Request to get transient data from [%s] for [%s] was cancelled", r.channelID, endorser, key)
		} else {
			logger.Debugf("[%s] Error getting transient data from [%s] for [%s]: %s", r.channelID, endorser, key, err)
		}
		return common.Values{nil}, err
	}

	if value == nil {
		logger.Debugf("[%s] Transient data not found on [%s] for [%s]", r.channelID, endorser, key)
	} else {
		logger.Debugf("[%s] Got transient data from [%s] for [%s]", r.channelID, endorser, key)
	}

	return common.Values{value}, nil
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

func (r *retriever) getTransientData(ctxt context.Context, key *storeapi.Key, endorsers ...*discovery.Member) (*storeapi.ExpiringValue, error) {
	logger.Debugf("[%s] Sending Gossip request to %s for transient data for [%s]", r.channelID, endorsers, key)

	req := r.reqMgr.NewRequest()

	logger.Debugf("[%s] Creating Gossip request %d for transient data for [%s]", r.channelID, req.ID(), key)
	msg := r.createCollDataRequestMsg(req, key)

	logger.Debugf("[%s] Sending Gossip request %d for transient data for [%s]", r.channelID, req.ID(), key)
	r.gossipAdapter.Send(msg, asRemotePeers(endorsers)...)

	logger.Debugf("[%s] Waiting for response for %d for transient data for [%s]", r.channelID, req.ID(), key)
	res, err := req.GetResponse(ctxt)
	if err != nil {
		return nil, err
	}

	logger.Debugf("[%s] Got response for %d for transient data for [%s]", r.channelID, req.ID(), key)
	return r.findValue(res.Data, key)
}

func (r *retriever) findValue(data requestmgr.Elements, key *storeapi.Key) (*storeapi.ExpiringValue, error) {
	for _, e := range data {
		if e.Namespace == key.Namespace && e.Collection == key.Collection && e.Key == key.Key {
			logger.Debugf("[%s] Got response for transient data for [%s]", r.channelID, key)
			if e.Value == nil {
				return nil, nil
			}
			return &storeapi.ExpiringValue{Value: e.Value, Expiry: e.Expiry}, nil
		}
	}
	return nil, errors.Errorf("expecting a response to a transient data request for [%s] but got a response for another key", key)
}

func (r *retriever) createCollDataRequestMsg(req requestmgr.Request, key *storeapi.Key) *gproto.GossipMessage {
	return &gproto.GossipMessage{
		Tag:     gproto.GossipMessage_CHAN_ONLY,
		Channel: []byte(r.channelID),
		Content: &gproto.GossipMessage_CollDataReq{
			CollDataReq: &gproto.RemoteCollDataRequest{
				Nonce: req.ID(),
				Digests: []*gproto.CollDataDigest{
					{
						Namespace:      key.Namespace,
						Collection:     key.Collection,
						Key:            key.Key,
						EndorsedAtTxID: key.EndorsedAtTxID,
					},
				},
			},
		},
	}
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
