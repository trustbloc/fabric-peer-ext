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
	if len(m) > 0 {
		logger.Infof("Pre-reset heights loaded: %v", m)
	}
	return m, nil
}
