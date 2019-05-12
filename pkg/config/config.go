/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import "github.com/spf13/viper"

const confRoles = "ledger.roles"
const confPvtDataCacheSize = "ledger.blockchain.pvtDataStorage.cacheSize"

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
