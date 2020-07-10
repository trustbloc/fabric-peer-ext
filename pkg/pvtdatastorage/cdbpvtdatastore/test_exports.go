/*
Copyright IBM Corp, SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pvtdatastorage

import (
	"os"
	"testing"

	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/core/ledger"
	couchdb "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"
	"github.com/hyperledger/fabric/core/ledger/pvtdatapolicy"
	xstorageapi "github.com/hyperledger/fabric/extensions/storage/api"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/testutil"
)

// StoreEnv provides the  store env for testing
type StoreEnv struct {
	t                 testing.TB
	TestStoreProvider xstorageapi.PrivateDataProvider
	TestStore         xstorageapi.PrivateDataStore
	ledgerid          string
	btlPolicy         pvtdatapolicy.BTLPolicy
	couchDBConfig     *ledger.CouchDBConfig
}

// NewTestStoreEnv construct a StoreEnv for testing
func NewTestStoreEnv(t *testing.T, ledgerid string, btlPolicy pvtdatapolicy.BTLPolicy, couchDBConfig *ledger.CouchDBConfig) *StoreEnv {
	removeStorePath()
	req := require.New(t)
	conf := testutil.TestPrivateDataConf()
	testStoreProvider, err := NewProvider(conf, testutil.TestLedgerConf())
	req.NoError(err)
	testStore, err := testStoreProvider.OpenStore(ledgerid)
	req.NoError(err)
	testStore.Init(btlPolicy)
	s := &StoreEnv{t, testStoreProvider, testStore, ledgerid, btlPolicy, couchDBConfig}
	return s
}

// CloseAndReopen closes and opens the store provider
func (env *StoreEnv) CloseAndReopen() {
	var err error
	env.TestStoreProvider.Close()
	conf := testutil.TestPrivateDataConf()
	env.TestStoreProvider, err = NewProvider(conf, testutil.TestLedgerConf())
	require.NoError(env.t, err)
	env.TestStore, err = env.TestStoreProvider.OpenStore(env.ledgerid)
	env.TestStore.Init(env.btlPolicy)
	require.NoError(env.t, err)
}

//Cleanup env test
func (env *StoreEnv) Cleanup(ledgerid string) {
	//create a new connection
	couchInstance, err := couchdb.CreateCouchInstance(env.couchDBConfig, &disabled.Provider{})
	if err != nil {
		panic(err.Error())
	}
	pvtDataStoreDBName := couchdb.ConstructBlockchainDBName(ledgerid, pvtDataStoreName)
	db := couchdb.CouchDatabase{CouchInstance: couchInstance, DBName: pvtDataStoreDBName}
	//drop the test database
	if _, err := db.DropDatabase(); err != nil {
		panic(err.Error())
	}
	removeStorePath()
}

func removeStorePath() {
	dbPath := testutil.TestPrivateDataConf().StorePath
	if err := os.RemoveAll(dbPath); err != nil {
		panic(err.Error())
	}
}
