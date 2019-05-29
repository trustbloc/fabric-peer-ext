/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"path/filepath"
	"time"

	"github.com/hyperledger/fabric/core/config"
	"github.com/hyperledger/fabric/core/ledger/util/couchdb"
	"github.com/spf13/viper"
)

const (
	confPeerFileSystemPath = "peer.fileSystemPath"
	confLedgerDataPath     = "ledgersData"
	confPvtdataStore       = "pvtdataStore"

	confRoles            = "ledger.roles"
	confPvtDataCacheSize = "ledger.blockchain.pvtDataStorage.cacheSize"

	confTransientDataLeveldb             = "transientDataLeveldb"
	confTransientDataCleanupIntervalTime = "coll.transientdata.cleanupExpired.Interval"
	confTransientDataCacheSize           = "coll.transientdata.cacheSize"
	confTransientDataPullTimeout         = "peer.gossip.transientData.pullTimeout"

	confOLCollLeveldb              = "offLedgerLeveldb"
	confOLCollCleanupIntervalTime  = "coll.offledger.cleanupExpired.Interval"
	confOLCollMaxPeersForRetrieval = "coll.offledger.maxpeers"
	confOLCollCacheSize            = "coll.offledger.cacheSize"
	confOLCollPullTimeout          = "coll.offledger.gossip.pullTimeout"

	confBlockPublisherBufferSize = "blockpublisher.buffersize"

	defaultTransientDataCleanupIntervalTime = 5 * time.Second
	defaultTransientDataCacheSize           = 100000
	defaultTransientDataPullTimeout         = 5 * time.Second

	defaultOLCollCleanupIntervalTime  = 5 * time.Second
	defaultOLCollMaxPeersForRetrieval = 2
	defaultOLCollCacheSize            = 10000
	defaultOLCollPullTimeout          = 5 * time.Second

	defaultBlockPublisherBufferSize = 100
)

// GetRoles returns the roles of the peer. Empty return value indicates that the peer has all roles.
func GetRoles() string {
	return viper.GetString(confRoles)
}

// GetPvtDataCacheSize returns the number of pvt data per block to keep the in the cache
func GetPvtDataCacheSize() int {
	pvtDataCacheSize := viper.GetInt(confPvtDataCacheSize)
	if !viper.IsSet(confPvtDataCacheSize) {
		pvtDataCacheSize = 10
	}
	return pvtDataCacheSize
}

// GetTransientDataLevelDBPath returns the filesystem path that is used to maintain the transient data level db
func GetTransientDataLevelDBPath() string {
	return filepath.Join(GetLedgerConfigRootPath(), confTransientDataLeveldb)
}

// GetTransientDataExpiredIntervalTime is time when background routine check expired transient data in db to cleanup.
func GetTransientDataExpiredIntervalTime() time.Duration {
	timeout := viper.GetDuration(confTransientDataCleanupIntervalTime)
	if timeout == 0 {
		return defaultTransientDataCleanupIntervalTime
	}
	return timeout
}

// GetTransientDataCacheSize returns the size of the transient data cache
func GetTransientDataCacheSize() int {
	size := viper.GetInt(confTransientDataCacheSize)
	if size <= 0 {
		return defaultTransientDataCacheSize
	}
	return size
}

// GetOLCollLevelDBPath returns the filesystem path that is used to maintain the off-ledger level db
func GetOLCollLevelDBPath() string {
	return filepath.Join(GetLedgerConfigRootPath(), confOLCollLeveldb)
}

// GetOLCollExpirationCheckInterval is time when the background routine checks expired collection data in db to cleanup.
func GetOLCollExpirationCheckInterval() time.Duration {
	timeout := viper.GetDuration(confOLCollCleanupIntervalTime)
	if timeout == 0 {
		return defaultOLCollCleanupIntervalTime
	}
	return timeout
}

// GetTransientDataPullTimeout is the amount of time a peer waits for a response from another peer for transient data.
func GetTransientDataPullTimeout() time.Duration {
	timeout := viper.GetDuration(confTransientDataPullTimeout)
	if timeout == 0 {
		timeout = defaultTransientDataPullTimeout
	}
	return timeout
}

// GetBlockPublisherBufferSize returns the size of the block publisher channel buffer for various block events
func GetBlockPublisherBufferSize() int {
	size := viper.GetInt(confBlockPublisherBufferSize)
	if size == 0 {
		return defaultBlockPublisherBufferSize
	}
	return size
}

// GetOLCollMaxPeersForRetrieval returns the number of peers that should be messaged
// to retrieve collection data that is not stored locally.
func GetOLCollMaxPeersForRetrieval() int {
	maxPeers := viper.GetInt(confOLCollMaxPeersForRetrieval)
	if maxPeers <= 0 {
		maxPeers = defaultOLCollMaxPeersForRetrieval
	}
	return maxPeers
}

