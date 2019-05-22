/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package statedb

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/trustbloc/fabric-peer-ext/pkg/config"
	statecahe "github.com/trustbloc/fabric-peer-ext/pkg/statedb/api/cache"
	"github.com/trustbloc/fabric-peer-ext/pkg/statedb/cachestore"
)

var logger = flogging.MustGetLogger("statedb")

// VersionedDBProvider implements interface VersionedDBProvider
type VersionedDBProvider struct {
	statedbProvider statedb.VersionedDBProvider
}

// NewVersionedDBProvider instantiates VersionedDBProvider
func NewVersionedDBProvider(vdbProvider statedb.VersionedDBProvider) statedb.VersionedDBProvider {
	return &VersionedDBProvider{statedbProvider: vdbProvider}
}

// GetDBHandle gets the handle to a named database
func (provider *VersionedDBProvider) GetDBHandle(dbName string) (statedb.VersionedDB, error) {
	return newVersionedDB(provider, dbName), nil
}

// Close closes the underlying db
func (provider *VersionedDBProvider) Close() {
	provider.statedbProvider.Close()
}

// VersionedDB implements VersionedDB interface
type versionedDB struct {
	statedb         statedb.VersionedDB
	statecachestore statecahe.StateCache
}

// newVersionedDB constructs an instance of VersionedDB
func newVersionedDB(provider *VersionedDBProvider, dbName string) *versionedDB {
	statedbHandle, _ := provider.statedbProvider.GetDBHandle(dbName)
	statecachestore := cachestore.NewStateCacheStore(dbName, config.GetStateDataCacheSize())

	return &versionedDB{statedb: statedbHandle, statecachestore: statecachestore}
}

// Open implements method in VersionedDB interface
func (vdb *versionedDB) Open() error {
	return vdb.statedb.Open()
}

// Close implements method in VersionedDB interface
func (vdb *versionedDB) Close() {
	vdb.statedb.Close()
}

// ValidateKeyValue implements method in VersionedDB interface
func (vdb *versionedDB) ValidateKeyValue(key string, value []byte) error {
	return vdb.statedb.ValidateKeyValue(key, value)
}

// BytesKeySupported implements method in VersionedDB interface
func (vdb *versionedDB) BytesKeySupported() bool {
	return vdb.statedb.BytesKeySupported()
}

// GetState implements method in VersionedDB interface
func (vdb *versionedDB) GetState(namespace string, key string) (*statedb.VersionedValue, error) {

	val := vdb.statecachestore.Get(namespace, key)

	if val != nil {
		return val, nil
	}

	val, err := vdb.statedb.GetState(namespace, key)

	if err == nil && val != nil {
		go func() {
			vdb.statecachestore.Add(namespace, key, val)
		}()
	}

	return val, err
}

// GetVersion implements method in VersionedDB interface
func (vdb *versionedDB) GetVersion(namespace string, key string) (*version.Height, error) {
	versionedValue, err := vdb.GetState(namespace, key)
	if err != nil {
		return nil, err
	}
	if versionedValue == nil {
		return nil, nil
	}

	return versionedValue.Version, nil
}

// GetStateMultipleKeys implements method in VersionedDB interface
func (vdb *versionedDB) GetStateMultipleKeys(namespace string, keys []string) ([]*statedb.VersionedValue, error) {
	vals := make([]*statedb.VersionedValue, len(keys))
	for i, key := range keys {
		val, err := vdb.GetState(namespace, key)
		if err != nil {
			return nil, err
		}
		vals[i] = val
	}
	return vals, nil
}

// GetStateRangeScanIterator implements method in VersionedDB interface
// startKey is inclusive
// endKey is exclusive
func (vdb *versionedDB) GetStateRangeScanIterator(namespace string, startKey string, endKey string) (statedb.ResultsIterator, error) {
	return vdb.GetStateRangeScanIteratorWithMetadata(namespace, startKey, endKey, nil)
}

// GetStateRangeScanIteratorWithMetadata implements method in VersionedDB interface
func (vdb *versionedDB) GetStateRangeScanIteratorWithMetadata(namespace string, startKey string, endKey string, metadata map[string]interface{}) (statedb.QueryResultsIterator, error) {
	return vdb.statedb.GetStateRangeScanIteratorWithMetadata(namespace, startKey, endKey, metadata)
}

// ExecuteQuery implements method in VersionedDB interface
func (vdb *versionedDB) ExecuteQuery(namespace, query string) (statedb.ResultsIterator, error) {
	return vdb.statedb.ExecuteQuery(namespace, query)
}

// ExecuteQueryWithMetadata implements method in VersionedDB interface
func (vdb *versionedDB) ExecuteQueryWithMetadata(namespace, query string, metadata map[string]interface{}) (statedb.QueryResultsIterator, error) {
	return vdb.statedb.ExecuteQueryWithMetadata(namespace, query, metadata)
}

// ApplyUpdates implements method in VersionedDB interface
func (vdb *versionedDB) ApplyUpdates(batch *statedb.UpdateBatch, height *version.Height) error {
	vdb.applyUpdatesToCache(batch)

	return vdb.statedb.ApplyUpdates(batch, height)
}

// GetLatestSavePoint implements method in VersionedDB interface
func (vdb *versionedDB) GetLatestSavePoint() (*version.Height, error) {
	return vdb.statedb.GetLatestSavePoint()
}

// ProcessIndexesForChaincodeDeploy creates indexes for a specified namespace
func (vdb *versionedDB) ProcessIndexesForChaincodeDeploy(namespace string, fileEntries []*ccprovider.TarFileEntry) error {
	//Check to see if the interface for IndexCapable is implemented
	indexCapable, ok := vdb.statedb.(statedb.IndexCapable)
	if !ok {
		return nil
	}

	return indexCapable.ProcessIndexesForChaincodeDeploy(namespace, fileEntries)
}

// GetDBType returns the hosted stateDB
func (vdb *versionedDB) GetDBType() string {
	//Check to see if the interface for IndexCapable is implemented
	indexCapable, ok := vdb.statedb.(statedb.IndexCapable)
	if !ok {
		return ""
	}

	return indexCapable.GetDBType()
}

// ApplyUpdates applies state data batch updates to cache store
func (vdb *versionedDB) applyUpdatesToCache(batch *statedb.UpdateBatch) {
	namespaces := batch.GetUpdatedNamespaces()
	for _, ns := range namespaces {
		updates := batch.GetUpdates(ns)
		for key, vv := range updates {
			compositeKey := cachestore.ConstructCompositeKey(ns, key)
			logger.Debugf("statecachestore : Applying key(string)=[%s]", compositeKey)

			if vv.Value == nil {
				vdb.statecachestore.Remove(ns, key)
			} else {
				vdb.statecachestore.Add(ns, key, vv)
			}
		}
	}
}
