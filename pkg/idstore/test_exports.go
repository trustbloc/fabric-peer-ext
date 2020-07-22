// +build testing

/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idstore

import (
	"testing"

	"github.com/hyperledger/fabric/core/ledger"
	couchdb "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"
	"github.com/pkg/errors"

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

// SaveMetadataDoc save metadata
func SaveMetadataDoc(testStore *Store, format string) error {
	jsonBytes, e := createMetadataDoc("", format)
	if e != nil {
		return e
	}

	_, e = testStore.db.SaveDoc(metadataID, "", &couchdb.CouchDoc{JSONValue: jsonBytes})
	if e != nil {
		return errors.WithMessage(e, "update of metadata in CouchDB failed")
	}

	return nil
}

var openIDStore = func(ledgerconfig *ledger.Config) (*Store, error) {
	return OpenIDStore(ledgerconfig)
}
