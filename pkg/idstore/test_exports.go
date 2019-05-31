/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idstore

import (
	"os"
	"testing"

	"github.com/trustbloc/fabric-peer-ext/pkg/testutil"

	"github.com/hyperledger/fabric/common/metrics/disabled"
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
	removeStorePath()
	testStore := OpenIDStore(ledgerid, testutil.TestLedgerConf())
	s := &StoreEnv{t, testStore, ledgerid, couchDBConfig}
	return s
}

//Cleanup env test
func (env *StoreEnv) Cleanup(ledgerid string) {
	//create a new connection
	couchInstance, err := couchdb.CreateCouchInstance(env.couchDBConfig, &disabled.Provider{})
	if err != nil {
		panic(err.Error())
	}
	pvtDataStoreDBName := couchdb.ConstructBlockchainDBName(systemID, inventoryName)
	db := couchdb.CouchDatabase{CouchInstance: couchInstance, DBName: pvtDataStoreDBName}
	//drop the test database
	if _, err := db.DropDatabase(); err != nil {
		panic(err.Error())
	}
	env.TestStore.Close()

	removeStorePath()
}

func removeStorePath() {
	dbPath := testutil.TestLedgerConf().PrivateData.StorePath
	if err := os.RemoveAll(dbPath); err != nil {
		panic(err.Error())
	}
}
