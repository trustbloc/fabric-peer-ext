/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idstore

import (
	"testing"

	"github.com/trustbloc/fabric-peer-ext/pkg/testutil"

	"github.com/hyperledger/fabric/core/ledger/kvledger/idstore"
	"github.com/hyperledger/fabric/core/ledger/util/couchdb"
)

// StoreEnv provides the  store env for testing
type StoreEnv struct {
	t             testing.TB
	TestStore     idstore.IDStore
	ledgerid      string
	couchDBConfig *couchdb.Config
}

// NewTestStoreEnv construct a StoreEnv for testing
func NewTestStoreEnv(t *testing.T, ledgerid string, couchDBConfig *couchdb.Config) *StoreEnv {
	testStore, err := OpenIDStore(testutil.TestLedgerConf())
	if err != nil {
		panic(err.Error())
	}
	s := &StoreEnv{t, testStore, ledgerid, couchDBConfig}
	return s
}
