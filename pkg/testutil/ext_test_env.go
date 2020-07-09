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

	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/metrics/disabled"
	coreconfig "github.com/hyperledger/fabric/core/config"
	"github.com/hyperledger/fabric/core/ledger"
	couchdb "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"
	"github.com/hyperledger/fabric/core/ledger/pvtdatastorage"
	"github.com/hyperledger/fabric/integration/runner"
	viper "github.com/spf13/viper2015"

	clientmocks "github.com/trustbloc/fabric-peer-ext/pkg/collections/client/mocks"
	olretriever "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/mocks"
	storemocks "github.com/trustbloc/fabric-peer-ext/pkg/collections/storeprovider/mocks"
	tdretriever "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/gossip/appdata"
	statemocks "github.com/trustbloc/fabric-peer-ext/pkg/gossip/state/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/resource"
)

var logger = flogging.MustGetLogger("testutil")

//SetupExtTestEnv creates new couchdb instance for test
//returns couchdb address, cleanup and stop function handle.
func SetupExtTestEnv() (addr string, cleanup func(string), stop func()) {
	externalCouch, set := os.LookupEnv("COUCHDB_ADDR")
	if set {
		return externalCouch, func(string) {}, func() {}
	}

	couchDB := &runner.CouchDB{}

	if err := couchDB.Start(); err != nil {
		panic(fmt.Errorf("failed to start couchDB: %s", err))
	}

	oldAddr := viper.GetString("ledger.state.couchDBConfig.couchDBAddress")

	//update config
	updateConfig(couchDB.Address())

	clearResources := SetupResources()

	return couchDB.Address(),
		func(name string) {
			cleanupCouchDB(name)
		}, func() {
			//reset viper cdb config
			updateConfig(oldAddr)
			if err := couchDB.Stop(); err != nil {
				panic(err.Error())
			}
			clearResources()
		}
}

// SetupResources sets up all of the mock resource providers
func SetupResources() func() {
	logger.Info("... registering all resources...")

	// Register all of the dependent (mock) resources
	resource.Register(func() *mocks.CollectionConfigProvider { return &mocks.CollectionConfigProvider{} })
	resource.Register(func() *storemocks.TransientDataStoreProvider { return storemocks.NewTransientDataStoreProvider() })
	resource.Register(func() *storemocks.StoreProvider { return storemocks.NewOffLedgerStoreProvider() })
	resource.Register(func() *tdretriever.TransientDataProvider { return &tdretriever.TransientDataProvider{} })
	resource.Register(func() *olretriever.Provider { return &olretriever.Provider{} })
	resource.Register(appdata.NewHandlerRegistry)

	l := &mocks.Ledger{BlockchainInfo: &common.BlockchainInfo{Height: 1000}}
	ledgerProvider := &mocks.LedgerProvider{}
	ledgerProvider.GetLedgerReturns(l)

	if err := resource.Mgr.Initialize(
		mocks.NewBlockPublisherProvider(),
		ledgerProvider,
		&mocks.GossipProvider{},
		&clientmocks.PvtDataDistributor{},
		&mocks.IdentityDeserializerProvider{},
		&mocks.IdentifierProvider{},
		&mocks.IdentityProvider{},
		&statemocks.CCEventMgrProvider{},
	); err != nil {
		panic(err)
	}

	return func() {
		resource.Mgr.Clear()
	}
}

func cleanupCouchDB(name string) {
	couchDBConfig := TestLedgerConf().StateDBConfig.CouchDB
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
	viper.Set("ledger.state.couchDBConfig.username", "admin")
	viper.Set("ledger.state.couchDBConfig.password", "adminpw")
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

	rootFSPath := filepath.Join(coreconfig.GetPath("peer.fileSystemPath"), "ledgersData")
	conf := &ledger.Config{
		RootFSPath: rootFSPath,
		StateDBConfig: &ledger.StateDBConfig{
			StateDatabase: viper.GetString("ledger.state.stateDatabase"),
			//LevelDBPath:   filepath.Join(rootFSPath, "stateLeveldb"),
			CouchDB: &ledger.CouchDBConfig{},
		},
		PrivateDataConfig: &ledger.PrivateDataConfig{},
		HistoryDBConfig: &ledger.HistoryDBConfig{
			Enabled: viper.GetBool("ledger.history.enableHistoryDatabase"),
		},
	}

	conf.StateDBConfig.CouchDB = &ledger.CouchDBConfig{
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

// TestPrivateDataConf returns mock private data config for unit tests
func TestPrivateDataConf() *pvtdatastorage.PrivateDataConfig {
	rootFSPath := filepath.Join(coreconfig.GetPath("peer.fileSystemPath"), "ledgersData")
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

	return &pvtdatastorage.PrivateDataConfig{
		PrivateDataConfig: &ledger.PrivateDataConfig{
			MaxBatchSize:    collElgProcMaxDbBatchSize,
			BatchesInterval: collElgProcDbBatchesInterval,
			PurgeInterval:   purgeInterval,
		},
		StorePath: filepath.Join(rootFSPath, "pvtdatastorage"),
	}
}
