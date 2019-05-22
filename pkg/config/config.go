/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"path/filepath"
	"time"

	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/spf13/viper"
)

const (
	confRoles            = "ledger.roles"
	confPvtDataCacheSize = "ledger.blockchain.pvtDataStorage.cacheSize"

	confTransientDataLeveldb             = "transientDataLeveldb"
	confTransientDataCleanupIntervalTime = "coll.transientdata.cleanupExpired.Interval"
	confTransientDataCacheSize           = "coll.transientdata.cacheSize"

	confOLCollLeveldb             = "offLedgerLeveldb"
	confOLCollCleanupIntervalTime = "coll.offledger.cleanupExpired.Interval"

	defaultTransientDataCleanupIntervalTime = 5 * time.Second
	defaultTransientDataCacheSize           = 100000
	defaultOLCollCleanupIntervalTime        = 5 * time.Second

	// state cache
	stateDataCacheSize        = "ledger.state.cache.size"
	defaultStateDataCacheSize = 100
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
	return filepath.Join(ledgerconfig.GetRootPath(), confTransientDataLeveldb)
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
	return filepath.Join(ledgerconfig.GetRootPath(), confOLCollLeveldb)
}

// GetOLCollExpirationCheckInterval is time when background routine check expired collection data in db to cleanup.
func GetOLCollExpirationCheckInterval() time.Duration {
	timeout := viper.GetDuration(confOLCollCleanupIntervalTime)
	if timeout == 0 {
		return defaultOLCollCleanupIntervalTime
	}
	return timeout
}

// GetStateDataCacheSize returns the state data cache size value. If not set, the default value is 100.
func GetStateDataCacheSize() int {
	cap := viper.GetInt(stateDataCacheSize)
	if cap <= 0 {
		cap = defaultStateDataCacheSize
	}
	return cap
}
