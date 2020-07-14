/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package testutil

import (
	"testing"

	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/pvtdatastorage"
	"github.com/trustbloc/fabric-peer-ext/pkg/testutil"
)

//SetupExtTestEnv creates new couchdb instance for test
//returns couchdbd address, cleanup and stop function handle.
func SetupExtTestEnv() (addr string, cleanup func(string), stop func()) {
	return testutil.SetupExtTestEnv()
}

// SetupResources sets up all of the mock resource providers
func SetupResources() func() {
	return testutil.SetupResources()
}

// GetExtStateDBProvider returns the implementation of the versionedDBProvider
func GetExtStateDBProvider(t testing.TB, dbProvider statedb.VersionedDBProvider) statedb.VersionedDBProvider {
	return nil
}

// TestLedgerConf return the ledger configs
func TestLedgerConf() *ledger.Config {
	return testutil.TestLedgerConf()
}

// TestPrivateDataConf return the private data configs
func TestPrivateDataConf() *pvtdatastorage.PrivateDataConfig {
	return testutil.TestPrivateDataConf()
}

// Skip skips the unit test for extensions
func Skip(t *testing.T, msg string) {
	t.Skip(msg)
}
