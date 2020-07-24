/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cdbblkstorage

import (
	"strconv"
	"strings"

	couchdb "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"
)

// DeleteBlockStore deletes block store
func DeleteBlockStore(couchInstance *couchdb.CouchInstance) error {
	dbsName, _ := couchInstance.RetrieveApplicationDBNames()
	for _, dbName := range dbsName {
		if strings.Contains(dbName, "$$blocks_") || strings.Contains(dbName, "$$transactions_") {
			if err := dropDB(couchInstance, dbName); err != nil {
				logger.Errorf("Error dropping CouchDB database %s", dbName)
				return err
			}
		}
	}

	return nil
}

func dropDB(couchInstance *couchdb.CouchInstance, dbName string) error {
	db := &couchdb.CouchDatabase{
		CouchInstance: couchInstance,
		DBName:        dbName,
	}

	_, err := db.DropDatabase()

	return err
}

// recordHeightIfGreaterThanPreviousRecording creates a file "__preResetHeight" in the ledger's
// directory. This file contains human readable string for the current block height. This function
// only overwrites this information if the current block height is higher than the one recorded in
// the existing file (if present). This helps in achieving fail-safe behviour of reset utility
func recordHeightIfGreaterThanPreviousRecording(couchInstance *couchdb.CouchInstance, ledgerID string) error {
	logger.Infof("Preparing to record current height for ledger at [%s]", ledgerID)
	id := strings.ToLower(ledgerID)
	blockStoreDBName := couchdb.ConstructBlockchainDBName(id, blockStoreName)
	blockStoreDB := couchdb.CouchDatabase{CouchInstance: couchInstance, DBName: blockStoreDBName}
	blockStore := newCDBBlockStore(&couchdb.CouchDatabase{CouchInstance: couchInstance, DBName: blockStoreDBName}, nil, ledgerID)

	doc, _, err := blockStoreDB.ReadDoc(preResetHeightKey)
	if err != nil {
		return err
	}

	exists := false
	if doc != nil {
		exists = true
	}

	logger.Infof("preResetHt already exists? = %t", exists)
	previuoslyRecordedHt := uint64(0)
	if exists {
		v, err := couchDocToJSON(doc)
		if err != nil {
			return err
		}

		if previuoslyRecordedHt, err = strconv.ParseUint(v["height"].(string), 10, 64); err != nil {
			return err
		}
		logger.Infof("preResetHt contains height = %d", previuoslyRecordedHt)
	}

	currentHt := blockStore.cpInfo.lastBlockNumber + 1
	if currentHt > previuoslyRecordedHt {
		logger.Infof("Recording current height [%d]", currentHt)
		doc, err := preResetHeightKeyToCouchDoc(strconv.FormatUint(currentHt, 10))
		if err != nil {
			return err
		}

		_, err = blockStoreDB.SaveDoc(preResetHeightKey, "", doc)
		return err
	}
	logger.Infof("Not recording current height [%d] since this is less than previously recorded height [%d]",
		currentHt, previuoslyRecordedHt)

	return nil
}

// LoadPreResetHeight searches the preResetHeight files for the specified ledgers and
// returns a map of channelname to the last recorded block height during one of the reset operations.
func LoadPreResetHeight(couchInstance *couchdb.CouchInstance, ledgerIDs []string) (map[string]uint64, error) {
	logger.Debug("Loading Pre-reset heights")
	m := map[string]uint64{}
	for _, ledgerID := range ledgerIDs {
		id := strings.ToLower(ledgerID)
		blockStoreDBName := couchdb.ConstructBlockchainDBName(id, blockStoreName)
		blockStoreDB := couchdb.CouchDatabase{CouchInstance: couchInstance, DBName: blockStoreDBName}

		doc, _, err := blockStoreDB.ReadDoc(preResetHeightKey)
		if err != nil {
			return nil, err
		}

		if doc != nil {
			v, err := couchDocToJSON(doc)
			if err != nil {
				return nil, err
			}

			previuoslyRecordedHt, err := strconv.ParseUint(v["height"].(string), 10, 64)
			if err != nil {
				return nil, err
			}

			m[ledgerID] = previuoslyRecordedHt
		}
	}
	if len(m) > 0 {
		logger.Infof("Pre-reset heights loaded: %v", m)
	}
	return m, nil
}

