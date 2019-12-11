/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idstore

import (
	"testing"

	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/util/couchdb"
	"github.com/trustbloc/fabric-peer-ext/pkg/testutil"
)

// StoreEnv provides the  store env for testing
type StoreEnv struct {
	t             testing.TB
	TestStore     *Store
	ledgerid      string
	couchDBConfig *couchdb.Config
}

// NewTestStoreEnv construct a StoreEnv for testing
func NewTestStoreEnv(t *testing.T, ledgerid string, couchDBConfig *couchdb.Config) *StoreEnv {
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
