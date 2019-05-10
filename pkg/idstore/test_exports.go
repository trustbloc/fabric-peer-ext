/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idstore

import (
	"os"
	"testing"

	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/core/ledger/kvledger/idstore"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/hyperledger/fabric/core/ledger/util/couchdb"
)

// StoreEnv provides the  store env for testing
type StoreEnv struct {
	t          testing.TB
	TestStore  idstore.IDStore
	ledgerid   string
	couchDBDef *couchdb.CouchDBDef
}

// NewTestStoreEnv construct a StoreEnv for testing
func NewTestStoreEnv(t *testing.T, ledgerid string, couchDBDef *couchdb.CouchDBDef) *StoreEnv {
	removeStorePath()
	testStore := OpenIDStore(ledgerid)
	s := &StoreEnv{t, testStore, ledgerid, couchDBDef}
	return s
}

//Cleanup env test
func (env *StoreEnv) Cleanup(ledgerid string) {
	//create a new connection
	couchInstance, err := couchdb.CreateCouchInstance(env.couchDBDef.URL, env.couchDBDef.Username, env.couchDBDef.Password,
		env.couchDBDef.MaxRetries, env.couchDBDef.MaxRetriesOnStartup, env.couchDBDef.RequestTimeout, env.couchDBDef.CreateGlobalChangesDB, &disabled.Provider{})
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
	dbPath := ledgerconfig.GetPvtdataStorePath()
	if err := os.RemoveAll(dbPath); err != nil {
		panic(err.Error())
	}
}
