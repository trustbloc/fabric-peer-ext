/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"path/filepath"
	"time"

	"github.com/hyperledger/fabric/core/config"
	viper "github.com/spf13/viper2015"
)

const (
	confPeerFileSystemPath = "peer.fileSystemPath"
	confLedgerDataPath     = "ledgersData"

	confRoles            = "ledger.roles"
	confPvtDataCacheSize = "ledger.blockchain.pvtDataStorage.cacheSize"

	confTransientDataLeveldb             = "transientDataLeveldb"
	confTransientDataCleanupIntervalTime = "coll.transientdata.cleanupExpired.Interval"
	confTransientDataCacheSize           = "coll.transientdata.cacheSize"
	confTransientDataPullTimeout         = "peer.gossip.transientData.pullTimeout"

	confOLCollLeveldb              = "offLedgerLeveldb"
	confOLCollCleanupIntervalTime  = "coll.offledger.cleanupExpired.Interval"
	confOLCollMaxPeersForRetrieval = "coll.offledger.maxpeers"
	confOLCollMaxRetrievalAttempts = "coll.offledger.maxRetrievalAttempts"
	confOLCollCacheEnabled         = "coll.offledger.cache.enable"
	confOLCollCacheSize            = "coll.offledger.cache.size"
	confOLCollPullTimeout          = "coll.offledger.gossip.pullTimeout"

	confConfigUpdatePublisherBufferSize = "configpublisher.buffersize"

	defaultTransientDataCleanupIntervalTime = 5 * time.Second
	defaultTransientDataCacheSize           = 100000
	defaultTransientDataPullTimeout         = 5 * time.Second

	defaultOLCollCleanupIntervalTime  = 5 * time.Second
	defaultOLCollMaxPeersForRetrieval = 2
	defaultOLCollMaxRetrievalAttempts = 3
	defaultOLCollCacheSize            = 10000
	defaultOLCollPullTimeout          = 5 * time.Second

	defaultConfigUpdatePublisherBufferSize = 100

	// ConfBlockStoreDBType is the config key for the block store database type
	ConfBlockStoreDBType = "ledger.storage.blockStore.dbtype"
	// ConfIDStoreDBType is the config key for the ID store database type
	ConfIDStoreDBType = "ledger.storage.idStore.dbtype"
	// ConfPrivateDataStoreDBType is the config key for the private data store database type
	ConfPrivateDataStoreDBType = "ledger.storage.privateDataStore.dbtype"
	// ConfTransientStoreDBType is the config key for the transient store database type
	ConfTransientStoreDBType = "ledger.storage.transientStore.dbtype"

	defaultBlockStoreDBType       = CouchDBType
	defaultIDStoreDBType          = CouchDBType
	defaultPrivateDataStoreDBType = CouchDBType
	defaultTransientStoreDBType   = MemDBType
)

// DBType is the database type
type DBType = string

const (
	// CouchDBType indicates that the storage type is CouchDB
	CouchDBType DBType = "couchdb"
	// LevelDBType indicates that the storage type is LevelDB
	LevelDBType DBType = "leveldb"
	// MemDBType indicates that the storage type is in-memory
	MemDBType DBType = "memory"
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
	return filepath.Join(filepath.Clean(config.GetPath(confPeerFileSystemPath)), confTransientDataLeveldb)
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
	return filepath.Join(filepath.Join(filepath.Clean(config.GetPath(confPeerFileSystemPath)), confLedgerDataPath), confOLCollLeveldb)
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

// GetOLCollMaxPeersForRetrieval returns the number of peers that should be concurrently messaged
// to retrieve collection data that is not stored locally.
func GetOLCollMaxPeersForRetrieval() int {
	maxPeers := viper.GetInt(confOLCollMaxPeersForRetrieval)
	if maxPeers <= 0 {
		maxPeers = defaultOLCollMaxPeersForRetrieval
	}
	return maxPeers
}

// GetOLCollMaxRetrievalAttempts returns the maximum number of attempts to retrieve collection data from remote
// peers. On each attempt, multiple peers are messaged (up to a maximum number given by confOLCollMaxPeersForRetrieval).
// If not all data is retrieved on an attempt, then a new set of peers is chosen. This process continues
// until MaxRetrievalAttempts is reached or no more peers are left (that haven't already been attempted).
func GetOLCollMaxRetrievalAttempts() int {
	maxPeers := viper.GetInt(confOLCollMaxRetrievalAttempts)
	if maxPeers <= 0 {
		maxPeers = defaultOLCollMaxRetrievalAttempts
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

// GetOLCollCacheEnabled returns if off-ledger cache is enabled
func GetOLCollCacheEnabled() bool {
	enabled := viper.GetBool(confOLCollCacheEnabled)
	return enabled
}

// GetOLCollPullTimeout is the amount of time a peer waits for a response from another peer for transient data.
func GetOLCollPullTimeout() time.Duration {
	timeout := viper.GetDuration(confOLCollPullTimeout)
	if timeout == 0 {
		timeout = defaultOLCollPullTimeout
	}
	return timeout
}

// GetConfigUpdatePublisherBufferSize returns the size of the config update publisher channel buffer for ledger config update events
func GetConfigUpdatePublisherBufferSize() int {
	size := viper.GetInt(confConfigUpdatePublisherBufferSize)
	if size == 0 {
		return defaultConfigUpdatePublisherBufferSize
	}
	return size
}

// GetBlockStoreDBType returns the type of database that should be used for block storage
func GetBlockStoreDBType() DBType {
	dbType := viper.GetString(ConfBlockStoreDBType)
	if dbType == "" {
		return defaultBlockStoreDBType
	}

	return dbType
}

// GetIDStoreDBType returns the type of database that should be used for ID storage
func GetIDStoreDBType() DBType {
	dbType := viper.GetString(ConfIDStoreDBType)
	if dbType == "" {
		return defaultIDStoreDBType
	}

	return dbType
}

// GetPrivateDataStoreDBType returns the type of database that should be used for private data storage
func GetPrivateDataStoreDBType() DBType {
	dbType := viper.GetString(ConfPrivateDataStoreDBType)
	if dbType == "" {
		return defaultPrivateDataStoreDBType
	}

	return dbType
}

// GetTransientStoreDBType returns the type of database that should be used for private data transient storage
func GetTransientStoreDBType() DBType {
	dbType := viper.GetString(ConfTransientStoreDBType)
	if dbType == "" {
		return defaultTransientStoreDBType
	}

	return dbType
}
