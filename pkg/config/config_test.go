/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"testing"
	"time"

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
