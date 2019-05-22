/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package testutil

import (
	"testing"

	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/trustbloc/fabric-peer-ext/pkg/testutil"
)

//SetupExtTestEnv creates new couchdb instance for test
//returns couchdbd address, cleanup and stop function handle.
func SetupExtTestEnv() (addr string, cleanup func(string), stop func()) {
	return testutil.SetupExtTestEnv()
}

// GetExtStateDBProvider returns the implementation of the versionedDBProvider
func GetExtStateDBProvider(t testing.TB, dbProvider statedb.VersionedDBProvider) statedb.VersionedDBProvider {
	return nil
}
