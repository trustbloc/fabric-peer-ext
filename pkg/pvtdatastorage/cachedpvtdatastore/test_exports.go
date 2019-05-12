/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cachedpvtdatastore

import (
	"testing"

	"github.com/hyperledger/fabric/core/ledger/pvtdatapolicy"
	"github.com/hyperledger/fabric/core/ledger/pvtdatastorage"
	"github.com/stretchr/testify/require"
)

// StoreEnv provides the  store env for testing
type StoreEnv struct {
	t                 testing.TB
	TestStoreProvider pvtdatastorage.Provider
	TestStore         pvtdatastorage.Store
	ledgerid          string
	btlPolicy         pvtdatapolicy.BTLPolicy
}

// NewTestStoreEnv construct a StoreEnv for testing
func NewTestStoreEnv(t *testing.T, ledgerid string, btlPolicy pvtdatapolicy.BTLPolicy) *StoreEnv {
	req := require.New(t)
	testStoreProvider := NewProvider()
	testStore, err := testStoreProvider.OpenStore(ledgerid)
	req.NoError(err)
	testStore.Init(btlPolicy)
	s := &StoreEnv{t, testStoreProvider, testStore, ledgerid, btlPolicy}
	return s
}

// CloseAndReopen closes and opens the store provider
func (env *StoreEnv) CloseAndReopen() {
	var err error
	env.TestStoreProvider.Close()
	env.TestStoreProvider = NewProvider()
	require.NoError(env.t, err)
	env.TestStore, err = env.TestStoreProvider.OpenStore(env.ledgerid)
	env.TestStore.Init(env.btlPolicy)
	require.NoError(env.t, err)
}
