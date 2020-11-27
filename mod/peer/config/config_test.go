/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"testing"

	viper "github.com/spf13/viper2015"
	"github.com/stretchr/testify/require"
)

func TestIsSkipCheckForDupTxnID(t *testing.T) {
	require.False(t, IsSkipCheckForDupTxnID())

	viper.Set("peer.skipCheckForDupTxnID", true)
	require.True(t, IsSkipCheckForDupTxnID())
}

func TestIsPrePopulateStateCache(t *testing.T) {
	viper.Set("ledger.state.dbConfig.cache.prePopulate", false)
	require.False(t, IsPrePopulateStateCache())

	viper.Set("ledger.state.dbConfig.cache.prePopulate", true)
	require.True(t, IsPrePopulateStateCache())
}

func TestIsSaveCacheUpdates(t *testing.T) {
	viper.Set("ledger.state.dbConfig.cache.prePopulate", false)
	require.False(t, IsSaveCacheUpdates())

	viper.Set("ledger.state.dbConfig.cache.prePopulate", true)
	require.True(t, IsSaveCacheUpdates())
}
