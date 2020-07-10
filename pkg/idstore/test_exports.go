/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idstore

import (
	"testing"

	"github.com/hyperledger/fabric/core/ledger"
	"github.com/trustbloc/fabric-peer-ext/pkg/testutil"
)

// StoreEnv provides the  store env for testing
type StoreEnv struct {
	t             testing.TB
	TestStore     *Store
	ledgerid      string
	couchDBConfig *ledger.CouchDBConfig
}

// NewTestStoreEnv construct a StoreEnv for testing
func NewTestStoreEnv(t *testing.T, ledgerid string, couchDBConfig *ledger.CouchDBConfig) *StoreEnv {
	testStore, err := openIDStore(testutil.TestLedgerConf())
	if err != nil {
		panic(err.Error())
	}
	s := &StoreEnv{t, testStore, ledgerid, couchDBConfig}
	return s
}

var openIDStore = func(ledgerconfig *ledger.Config) (*Store, error) {
	return OpenIDStore(ledgerconfig)
}
