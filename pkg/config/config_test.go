/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
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
