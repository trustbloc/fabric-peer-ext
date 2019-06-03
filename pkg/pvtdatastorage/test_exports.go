/*
Copyright IBM Corp, SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pvtdatastorage

import (
	"os"
	"testing"

	"github.com/trustbloc/fabric-peer-ext/pkg/testutil"

	"github.com/hyperledger/fabric/core/ledger"

	"github.com/hyperledger/fabric/common/metrics/disabled"
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
	couchDBConfig     *couchdb.Config
}

// NewTestStoreEnv construct a StoreEnv for testing
func NewTestStoreEnv(t *testing.T, ledgerid string, btlPolicy pvtdatapolicy.BTLPolicy, couchDBConfig *couchdb.Config) *StoreEnv {
	removeStorePath()
	req := require.New(t)
	conf := testutil.TestLedgerConf().PrivateData
	testStoreProvider := NewProvider(conf, testutil.TestLedgerConf())
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
	conf := &ledger.PrivateData{
		StorePath:     testutil.TestLedgerConf().PrivateData.StorePath,
		PurgeInterval: 1,
	}
	env.TestStoreProvider = NewProvider(conf, testutil.TestLedgerConf())
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
	dbPath := testutil.TestLedgerConf().PrivateData.StorePath
	if err := os.RemoveAll(dbPath); err != nil {
		panic(err.Error())
	}
}
