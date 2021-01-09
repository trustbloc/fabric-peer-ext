/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"testing"
	"time"

	unixfs "github.com/ipfs/go-unixfs/importer/helpers"
	viper "github.com/spf13/viper2015"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRoles(t *testing.T) {
	oldVal := viper.Get(confRoles)
	defer viper.Set(confRoles, oldVal)

	roles := "endorser,committer"
	viper.Set(confRoles, roles)
	assert.Equal(t, roles, GetRoles())
}

func TestGetPvtDataCacheSize(t *testing.T) {
	oldVal := viper.Get(confPvtDataCacheSize)
	defer viper.Set(confPvtDataCacheSize, oldVal)

	val := GetPvtDataCacheSize()
	assert.Equal(t, val, 10)

	viper.Set(confPvtDataCacheSize, 99)
	val = GetPvtDataCacheSize()
	assert.Equal(t, val, 99)

}

func TestGetTransientDataLevelDBPath(t *testing.T) {
	oldVal := viper.Get("peer.fileSystemPath")
	defer viper.Set("peer.fileSystemPath", oldVal)

	viper.Set("peer.fileSystemPath", "/tmp123")

	assert.Equal(t, "/tmp123/transientDataLeveldb", GetTransientDataLevelDBPath())
}

func TestGetTransientDataExpiredIntervalTime(t *testing.T) {
	oldVal := viper.Get(confTransientDataCleanupIntervalTime)
	defer viper.Set(confTransientDataCleanupIntervalTime, oldVal)

	viper.Set(confTransientDataCleanupIntervalTime, "")
	assert.Equal(t, defaultTransientDataCleanupIntervalTime, GetTransientDataExpiredIntervalTime())

	viper.Set(confTransientDataCleanupIntervalTime, 111*time.Second)
	assert.Equal(t, 111*time.Second, GetTransientDataExpiredIntervalTime())
}

func TestGetTransientDataCacheSize(t *testing.T) {
	oldVal := viper.Get(confTransientDataCacheSize)
	defer viper.Set(confTransientDataCacheSize, oldVal)

	viper.Set(confTransientDataCacheSize, 0)
	assert.Equal(t, defaultTransientDataCacheSize, GetTransientDataCacheSize())

	viper.Set(confTransientDataCacheSize, 10)
	assert.Equal(t, 10, GetTransientDataCacheSize())
}

func TestGetTransientDataAlwaysPersist(t *testing.T) {
	require.Equal(t, defaultTransientDataAlwaysPersist, GetTransientDataAlwaysPersist())

	viper.Set(confTransientDataAlwaysPersist, false)
	require.Equal(t, false, GetTransientDataAlwaysPersist())
}

func TestGetOLLevelDBPath(t *testing.T) {
	oldVal := viper.Get("peer.fileSystemPath")
	defer viper.Set("peer.fileSystemPath", oldVal)

	viper.Set("peer.fileSystemPath", "/tmp123")

	assert.Equal(t, "/tmp123/ledgersData/offLedgerLeveldb", GetOLCollLevelDBPath())
}

func TestGetOLCollExpiredIntervalTime(t *testing.T) {
	oldVal := viper.Get(confOLCollCleanupIntervalTime)
	defer viper.Set(confOLCollCleanupIntervalTime, oldVal)

	viper.Set(confOLCollCleanupIntervalTime, "")
	assert.Equal(t, defaultOLCollCleanupIntervalTime, GetOLCollExpirationCheckInterval())

	viper.Set(confOLCollCleanupIntervalTime, 111*time.Second)
	assert.Equal(t, 111*time.Second, GetOLCollExpirationCheckInterval())
}

func TestGetOLCacheSize(t *testing.T) {
	oldVal := viper.Get(confOLCollCacheSize)
	defer viper.Set(confOLCollCacheSize, oldVal)

	require.Equal(t, defaultOLCollCacheSize, GetOLCollCacheSize())

	viper.Set(confOLCollCacheSize, 0)
	require.Equal(t, 0, GetOLCollCacheSize())

	viper.Set(confOLCollCacheSize, 10)
	assert.Equal(t, 10, GetOLCollCacheSize())
}

func TestGetTransientDataPullTimeout(t *testing.T) {
	oldVal := viper.Get(confTransientDataPullTimeout)
	defer viper.Set(confTransientDataPullTimeout, oldVal)

	viper.Set(confTransientDataPullTimeout, "")
	assert.Equal(t, defaultTransientDataPullTimeout, GetTransientDataPullTimeout())

	viper.Set(confTransientDataPullTimeout, 111*time.Second)
	assert.Equal(t, 111*time.Second, GetTransientDataPullTimeout())
}

func TestGetOLCollMaxPeersForRetrieval(t *testing.T) {
	oldVal := viper.Get(confOLCollMaxPeersForRetrieval)
	defer viper.Set(confOLCollMaxPeersForRetrieval, oldVal)

	viper.Set(confOLCollMaxPeersForRetrieval, "")
	assert.Equal(t, defaultOLCollMaxPeersForRetrieval, GetOLCollMaxPeersForRetrieval())

	viper.Set(confOLCollMaxPeersForRetrieval, 7)
	assert.Equal(t, 7, GetOLCollMaxPeersForRetrieval())
}