// GetOLCollCacheSize returns the size of the off-ledger cache
func GetOLCollCacheSize() int {
	size := viper.GetInt(confOLCollCacheSize)
	if size <= 0 {
		return defaultOLCollCacheSize
	}
	return size
}

// GetOLCollPullTimeout is the amount of time a peer waits for a response from another peer for transient data.
func GetOLCollPullTimeout() time.Duration {
	timeout := viper.GetDuration(confOLCollPullTimeout)
	if timeout == 0 {
		timeout = defaultOLCollPullTimeout
	}
	return timeout
}

// GetPeerRootPath returns the peer file system root path
func GetPeerRootPath() string {
	return filepath.Clean(config.GetPath(confPeerFileSystemPath))
}

// GetLedgerConfigRootPath returns the ledger management root path
func GetLedgerConfigRootPath() string {
	return filepath.Join(GetPeerRootPath(), confLedgerDataPath)
}

// GetPvtdataStorePath returns the private data store root path
func GetPvtdataStorePath() string {
	return filepath.Join(GetLedgerConfigRootPath(), confPvtdataStore)
}

// GetPvtdataStoreCollElgProcMaxDbBatchSize returns the maximum db batch size for converting
// the ineligible missing data entries to eligible missing data entries
// This method was referred from https://github.com/hyperledger/fabric/blob/master/internal/peer/node/config.go
func GetPvtdataStoreCollElgProcMaxDbBatchSize() int {
	collElgProcMaxDbBatchSize := 5000
	if viper.IsSet("ledger.pvtdataStore.collElgProcMaxDbBatchSize") {
		collElgProcMaxDbBatchSize = viper.GetInt("ledger.pvtdataStore.collElgProcMaxDbBatchSize")
	}
	return collElgProcMaxDbBatchSize
}

// GetPvtdataStoreCollElgProcDbBatchesInterval returns the minimum duration (in milliseconds) between writing
// two consecutive db batches for converting the ineligible missing data entries to eligible missing data entries
// This method was referred from https://github.com/hyperledger/fabric/blob/master/internal/peer/node/config.go
func GetPvtdataStoreCollElgProcDbBatchesInterval() int {
	collElgProcDbBatchesInterval := 1000
	if viper.IsSet("ledger.pvtdataStore.collElgProcDbBatchesInterval") {
		collElgProcDbBatchesInterval = viper.GetInt("ledger.pvtdataStore.collElgProcDbBatchesInterval")
	}
	return collElgProcDbBatchesInterval
}

// GetPvtdataStorePurgeInterval returns private data store purge interval
// This method was referred from https://github.com/hyperledger/fabric/blob/master/internal/peer/node/config.go
func GetPvtdataStorePurgeInterval() uint64 {
	purgeInterval := 100
	if viper.IsSet("ledger.pvtdataStore.purgeInterval") {
		purgeInterval = viper.GetInt("ledger.pvtdataStore.purgeInterval")
	}
	return uint64(purgeInterval)
}

// GetCouchDBConfig returns couch db config
// This method was referred from https://github.com/hyperledger/fabric/blob/master/internal/peer/node/config.go
func GetCouchDBConfig() *couchdb.Config {
	warmAfterNBlocks := 1
	if viper.IsSet("ledger.state.couchDBConfig.warmIndexesAfterNBlocks") {
		warmAfterNBlocks = viper.GetInt("ledger.state.couchDBConfig.warmIndexesAfterNBlocks")
	}
	internalQueryLimit := 1000
	if viper.IsSet("ledger.state.couchDBConfig.internalQueryLimit") {
		internalQueryLimit = viper.GetInt("ledger.state.couchDBConfig.internalQueryLimit")
	}
	maxBatchUpdateSize := 500
	if viper.IsSet("ledger.state.couchDBConfig.maxBatchUpdateSize") {
		maxBatchUpdateSize = viper.GetInt("ledger.state.couchDBConfig.maxBatchUpdateSize")
	}

	return &couchdb.Config{
		Address:                 viper.GetString("ledger.state.couchDBConfig.couchDBAddress"),
		Username:                viper.GetString("ledger.state.couchDBConfig.username"),
		Password:                viper.GetString("ledger.state.couchDBConfig.password"),
		MaxRetries:              viper.GetInt("ledger.state.couchDBConfig.maxRetries"),
		MaxRetriesOnStartup:     viper.GetInt("ledger.state.couchDBConfig.maxRetriesOnStartup"),
		RequestTimeout:          viper.GetDuration("ledger.state.couchDBConfig.requestTimeout"),
		InternalQueryLimit:      internalQueryLimit,
		MaxBatchUpdateSize:      maxBatchUpdateSize,
		WarmIndexesAfterNBlocks: warmAfterNBlocks,
		CreateGlobalChangesDB:   viper.GetBool("ledger.state.couchDBConfig.createGlobalChangesDB"),
		RedoLogPath:             filepath.Join(GetLedgerConfigRootPath(), "couchdbRedoLogs"),
	}
}