// ClearPreResetHeight deletes the files that contain the last recorded reset heights for the specified ledgers
func ClearPreResetHeight(couchInstance *couchdb.CouchInstance, ledgerIDs []string) error {
	logger.Info("Clearing Pre-reset heights")
	for _, ledgerID := range ledgerIDs {
		id := strings.ToLower(ledgerID)
		blockStoreDBName := couchdb.ConstructBlockchainDBName(id, blockStoreName)
		blockStoreDB := couchdb.CouchDatabase{CouchInstance: couchInstance, DBName: blockStoreDBName}

		doc, _, err := blockStoreDB.ReadDoc(preResetHeightKey)
		if err != nil {
			return err
		}

		doc, err = addDeletedFlagToCouchDoc(doc.JSONValue)
		if err != nil {
			return err
		}

		_, err = blockStoreDB.BatchUpdateDocuments([]*couchdb.CouchDoc{doc})
		if err != nil {
			return err
		}
	}
	logger.Info("Cleared off Pre-reset heights")

	return nil

}

// ResetBlockStore drops the block storage index and truncates the blocks files for all channels/ledgers to genesis blocks
func ResetBlockStore(couchInstance *couchdb.CouchInstance) error { // nolint: gocyclo
	dbsName, _ := couchInstance.RetrieveApplicationDBNames()
	for _, dbName := range dbsName {
		if strings.Contains(dbName, "$$blocks_") {
			split := strings.Split(dbName, "$$")
			blockStoreDBName := couchdb.ConstructBlockchainDBName(split[0], blockStoreName)
			blockStoreDB := couchdb.CouchDatabase{CouchInstance: couchInstance, DBName: blockStoreDBName}
			txnStoreDBName := couchdb.ConstructBlockchainDBName(split[0], txnStoreName)
			txnStoreDB := couchdb.CouchDatabase{CouchInstance: couchInstance, DBName: txnStoreDBName}

			if err := recordHeightIfGreaterThanPreviousRecording(couchInstance, split[0]); err != nil {
				return err
			}

			// get genesis block
			genesisDoc, _, errReadDoc := blockStoreDB.ReadDoc(blockNumberToKey(0))
			if errReadDoc != nil {
				return errReadDoc
			}

			genesisBlock, errToBlock := couchDocToBlock(genesisDoc)
			if errToBlock != nil {
				return errToBlock
			}

			// get preResetHeight
			preResetHeight, errLoadPreReset := LoadPreResetHeight(couchInstance, []string{split[0]})
			if errLoadPreReset != nil {
				return errLoadPreReset
			}

			// drop block store
			if _, err := blockStoreDB.DropDatabase(); err != nil {
				return err
			}
			// drop txn store
			if _, err := txnStoreDB.DropDatabase(); err != nil {
				return err
			}

			// create block store db
			_, err := createCommitterBlockStore(couchInstance, split[0], blockStoreDBName, txnStoreDBName)
			if err != nil {
				return err
			}

			// add genesis block
			store := newCDBBlockStore(&blockStoreDB, &txnStoreDB, split[0])
			if errAddBlock := store.AddBlock(genesisBlock); errAddBlock != nil {
				return errAddBlock
			}

			preResetHeightDoc, err := preResetHeightKeyToCouchDoc(strconv.FormatUint(preResetHeight[split[0]], 10))
			if err != nil {
				return err
			}

			_, errSaveDoc := blockStoreDB.SaveDoc(preResetHeightKey, "", preResetHeightDoc)
			if errSaveDoc != nil {
				return errSaveDoc
			}
		}
	}

	return nil
}
