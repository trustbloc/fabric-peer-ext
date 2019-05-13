/*
Copyright IBM Corp, SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pvtdatastorage

import (
	"os"
	"testing"

	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/hyperledger/fabric/core/ledger/pvtdatapolicy"
	"github.com/hyperledger/fabric/core/ledger/pvtdatastorage"
	"github.com/hyperledger/fabric/core/ledger/util/couchdb"
	"github.com/stretchr/testify/require"
)

// StoreEnv provides the  store env for testing
type StoreEnv struct {
	t                 testing.TB
	TestStoreProvider pvtdatastorage.Provider
	TestStore         pvtdatastorage.Store
	ledgerid          string
	btlPolicy         pvtdatapolicy.BTLPolicy
	couchDBDef        *couchdb.CouchDBDef
}

// NewTestStoreEnv construct a StoreEnv for testing
func NewTestStoreEnv(t *testing.T, ledgerid string, btlPolicy pvtdatapolicy.BTLPolicy, couchDBDef *couchdb.CouchDBDef) *StoreEnv {
	removeStorePath()
	req := require.New(t)
	testStoreProvider := NewProvider()
	testStore, err := testStoreProvider.OpenStore(ledgerid)
	req.NoError(err)
	testStore.Init(btlPolicy)
	s := &StoreEnv{t, testStoreProvider, testStore, ledgerid, btlPolicy, couchDBDef}
	return s
}

// CloseAndReopen closes and opens the store provider
func (env *StoreEnv) CloseAndReopen() {
	var err error
	env.TestStoreProvider.Close()
	env.TestStoreProvider = NewProvider()
	env.TestStore, err = env.TestStoreProvider.OpenStore(env.ledgerid)
	env.TestStore.Init(env.btlPolicy)
	require.NoError(env.t, err)
}

//Cleanup env test
func (env *StoreEnv) Cleanup(ledgerid string) {
	//create a new connection
	couchInstance, err := couchdb.CreateCouchInstance(env.couchDBDef.URL, env.couchDBDef.Username, env.couchDBDef.Password,
		env.couchDBDef.MaxRetries, env.couchDBDef.MaxRetriesOnStartup, env.couchDBDef.RequestTimeout, env.couchDBDef.CreateGlobalChangesDB, &disabled.Provider{})
	if err != nil {
		panic(err.Error())
	}
	pvtDataStoreDBName := couchdb.ConstructBlockchainDBName(ledgerid, "pvtdata")
	db := couchdb.CouchDatabase{CouchInstance: couchInstance, DBName: pvtDataStoreDBName}
	//drop the test database
	if _, err := db.DropDatabase(); err != nil {
		panic(err.Error())
	}
	env.TestStore.Shutdown()

	removeStorePath()
}

func removeStorePath() {
	dbPath := ledgerconfig.GetPvtdataStorePath()
	if err := os.RemoveAll(dbPath); err != nil {
		panic(err.Error())
	}
}
