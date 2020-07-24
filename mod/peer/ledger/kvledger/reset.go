/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvledger

import (
	"github.com/pkg/errors"

	"github.com/hyperledger/fabric/common/ledger/util/leveldbhelper"
	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/core/ledger"
	couchdb "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"

	"github.com/trustbloc/fabric-peer-ext/pkg/blkstorage/cdbblkstorage"
)

// LoadPreResetHeight returns the prereset height for the specified ledgers.
func LoadPreResetHeight(ledgerconfig *ledger.Config, ledgerIDs []string) (map[string]uint64, error) {
	stateDBCouchInstance, err := couchdb.CreateCouchInstance(ledgerconfig.StateDBConfig.CouchDB, &disabled.Provider{})
	if err != nil {
		return nil, errors.WithMessage(err, "obtaining CouchDB instance failed")
	}

	return cdbblkstorage.LoadPreResetHeight(stateDBCouchInstance, ledgerIDs)
}

func ClearPreResetHeight(ledgerconfig *ledger.Config, ledgerIDs []string) error {
	stateDBCouchInstance, err := couchdb.CreateCouchInstance(ledgerconfig.StateDBConfig.CouchDB, &disabled.Provider{})
	if err != nil {
		return errors.WithMessage(err, "obtaining CouchDB instance failed")
	}

	return cdbblkstorage.ClearPreResetHeight(stateDBCouchInstance, ledgerIDs)
}

func ResetAllKVLedgers(ledgerconfig *ledger.Config) error {
	fileLockPath := fileLockPath(ledgerconfig.RootFSPath)
	fileLock := leveldbhelper.NewFileLock(fileLockPath)
	if err := fileLock.Lock(); err != nil {
		return errors.Wrap(err, "as another peer node command is executing,"+
			" wait for that command to complete its execution or terminate it before retrying")
	}
	defer fileLock.Unlock()

	logger.Info("Resetting all channel ledgers to genesis block")
	logger.Infof("Ledger data folder from config = [%s]", ledgerconfig.RootFSPath)
	if err := dropDBs(ledgerconfig); err != nil {
		return err
	}
	if err := ResetBlockStore(ledgerconfig); err != nil {
		return err
	}
	logger.Info("All channel ledgers have been successfully reset to the genesis block")
	return nil
}

func ResetBlockStore(ledgerconfig *ledger.Config) error {
	logger.Infof("Resetting BlockStore to genesis block")
	stateDBCouchInstance, err := couchdb.CreateCouchInstance(ledgerconfig.StateDBConfig.CouchDB, &disabled.Provider{})
	if err != nil {
		return errors.WithMessage(err, "obtaining CouchDB instance failed")
	}
	return cdbblkstorage.ResetBlockStore(stateDBCouchInstance)
}