func TestGetOLCollMaxRetrievalAttempts(t *testing.T) {
	oldVal := viper.Get(confOLCollMaxRetrievalAttempts)
	defer viper.Set(confOLCollMaxRetrievalAttempts, oldVal)

	viper.Set(confOLCollMaxRetrievalAttempts, "")
	assert.Equal(t, defaultOLCollMaxRetrievalAttempts, GetOLCollMaxRetrievalAttempts())

	viper.Set(confOLCollMaxRetrievalAttempts, 7)
	assert.Equal(t, 7, GetOLCollMaxRetrievalAttempts())
}

func TestGetOLCollPullTimeout(t *testing.T) {
	oldVal := viper.Get(confOLCollPullTimeout)
	defer viper.Set(confOLCollPullTimeout, oldVal)

	viper.Set(confOLCollPullTimeout, "")
	assert.Equal(t, defaultOLCollPullTimeout, GetOLCollPullTimeout())

	viper.Set(confOLCollPullTimeout, 111*time.Second)
	assert.Equal(t, 111*time.Second, GetOLCollPullTimeout())
}

func TestGetConfigUpdatePublisherBufferSize(t *testing.T) {
	oldVal := viper.Get(confConfigUpdatePublisherBufferSize)
	defer viper.Set(confConfigUpdatePublisherBufferSize, oldVal)

	viper.Set(confConfigUpdatePublisherBufferSize, "")
	assert.Equal(t, defaultConfigUpdatePublisherBufferSize, GetConfigUpdatePublisherBufferSize())

	viper.Set(confConfigUpdatePublisherBufferSize, 1234)
	assert.Equal(t, 1234, GetConfigUpdatePublisherBufferSize())
}

func TestGetBlockStoreDBType(t *testing.T) {
	oldVal := viper.Get(ConfBlockStoreDBType)
	defer viper.Set(ConfBlockStoreDBType, oldVal)

	viper.Set(ConfBlockStoreDBType, "")
	require.Equal(t, defaultBlockStoreDBType, GetBlockStoreDBType())

	viper.Set(ConfBlockStoreDBType, LevelDBType)
	require.Equal(t, LevelDBType, GetBlockStoreDBType())
}

func TestGetIDStoreDBType(t *testing.T) {
	oldVal := viper.Get(ConfIDStoreDBType)
	defer viper.Set(ConfIDStoreDBType, oldVal)

	viper.Set(ConfIDStoreDBType, "")
	require.Equal(t, defaultIDStoreDBType, GetIDStoreDBType())

	viper.Set(ConfIDStoreDBType, LevelDBType)
	require.Equal(t, LevelDBType, GetIDStoreDBType())
}

func TestGetPrivateDataStoreDBType(t *testing.T) {
	oldVal := viper.Get(ConfPrivateDataStoreDBType)
	defer viper.Set(ConfPrivateDataStoreDBType, oldVal)

	viper.Set(ConfPrivateDataStoreDBType, "")
	require.Equal(t, defaultPrivateDataStoreDBType, GetPrivateDataStoreDBType())

	viper.Set(ConfPrivateDataStoreDBType, LevelDBType)
	require.Equal(t, LevelDBType, GetPrivateDataStoreDBType())
}

func TestGetTransientStoreDBType(t *testing.T) {
	oldVal := viper.Get(ConfTransientStoreDBType)
	defer viper.Set(ConfTransientStoreDBType, oldVal)

	viper.Set(ConfTransientStoreDBType, "")
	require.Equal(t, defaultTransientStoreDBType, GetTransientStoreDBType())

	viper.Set(ConfTransientStoreDBType, LevelDBType)
	require.Equal(t, LevelDBType, GetTransientStoreDBType())
}

func TestGetDCASBlockLayout(t *testing.T) {
	oldVal := viper.Get(confDCASBlockLayout)
	defer viper.Set(confDCASBlockLayout, oldVal)

	viper.Set(confDCASBlockLayout, "trickle")
	require.Equal(t, "trickle", GetDCASBlockLayout())
}

func TestGetDCASMaxBlockSize(t *testing.T) {
	oldVal := viper.Get(confDCASMaxBlockSize)
	defer viper.Set(confDCASMaxBlockSize, oldVal)

	viper.Set(confDCASMaxBlockSize, "")
	require.Equal(t, defaultDCASMaxBlockSize, GetDCASMaxBlockSize())

	viper.Set(confDCASMaxBlockSize, 456)
	require.Equal(t, int64(456), GetDCASMaxBlockSize())
}

