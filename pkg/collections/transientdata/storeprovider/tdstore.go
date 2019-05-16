/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package storeprovider

import (
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	pb "github.com/hyperledger/fabric/protos/transientstore"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/storeprovider/store/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/storeprovider/store/cache"
)

var logger = flogging.MustGetLogger("memtransientdatastore")

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

	logger.Debugf("[%s] Collection [%s:%s] is a transient data collection", s.channelID, ns, collRWSet.CollectionName)

	for _, wSet := range collRWSet.KvRwSet.Writes {
		s.persistKVWrite(txID, ns, collRWSet.CollectionName, wSet, ttl)
	}

	return nil
}

func (s *store) persistKVWrite(txID, ns, coll string, w *kvrwset.KVWrite, ttl time.Duration) {
	if w.IsDelete {
		logger.Debugf("[%s] Skipping key [%s] in collection [%s] in private data rw-set since it was deleted", s.channelID, w.Key, coll)
		return
	}

	key := api.Key{
		Namespace:  ns,
		Collection: coll,
		Key:        w.Key,
	}

	if s.cache.Get(key) != nil {
		logger.Warningf("[%s] Attempt to update transient data key [%s] in collection [%s]", s.channelID, w.Key, coll)
		return
	}

	s.cache.PutWithExpire(key, w.Value, txID, ttl)
}

func (s *store) getTransientData(txID, ns, coll, key string) *storeapi.ExpiringValue {
	value := s.cache.Get(api.Key{Namespace: ns, Collection: coll, Key: key})
	if value == nil {
		logger.Debugf("[%s] Key [%s] not found in transient store", s.channelID, key)
		return nil
	}

	// Check if the data was stored in the current transaction. If so, ignore it or else an endorsement mismatch may result.
	if value.TxID == txID {
		logger.Debugf("[%s] Key [%s] skipped since it was stored in the current transaction", s.channelID, key)
		return nil
	}

	logger.Debugf("[%s] Key [%s] found in transient store", s.channelID, key)

	return &storeapi.ExpiringValue{Value: value.Value, Expiry: value.ExpiryTime}
}

func (s *store) getTransientDataMultipleKeys(mkey *storeapi.MultiKey) storeapi.ExpiringValues {
	var values storeapi.ExpiringValues
	for _, key := range mkey.Keys {
		value := s.getTransientData(mkey.EndorsedAtTxID, mkey.Namespace, mkey.Collection, key)
		if value != nil {
			values = append(values, value)
		}
	}
	return values
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
