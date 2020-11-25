/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"path/filepath"
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/config"
	unixfs "github.com/ipfs/go-unixfs/importer/helpers"
	viper "github.com/spf13/viper2015"
)

var logger = flogging.MustGetLogger("ext_config")

const (
	confPeerFileSystemPath = "peer.fileSystemPath"
	confLedgerDataPath     = "ledgersData"

	confRoles            = "ledger.roles"
	confPvtDataCacheSize = "ledger.blockchain.pvtDataStorage.cacheSize"

	confTransientDataLeveldb             = "transientDataLeveldb"
	confTransientDataCleanupIntervalTime = "coll.transientdata.cleanupExpired.Interval"
	confTransientDataCacheSize           = "coll.transientdata.cacheSize"
	confTransientDataAlwaysPersist       = "coll.transientdata.alwaysPersist"
	confTransientDataPullTimeout         = "peer.gossip.transientData.pullTimeout"

	confOLCollLeveldb              = "offLedgerLeveldb"
	confOLCollCleanupIntervalTime  = "coll.offledger.cleanupExpired.Interval"
	confOLCollMaxPeersForRetrieval = "coll.offledger.maxpeers"
	confOLCollMaxRetrievalAttempts = "coll.offledger.maxRetrievalAttempts"
	confOLCollCacheSize            = "coll.offledger.cache.size"
	confOLCollPullTimeout          = "coll.offledger.gossip.pullTimeout"

	confDCASMaxLinksPerBlock = "coll.dcas.maxLinksPerBlock"
	confDCASRawLeaves        = "coll.dcas.rawLeaves"
	confDCASMaxBlockSize     = "coll.dcas.maxBlockSize"
	confDCASBlockLayout      = "coll.dcas.blockLayout"

	confConfigUpdatePublisherBufferSize = "configpublisher.buffersize"

	defaultTransientDataCleanupIntervalTime = 5 * time.Second
	defaultTransientDataCacheSize           = 100000
	defaultTransientDataPullTimeout         = 5 * time.Second
	defaultTransientDataAlwaysPersist       = true

	defaultOLCollCleanupIntervalTime  = 5 * time.Second
	defaultOLCollMaxPeersForRetrieval = 2
	defaultOLCollMaxRetrievalAttempts = 3
	defaultOLCollCacheSize            = 10000
	defaultOLCollPullTimeout          = 5 * time.Second

	defaultDCASrawLeaves          = true
	defaultDCASMaxBlockSize int64 = 1024 * 256

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

	confBlockStoreCacheSizeBlockByNum  = "ledger.storage.blockStore.cacheSize.blockByNum"
	confBlockStoreCacheSizeBlockByHash = "ledger.storage.blockStore.cacheSize.blockByHash"

	confSkipCheckForDupTxnID = "peer.skipCheckForDupTxnID"

	defaultBlockByNumCacheSize  = uint(20)
	defaultBlockByHashCacheSize = uint(20)
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

// GetTransientDataAlwaysPersist indicates whether transient data is to always be persisted or persisted only when the cache has exceeded its maximum size
func GetTransientDataAlwaysPersist() bool {
	if viper.IsSet(confTransientDataAlwaysPersist) {
		return viper.GetBool(confTransientDataAlwaysPersist)
	}

	return defaultTransientDataAlwaysPersist
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
	if !viper.IsSet(confOLCollCacheSize) {
		return defaultOLCollCacheSize
	}

	return viper.GetInt(confOLCollCacheSize)
}

// GetOLCollPullTimeout is the amount of time a peer waits for a response from another peer for transient data.
func GetOLCollPullTimeout() time.Duration {
	timeout := viper.GetDuration(confOLCollPullTimeout)
	if timeout == 0 {
		timeout = defaultOLCollPullTimeout
	}
	return timeout
}

// GetDCASMaxLinksPerBlock specifies the maximum number of links there will be per block in a Merkle DAG.
func GetDCASMaxLinksPerBlock() int {
	maxLinks := viper.GetInt(confDCASMaxLinksPerBlock)
	if maxLinks == 0 {
		return unixfs.DefaultLinksPerBlock
	}

	return maxLinks
}

// GetDCASMaxBlockSize specifies the maximum size of a block in a Merkle DAG.
func GetDCASMaxBlockSize() int64 {
	maxBlockSize := viper.GetInt(confDCASMaxBlockSize)
	if maxBlockSize == 0 {
		return defaultDCASMaxBlockSize
	}

	return int64(maxBlockSize)
}

// IsDCASRawLeaves indicates whether or not to use raw leaf nodes in a Merkle DAG.
func IsDCASRawLeaves() bool {
	if viper.IsSet(confDCASRawLeaves) {
		return viper.GetBool(confDCASRawLeaves)
	}

	return defaultDCASrawLeaves
}

// GetDCASBlockLayout returns the block layout strategy when creating a Merkle DAG.
// Supported values are "balanced" and "trickle". Leave empty to use the default.
func GetDCASBlockLayout() string {
	return viper.GetString(confDCASBlockLayout)
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

// GetBlockStoreBlockByNumCacheSize returns the size of the blocks-by-number cache.
// A value of zero disables the cache.
func GetBlockStoreBlockByNumCacheSize() uint {
	if !viper.IsSet(confBlockStoreCacheSizeBlockByNum) {
		return defaultBlockByNumCacheSize
	}

	size := viper.GetInt(confBlockStoreCacheSizeBlockByNum)
	if size < 0 {
		logger.Warnf("Invalid value for %s: %d. The value must be >= 0. Using default cache size %d",
			confBlockStoreCacheSizeBlockByNum, size, defaultBlockByNumCacheSize)

		return defaultBlockByNumCacheSize
	}

	return uint(size)
}

// GetBlockStoreBlockByHashCacheSize returns the size of the blocks-by-hash cache
// A value of zero disables the cache.
func GetBlockStoreBlockByHashCacheSize() uint {
	if !viper.IsSet(confBlockStoreCacheSizeBlockByHash) {
		return defaultBlockByHashCacheSize
	}

	size := viper.GetInt(confBlockStoreCacheSizeBlockByHash)
	if size < 0 {
		logger.Warnf("Invalid value for %s: %d. The value must be >= 0. Using default cache size %d",
			confBlockStoreCacheSizeBlockByHash, size, defaultBlockByHashCacheSize)

		return defaultBlockByHashCacheSize
	}

	return uint(size)
}

// IsSkipCheckForDupTxnID indicates whether or not endorsers should skip the check for duplicate transactions IDs. The check
// would still be performed during validation.
func IsSkipCheckForDupTxnID() bool {
	return viper.GetBool(confSkipCheckForDupTxnID)
}