func TestGetDCASMaxLinksPerBlock(t *testing.T) {
	oldVal := viper.Get(confDCASMaxLinksPerBlock)
	defer viper.Set(confDCASMaxLinksPerBlock, oldVal)

	viper.Set(confDCASMaxLinksPerBlock, "")
	require.Equal(t, unixfs.DefaultLinksPerBlock, GetDCASMaxLinksPerBlock())

	viper.Set(confDCASMaxLinksPerBlock, 323)
	require.Equal(t, 323, GetDCASMaxLinksPerBlock())
}

func TestIsDCASRawLeaves(t *testing.T) {
	oldVal := viper.Get(confDCASRawLeaves)
	defer viper.Set(confDCASRawLeaves, oldVal)

	require.Equal(t, defaultDCASrawLeaves, IsDCASRawLeaves())

	viper.Set(confDCASRawLeaves, false)
	require.False(t, IsDCASRawLeaves())
}

func TestGetBlockStoreBlockByNumCacheSize(t *testing.T) {
	oldVal := viper.Get(confBlockStoreCacheSizeBlockByNum)
	defer viper.Set(confBlockStoreCacheSizeBlockByNum, oldVal)

	require.Equal(t, defaultBlockByNumCacheSize, GetBlockStoreBlockByNumCacheSize())

	viper.Set(confBlockStoreCacheSizeBlockByNum, 300)
	require.Equal(t, uint(300), GetBlockStoreBlockByNumCacheSize())

	viper.Set(confBlockStoreCacheSizeBlockByNum, -1)
	require.Equal(t, defaultBlockByNumCacheSize, GetBlockStoreBlockByNumCacheSize())
}

func TestGetBlockStoreBlockByHashCacheSize(t *testing.T) {
	oldVal := viper.Get(confBlockStoreCacheSizeBlockByHash)
	defer viper.Set(confBlockStoreCacheSizeBlockByHash, oldVal)

	require.Equal(t, defaultBlockByHashCacheSize, GetBlockStoreBlockByHashCacheSize())

	viper.Set(confBlockStoreCacheSizeBlockByHash, 300)
	require.Equal(t, uint(300), GetBlockStoreBlockByHashCacheSize())

	viper.Set(confBlockStoreCacheSizeBlockByHash, -1)
	require.Equal(t, defaultBlockByHashCacheSize, GetBlockStoreBlockByHashCacheSize())
}

func TestIsSkipCheckForDupTxnID(t *testing.T) {
	oldVal := viper.Get(confSkipCheckForDupTxnID)
	defer viper.Set(confSkipCheckForDupTxnID, oldVal)

	require.False(t, IsSkipCheckForDupTxnID())

	viper.Set(confSkipCheckForDupTxnID, true)
	require.True(t, IsSkipCheckForDupTxnID())
}

func TestIsPrePopulateStateCache(t *testing.T) {
	oldVal := viper.Get(confStateCachePrePopulate)
	defer viper.Set(confStateCachePrePopulate, oldVal)

	require.False(t, IsPrePopulateStateCache())

	viper.Set(confStateCachePrePopulate, true)
	require.True(t, IsPrePopulateStateCache())
}

func TestGetStateCacheGossipTimeout(t *testing.T) {
	oldVal := viper.Get(confStateCacheGossipTimeout)
	defer viper.Set(confStateCacheGossipTimeout, oldVal)

	require.Equal(t, defaultStateCacheGossipTimeout, GetStateCacheGossipTimeout())

	viper.Set(confStateCacheGossipTimeout, 7*time.Second)
	require.Equal(t, 7*time.Second, GetStateCacheGossipTimeout())
}

func TestGetStateCacheRetentionSize(t *testing.T) {
	oldVal := viper.Get(confStateCacheRetentionSize)
	defer viper.Set(confStateCacheRetentionSize, oldVal)

	require.Equal(t, defaultStateCacheRetentionSize, GetStateCacheRetentionSize())

	viper.Set(confStateCacheRetentionSize, 55)
	require.Equal(t, 55, GetStateCacheRetentionSize())
}

func TestGetValidationSinglePeerTransactionThreshold(t *testing.T) {
	oldVal := viper.Get(ConfValidationSinglePeerTransactionThreshold)
	defer viper.Set(ConfValidationSinglePeerTransactionThreshold, oldVal)

	require.Equal(t, defaultValidationSinglePeerTransactionThreshold, GetValidationSinglePeerTransactionThreshold())

	viper.Set(ConfValidationSinglePeerTransactionThreshold, 23)
	require.Equal(t, 23, GetValidationSinglePeerTransactionThreshold())
}

func TestGetValidationCommitterTransactionThreshold(t *testing.T) {
	oldVal := viper.Get(ConfValidationCommitterTransactionThreshold)
	defer viper.Set(ConfValidationCommitterTransactionThreshold, oldVal)

	require.Equal(t, defaultValidationCommitterTransactionThreshold, GetValidationCommitterTransactionThreshold())

	viper.Set(ConfValidationCommitterTransactionThreshold, 13)
	require.Equal(t, 13, GetValidationCommitterTransactionThreshold())
}
