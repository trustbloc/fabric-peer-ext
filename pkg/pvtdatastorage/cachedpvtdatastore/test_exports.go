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
	TestStoreProvider *Provider
	TestStore         pvtdatastorage.Store
	ledgerid          string
	btlPolicy         pvtdatapolicy.BTLPolicy
}

// NewTestStoreEnv construct a StoreEnv for testing
func NewTestStoreEnv(t *testing.T, ledgerid string, btlPolicy pvtdatapolicy.BTLPolicy) *StoreEnv {
	testStoreProvider := NewProvider()
	testStore := testStoreProvider.Create(ledgerid, 0)
	testStore.Init(btlPolicy)
	s := &StoreEnv{t, testStoreProvider, testStore, ledgerid, btlPolicy}
	return s
}

// CloseAndReopen closes and opens the store Provider
func (env *StoreEnv) CloseAndReopen() {
	var err error
	env.TestStoreProvider = NewProvider()
	require.NoError(env.t, err)
	env.TestStore = env.TestStoreProvider.Create(env.ledgerid, 0)
	env.TestStore.Init(env.btlPolicy)
	require.NoError(env.t, err)
}
