/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvledger

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hyperledger/fabric/bccsp/sw"
	configtxtest "github.com/hyperledger/fabric/common/configtx/test"
	"github.com/hyperledger/fabric/common/metrics/disabled"
	lgr "github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger"
	couchdb "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"
	"github.com/hyperledger/fabric/core/ledger/mock"
	"github.com/stretchr/testify/require"
	xtestutil "github.com/trustbloc/fabric-peer-ext/pkg/testutil"
)

func TestRebuildDBs(t *testing.T) {
	conf, cleanup := testConfig(t)
	defer cleanup()
	provider := testutilNewProvider(conf, t, &mock.DeployedChaincodeInfoProvider{})

	numLedgers := 3
	for i := 0; i < numLedgers; i++ {
		genesisBlock, _ := configtxtest.MakeGenesisBlock(constructTestLedgerID(i))
		provider.Create(genesisBlock)
	}

	storeBlockDBCouchInstance, err := couchdb.CreateCouchInstance(conf.StateDBConfig.CouchDB, &disabled.Provider{})
	require.NoError(t, err)

	var blockStoreDBNames []string
	dbsName, _ := storeBlockDBCouchInstance.RetrieveApplicationDBNames()
	for _, dbName := range dbsName {
		if strings.Contains(dbName, "$$blocks_") || strings.Contains(dbName, "$$transactions_") {
			blockStoreDBNames = append(blockStoreDBNames, dbName)
		}
	}

	// rebuild should fail when provider is still open
	err = RebuildDBs(conf)
	require.Error(t, err, "as another peer node command is executing, wait for that command to complete its execution or terminate it before retrying")
	provider.Close()

	err = RebuildDBs(conf)
	require.NoError(t, err)

	// verify block store db is deleted
	dbsName, _ = storeBlockDBCouchInstance.RetrieveApplicationDBNames()
	for _, dbName := range dbsName {
		for _, blockStoreDBName := range blockStoreDBNames {
			require.NotEqual(t, dbName, blockStoreDBName)
		}
	}

	// verify configHistory, history, state, bookkeeper dbs are deleted
	rootFSPath := conf.RootFSPath
	_, err = os.Stat(ConfigHistoryDBPath(rootFSPath))
	require.Equal(t, os.IsNotExist(err), true)
	_, err = os.Stat(HistoryDBPath(rootFSPath))
	require.Equal(t, os.IsNotExist(err), true)
	_, err = os.Stat(StateDBPath(rootFSPath))
	require.Equal(t, os.IsNotExist(err), true)
	_, err = os.Stat(BookkeeperDBPath(rootFSPath))
	require.Equal(t, os.IsNotExist(err), true)

	// rebuild again should be successful
	err = RebuildDBs(conf)
	require.NoError(t, err)
}

func testutilNewProvider(conf *lgr.Config, t *testing.T, ccInfoProvider *mock.DeployedChaincodeInfoProvider) *kvledger.Provider {
	cryptoProvider, err := sw.NewDefaultSecurityLevelWithKeystore(sw.NewDummyKeyStore())
	require.NoError(t, err)

	conf.StateDBConfig.CouchDB = xtestutil.TestLedgerConf().StateDBConfig.CouchDB
	provider, err := kvledger.NewProvider(
		&lgr.Initializer{
			DeployedChaincodeInfoProvider: ccInfoProvider,
			MetricsProvider:               &disabled.Provider{},
			Config:                        conf,
			HashProvider:                  cryptoProvider,
		},
	)
	require.NoError(t, err, "Failed to create new Provider")
	return provider
}

func testConfig(t *testing.T) (conf *lgr.Config, cleanup func()) {
	path, err := ioutil.TempDir("", "kvledger")
	require.NoError(t, err, "Failed to create test ledger directory")
	//setup extension test environment
	_, _, destroy := xtestutil.SetupExtTestEnv()
	conf = &lgr.Config{
		RootFSPath: path,
		StateDBConfig: &lgr.StateDBConfig{
			CouchDB: xtestutil.TestLedgerConf().StateDBConfig.CouchDB,
		},
		PrivateDataConfig: &lgr.PrivateDataConfig{
			MaxBatchSize:    5000,
			BatchesInterval: 1000,
			PurgeInterval:   100,
		},
		HistoryDBConfig: &lgr.HistoryDBConfig{
			Enabled: true,
		},
		SnapshotsConfig: &lgr.SnapshotsConfig{
			RootDir: filepath.Join(path, "snapshots"),
		},
	}
	cleanup = func() {
		os.RemoveAll(path)
		destroy()
	}

	return conf, cleanup
}

func constructTestLedgerID(i int) string {
	return fmt.Sprintf("ledger_%06d", i)
}
