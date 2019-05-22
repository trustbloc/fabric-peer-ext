/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package cachestore

import (
	"testing"

	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/config"
)

func TestStateCacheStore(t *testing.T) {
	stateCache := NewStateCacheStore("test", config.GetStateDataCacheSize())

	val1 := "ns1+key1+val"
	blockNum1 := uint64(101)
	txNum1 := uint64(53129426795)

	// add state cache data
	stateCache.Add("ns1", "key1", &statedb.VersionedValue{Value: []byte("ns1+key1+val"), Version: &version.Height{
		blockNum1,
		txNum1,
	}})

	// validate state cache store functions
	require.Equal(t, val1, string(stateCache.Get("ns1", "key1").Value))
	require.Equal(t, blockNum1, stateCache.GetVersion("ns1", "key1").BlockNum)
	require.Equal(t, txNum1, stateCache.GetVersion("ns1", "key1").TxNum)

	// remove state cache
	stateCache.Remove("ns1", "key1")

	// validate state cache store after removing the key
	require.Empty(t, stateCache.Get("ns1", "key1"))
	require.Empty(t, stateCache.GetVersion("ns1", "key1"))

}
