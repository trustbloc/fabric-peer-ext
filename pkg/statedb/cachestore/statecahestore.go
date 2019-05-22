/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package cachestore

import (
	"github.com/bluele/gcache"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	statecahe "github.com/trustbloc/fabric-peer-ext/pkg/statedb/api/cache"
)

var logger = flogging.MustGetLogger("statecache")

// StateCacheStore state cache store
type StateCacheStore struct {
	cache gcache.Cache
	name  string
}

// NewStateCacheStore constructs an instance of State Cache store
func NewStateCacheStore(cacheName string, size int) statecahe.StateCache {
	cache := gcache.New(size).ARC().Build()

	return &StateCacheStore{cache: cache, name: cacheName}
}

// Get implements method in StateCache interface
func (store *StateCacheStore) Get(ns string, key string) *statedb.VersionedValue {
	val, err := store.cache.Get(ConstructCompositeKey(ns, key))
	if err != nil {
		if err != gcache.KeyNotFoundError {
			logger.Errorf("Retrieving State from Cache failed : %s", err)
		}
		return nil
	}

	if val == nil {
		return nil
	}

	logger.Debugf("Successfully retrieved state data from Cache :: ns=%s key=%s", ns, key)
	return val.(*statedb.VersionedValue)
}

// Add implements method in StateCache interface
func (store *StateCacheStore) Add(ns, key string, vval *statedb.VersionedValue) {
	err := store.cache.Set(ConstructCompositeKey(ns, key), vval)
	if err != nil {
		logger.Errorf("Adding State to Cache failed : %s", err)
	}
	logger.Debugf("Successfully added state data to Cache :: ns=%s key=%s", ns, key)
}

// Remove implements method in StateCache interface
func (store *StateCacheStore) Remove(ns string, key string) bool {
	logger.Debugf("Remove state data from Cache :: ns=%s key=%s", ns, key)
	return store.cache.Remove(ConstructCompositeKey(ns, key))
}

// GetVersion implements method in StateCache interface
func (store *StateCacheStore) GetVersion(namespace string, key string) *version.Height {
	versionedValue := store.Get(namespace, key)
	if versionedValue == nil {
		return nil
	}

	return versionedValue.Version
}
