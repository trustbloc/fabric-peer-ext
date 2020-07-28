/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvledger

import (
	"os"
	"strings"

	"github.com/pkg/errors"

	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/core/ledger"
	couchdb "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"
)

const (
	blockStorePostFix   = "$$blocks_"
	txnStorePostFix     = "$$transactions_"
	pvtDataStorePostFix = "$$pvtdata_"
	idStore             = "fabric_system_$$inventory_"
	internalIDStore     = "fabric__internal"
)

func dropDBs(ledgerconfig *ledger.Config) error {
	// During block commits to stateDB, the transaction manager updates the bookkeeperDB and one of the
	// state listener updates the config historyDB. As we drop the stateDB, we need to drop the
	// configHistoryDB and bookkeeperDB too so that during the peer startup after the reset/rollback,
	// we can get a correct configHistoryDB.
	// Note that it is necessary to drop the stateDB first before dropping the config history and
	// bookkeeper. Suppose if the config or bookkeeper is dropped first and the peer reset/rollback
	// command fails before dropping the stateDB, peer cannot start with consistent data (if the
	// user decides to start the peer without retrying the reset/rollback) as the stateDB would
	// not be rebuilt.

	if ledgerconfig.StateDBConfig.StateDatabase == "CouchDB" {
		if err := dropApplicationDBs(ledgerconfig.StateDBConfig.CouchDB); err != nil {
			return err
		}
	}

	if err := dropStateLevelDB(ledgerconfig.RootFSPath); err != nil {
		return err
	}
	if err := dropConfigHistoryDB(ledgerconfig.RootFSPath); err != nil {
		return err
	}
	if err := dropBookkeeperDB(ledgerconfig.RootFSPath); err != nil {
		return err
	}
	if err := dropHistoryDB(ledgerconfig.RootFSPath); err != nil {
		return err
	}
	return nil
}

// dropApplicationDBs drops all application databases.
func dropApplicationDBs(config *ledger.CouchDBConfig) error {
	couchInstance, err := couchdb.CreateCouchInstance(config, &disabled.Provider{})
	if err != nil {
		return err
	}
	dbNames, err := couchInstance.RetrieveApplicationDBNames()
	if err != nil {
		return err
	}
	for _, dbName := range dbNames {
		if !strings.Contains(dbName, idStore) && !strings.Contains(dbName, internalIDStore) &&
			!strings.Contains(dbName, blockStorePostFix) && !strings.Contains(dbName, txnStorePostFix) &&
			!strings.Contains(dbName, pvtDataStorePostFix) {
			if _, err = dropDB(couchInstance, dbName); err != nil {
				return err
			}
		}
	}
	return nil
}

func dropDB(couchInstance *couchdb.CouchInstance, dbName string) (*couchdb.DBOperationResponse, error) {
	db := &couchdb.CouchDatabase{
		CouchInstance: couchInstance,
		DBName:        dbName,
	}
	return db.DropDatabase()
}

func dropStateLevelDB(rootFSPath string) error {
	stateLeveldbPath := StateDBPath(rootFSPath)
	logger.Infof("Dropping StateLevelDB at location [%s] ...if present", stateLeveldbPath)
	return os.RemoveAll(stateLeveldbPath)
}

func dropConfigHistoryDB(rootFSPath string) error {
	configHistoryDBPath := ConfigHistoryDBPath(rootFSPath)
	logger.Infof("Dropping ConfigHistoryDB at location [%s]", configHistoryDBPath)
	err := os.RemoveAll(configHistoryDBPath)
	return errors.Wrapf(err, "error removing the ConfigHistoryDB located at %s", configHistoryDBPath)
}

func dropBookkeeperDB(rootFSPath string) error {
	bookkeeperDBPath := BookkeeperDBPath(rootFSPath)
	logger.Infof("Dropping BookkeeperDB at location [%s]", bookkeeperDBPath)
	err := os.RemoveAll(bookkeeperDBPath)
	return errors.Wrapf(err, "error removing the BookkeeperDB located at %s", bookkeeperDBPath)
}

func dropHistoryDB(rootFSPath string) error {
	historyDBPath := HistoryDBPath(rootFSPath)
	logger.Infof("Dropping HistoryDB at location [%s] ...if present", historyDBPath)
	return os.RemoveAll(historyDBPath)
}
