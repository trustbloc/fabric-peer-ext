/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	coreconfig "github.com/hyperledger/fabric/core/config"
	"github.com/hyperledger/fabric/core/ledger"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/core/ledger/util/couchdb"
	"github.com/hyperledger/fabric/integration/runner"
	"github.com/spf13/viper"
)

var logger = flogging.MustGetLogger("testutil")

//SetupExtTestEnv creates new couchdb instance for test
//returns couchdbd address, cleanup and stop function handle.
func SetupExtTestEnv() (addr string, cleanup func(string), stop func()) {
	externalCouch, set := os.LookupEnv("COUCHDB_ADDR")
	if set {
		return externalCouch, func(string) {}, func() {}
	}

	couchDB := &runner.CouchDB{}
	couchDB.Image = "couchdb:2.2.0"
	if err := couchDB.Start(); err != nil {
		panic(fmt.Errorf("failed to start couchDB: %s", err))
	}

	oldAddr := viper.GetString("ledger.state.couchDBConfig.couchDBAddress")

	//update config
	updateConfig(couchDB.Address())

	return couchDB.Address(),
		func(name string) {
			cleanupCouchDB(name)
		}, func() {
			//reset viper cdb config
			updateConfig(oldAddr)
			if err := couchDB.Stop(); err != nil {
				panic(err.Error())
			}
		}
}

func cleanupCouchDB(name string) {
	couchDBConfig := TestLedgerConf().StateDB.CouchDB
	couchInstance, _ := couchdb.CreateCouchInstance(couchDBConfig, &disabled.Provider{})

	blkdb := couchdb.CouchDatabase{CouchInstance: couchInstance, DBName: fmt.Sprintf("%s$$blocks_", name)}
	pvtdb := couchdb.CouchDatabase{CouchInstance: couchInstance, DBName: fmt.Sprintf("%s$$pvtdata_", name)}
	txndb := couchdb.CouchDatabase{CouchInstance: couchInstance, DBName: fmt.Sprintf("%s$$transactions_", name)}

	//drop the test databases
	_, err := blkdb.DropDatabase()
	if err != nil {
		logger.Warnf("Failed to drop db: %s, cause:%s", blkdb.DBName, err)
	}
	_, err = pvtdb.DropDatabase()
	if err != nil {
		logger.Warnf("Failed to drop db: %s, cause:%s", pvtdb.DBName, err)
	}
	_, err = txndb.DropDatabase()
	if err != nil {
		logger.Warnf("Failed to drop db: %s, cause:%s", txndb.DBName, err)
	}
}

//updateConfig updates 'couchAddress' in config
func updateConfig(couchAddress string) {

	viper.Set("ledger.state.stateDatabase", "CouchDB")
	viper.Set("ledger.state.couchDBConfig.couchDBAddress", couchAddress)
	// Replace with correct username/password such as
	// admin/admin if user security is enabled on couchdb.
	viper.Set("ledger.state.couchDBConfig.username", "")
	viper.Set("ledger.state.couchDBConfig.password", "")
	viper.Set("ledger.state.couchDBConfig.maxRetries", 1)
	viper.Set("ledger.state.couchDBConfig.maxRetriesOnStartup", 1)
	viper.Set("ledger.state.couchDBConfig.requestTimeout", time.Second*35)
	viper.Set("ledger.state.couchDBConfig.createGlobalChangesDB", false)

}

// TestLedgerConf return the ledger configs
func TestLedgerConf() *ledger.Config {
	// set defaults
	warmAfterNBlocks := 1
	if viper.IsSet("ledger.state.couchDBConfig.warmIndexesAfterNBlocks") {
		warmAfterNBlocks = viper.GetInt("ledger.state.couchDBConfig.warmIndexesAfterNBlocks")
	}
	internalQueryLimit := 1000
	if viper.IsSet("ledger.state.couchDBConfig.internalQueryLimit") {
		internalQueryLimit = viper.GetInt("ledger.state.couchDBConfig.internalQueryLimit")
	}
	maxBatchUpdateSize := 500
	if viper.IsSet("ledger.state.couchDBConfig.maxBatchUpdateSize") {
		maxBatchUpdateSize = viper.GetInt("ledger.state.couchDBConfig.maxBatchUpdateSize")
	}
	collElgProcMaxDbBatchSize := 5000
	if viper.IsSet("ledger.pvtdataStore.collElgProcMaxDbBatchSize") {
		collElgProcMaxDbBatchSize = viper.GetInt("ledger.pvtdataStore.collElgProcMaxDbBatchSize")
	}
	collElgProcDbBatchesInterval := 1000
	if viper.IsSet("ledger.pvtdataStore.collElgProcDbBatchesInterval") {
		collElgProcDbBatchesInterval = viper.GetInt("ledger.pvtdataStore.collElgProcDbBatchesInterval")
	}
	purgeInterval := 100
	if viper.IsSet("ledger.pvtdataStore.purgeInterval") {
		purgeInterval = viper.GetInt("ledger.pvtdataStore.purgeInterval")
	}

	rootFSPath := filepath.Join(coreconfig.GetPath("peer.fileSystemPath"), "ledgersData")
	conf := &ledger.Config{
		RootFSPath: rootFSPath,
		StateDB: &ledger.StateDB{
			StateDatabase: viper.GetString("ledger.state.stateDatabase"),
			LevelDBPath:   filepath.Join(rootFSPath, "stateLeveldb"),
			CouchDB:       &couchdb.Config{},
		},
		PrivateData: &ledger.PrivateData{
			StorePath:       filepath.Join(rootFSPath, "pvtdataStore"),
			MaxBatchSize:    collElgProcMaxDbBatchSize,
			BatchesInterval: collElgProcDbBatchesInterval,
			PurgeInterval:   purgeInterval,
		},
		HistoryDB: &ledger.HistoryDB{
			Enabled: viper.GetBool("ledger.history.enableHistoryDatabase"),
		},
	}

	conf.StateDB.CouchDB = &couchdb.Config{
		Address:                 viper.GetString("ledger.state.couchDBConfig.couchDBAddress"),
		Username:                viper.GetString("ledger.state.couchDBConfig.username"),
		Password:                viper.GetString("ledger.state.couchDBConfig.password"),
		MaxRetries:              viper.GetInt("ledger.state.couchDBConfig.maxRetries"),
		MaxRetriesOnStartup:     viper.GetInt("ledger.state.couchDBConfig.maxRetriesOnStartup"),
		RequestTimeout:          viper.GetDuration("ledger.state.couchDBConfig.requestTimeout"),
		InternalQueryLimit:      internalQueryLimit,
		MaxBatchUpdateSize:      maxBatchUpdateSize,
		WarmIndexesAfterNBlocks: warmAfterNBlocks,
		CreateGlobalChangesDB:   viper.GetBool("ledger.state.couchDBConfig.createGlobalChangesDB"),
		RedoLogPath:             filepath.Join(rootFSPath, "couchdbRedoLogs"),
	}

	return conf
}
