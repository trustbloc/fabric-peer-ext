/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package storeprovider

import (
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	gapi "github.com/hyperledger/fabric/gossip/api"
	gcommon "github.com/hyperledger/fabric/gossip/common"
	gdiscovery "github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/service"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	pb "github.com/hyperledger/fabric/protos/transientstore"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/dissemination"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/storeprovider/store/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/storeprovider/store/cache"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/discovery"
)

var logger = flogging.MustGetLogger("transientdata")

type store struct {
	channelID string
	cache     *cache.Cache
}

type db interface {
	AddKey(api.Key, *api.Value) error
	DeleteExpiredKeys() error
	GetKey(key api.Key) (*api.Value, error)
}

func newStore(channelID string, cacheSize int, transientDB db) *store {
	logger.Debugf("[%s] Creating new store - cacheSize=%d", channelID, cacheSize)
	return &store{
		channelID: channelID,
		cache:     cache.New(cacheSize, transientDB),
	}
}

// Persist persists all transient data within the private data simulation results
func (s *store) Persist(txID string, privateSimulationResultsWithConfig *pb.TxPvtReadWriteSetWithConfigInfo) error {
	rwSet, err := rwsetutil.TxPvtRwSetFromProtoMsg(privateSimulationResultsWithConfig.PvtRwset)
	if err != nil {
		return errors.WithMessage(err, "error getting pvt RW set from bytes")
	}

	for _, nsRWSet := range rwSet.NsPvtRwSet {
		for _, collRWSet := range nsRWSet.CollPvtRwSets {
			if err := s.persistColl(txID, nsRWSet.NameSpace, privateSimulationResultsWithConfig.CollectionConfigs, collRWSet); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetTransientData returns the transient data for the given key
func (s *store) GetTransientData(key *storeapi.Key) (*storeapi.ExpiringValue, error) {
	return s.getTransientData(key.EndorsedAtTxID, key.Namespace, key.Collection, key.Key), nil
}

// GetTransientData returns the transient data for the given keys
func (s *store) GetTransientDataMultipleKeys(key *storeapi.MultiKey) (storeapi.ExpiringValues, error) {
	return s.getTransientDataMultipleKeys(key), nil
}

// Close closes the transient data store
func (s *store) Close() {
	if s.cache != nil {
		logger.Debugf("[%s] Closing cache", s.channelID)
		s.cache.Close()
		s.cache = nil
	}
}

func (s *store) persistColl(txID string, ns string, collConfigPkgs map[string]*common.CollectionConfigPackage, collRWSet *rwsetutil.CollPvtRwSet) error {
	config, exists := getCollectionConfig(collConfigPkgs, ns, collRWSet.CollectionName)
	if !exists {
		logger.Debugf("[%s] Config for collection [%s:%s] not found in config packages", s.channelID, ns, collRWSet.CollectionName)
		return nil
	}

	ttl, err := time.ParseDuration(config.TimeToLive)
	if err != nil {
		// This shouldn't happen since the config was validated before being persisted
		return errors.Wrapf(err, "error parsing time-to-live for collection [%s]", collRWSet.CollectionName)
	}

	logger.Debugf("[%s] Collection [%s:%s] is a transient data collection. Persisting kv-writes...", s.channelID, ns, collRWSet.CollectionName)

	policy, err := s.loadPolicy(ns, config)
	if err != nil {
		return err
	}

	resolver := getResolver(s.channelID, ns, collRWSet.CollectionName, policy, gossipProvider())

	for _, wSet := range collRWSet.KvRwSet.Writes {
		endorsers, err := resolver.ResolveEndorsers(wSet.Key)
		if err != nil {
			return err
		}
		logger.Debugf("[%s] Endorsers for key [%s:%s:%s]: %s", s.channelID, ns, collRWSet.CollectionName, wSet.Key, endorsers)

		if !endorsers.ContainsLocal() {
			logger.Debugf("[%s] Not persisting [%s:%s:%s] since local endorser is not part of the endorser group for this key", s.channelID, ns, collRWSet.CollectionName, wSet.Key)
			continue
		}
		s.persistKVWrite(txID, ns, collRWSet.CollectionName, wSet, ttl)
	}

	return nil
}

func (s *store) persistKVWrite(txID, ns, coll string, w *kvrwset.KVWrite, ttl time.Duration) {
	if w.IsDelete {
		logger.Debugf("[%s] Skipping key [%s:%s:%s] in private data rw-set since it was deleted", s.channelID, ns, coll, w.Key)
		return
	}

	key := api.Key{
		Namespace:  ns,
		Collection: coll,
		Key:        w.Key,
	}

	if s.cache.Get(key) != nil {
		logger.Debugf("[%s] Transient data key [%s:%s:%s] already exists", s.channelID, ns, coll, w.Key)
		return
	}

	logger.Debugf("[%s] Add transient data key [%s]", s.channelID, key)
	s.cache.PutWithExpire(key, w.Value, txID, ttl)
}

func (s *store) getTransientData(txID, ns, coll, key string) *storeapi.ExpiringValue {
	k := api.Key{Namespace: ns, Collection: coll, Key: key}
	value := s.cache.Get(k)
	if value == nil {
		logger.Debugf("[%s] Key [%s] not found in transient store", s.channelID, k)
		return nil
	}

	// Check if the data was stored in the current transaction. If so, ignore it or else an endorsement mismatch may result.
	if value.TxID == txID {
		logger.Debugf("[%s] Key [%s] skipped since it was stored in the current transaction", s.channelID, k)
		return nil
	}

	logger.Debugf("[%s] Key [%s] found in transient store", s.channelID, k)

	return &storeapi.ExpiringValue{Value: value.Value, Expiry: value.ExpiryTime}
}

func (s *store) getTransientDataMultipleKeys(mkey *storeapi.MultiKey) storeapi.ExpiringValues {
	var values storeapi.ExpiringValues
	for _, key := range mkey.Keys {
		values = append(values, s.getTransientData(mkey.EndorsedAtTxID, mkey.Namespace, mkey.Collection, key))
	}
	return values
}

func (s *store) loadPolicy(ns string, config *common.StaticCollectionConfig) (privdata.CollectionAccessPolicy, error) {
	logger.Debugf("[%s] Loading collection policy for [%s:%s]", s.channelID, ns, config.Name)

	colAP := &privdata.SimpleCollection{}
	err := colAP.Setup(config, mspmgmt.GetIdentityDeserializer(s.channelID))
	if err != nil {
		return nil, errors.Wrapf(err, "error setting up collection policy %s", config.Name)
	}

	return colAP, nil
}

func getCollectionConfig(collConfigPkgs map[string]*common.CollectionConfigPackage, namespace, collName string) (*common.StaticCollectionConfig, bool) {
	collConfigPkg, ok := collConfigPkgs[namespace]
	if !ok {
		return nil, false
	}

	for _, collConfig := range collConfigPkg.Config {
		transientConfig := collConfig.GetStaticCollectionConfig()
		if transientConfig != nil && transientConfig.Type == common.CollectionType_COL_TRANSIENT && transientConfig.Name == collName {
			return transientConfig, true
		}
	}

	return nil, false
}

type gossipAdapter interface {
	PeersOfChannel(gcommon.ChainID) []gdiscovery.NetworkMember
	SelfMembershipInfo() gdiscovery.NetworkMember
	IdentityInfo() gapi.PeerIdentitySet
}

var gossipProvider = func() gossipAdapter {
	return service.GetGossipService()
}

type endorserResolver interface {
	ResolveEndorsers(key string) (discovery.PeerGroup, error)
}

var getResolver = func(channelID, ns, coll string, policy privdata.CollectionAccessPolicy, gossip gossipAdapter) endorserResolver {
	return dissemination.New(channelID, ns, coll, policy, gossipProvider())
}
