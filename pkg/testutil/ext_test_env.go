/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package testutil

import (
	"fmt"
	"os"
	"time"

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
			cleanupCouchDB(name, couchDB)
		}, func() {
			//reset viper cdb config
			updateConfig(oldAddr)
			if err := couchDB.Stop(); err != nil {
				panic(err.Error())
			}
		}
}

func cleanupCouchDB(name string, couchDB *runner.CouchDB) {
	couchDBDef := couchdb.GetCouchDBDefinition()
	couchInstance, _ := couchdb.CreateCouchInstance(couchDB.Address(), couchDBDef.Username, couchDBDef.Password,
		couchDBDef.MaxRetries, couchDBDef.MaxRetriesOnStartup, couchDBDef.RequestTimeout, couchDBDef.CreateGlobalChangesDB, &disabled.Provider{})

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
