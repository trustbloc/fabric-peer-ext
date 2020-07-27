/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cdbblkstorage

import (
	"strings"

	couchdb "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"
	"github.com/pkg/errors"
)

// Rollback reverts changes made to the block store beyond a given block number.
func Rollback(couchInstance *couchdb.CouchInstance, internalQueryLimit int, ledgerID string, targetBlockNum uint64) error {
	if err := recordHeightIfGreaterThanPreviousRecording(couchInstance, ledgerID); err != nil {
		return err
	}

	id := strings.ToLower(ledgerID)
	blockStoreDBName := couchdb.ConstructBlockchainDBName(id, blockStoreName)
	blockStoreDB := couchdb.CouchDatabase{CouchInstance: couchInstance, DBName: blockStoreDBName}
	blockStore := newCDBBlockStore(&blockStoreDB, nil, ledgerID)

	docs, errReadDoc := readDocRange(&blockStoreDB, blockNumberToKey(targetBlockNum+1),
		blockNumberToKey(blockStore.cpInfo.lastBlockNumber+1), int32(internalQueryLimit))
	if errReadDoc != nil {
		return errReadDoc
	}

	var deletedDocs []*couchdb.CouchDoc
	for _, doc := range docs {
		deletedDoc, err := addDeletedFlagToCouchDoc(doc.Value)
		if err != nil {
			return err
		}
		deletedDocs = append(deletedDocs, deletedDoc)
	}

	_, err := blockStoreDB.BatchUpdateDocuments(deletedDocs)
	if err != nil {
		return err
	}

	doc, _, err := blockStoreDB.ReadDoc(blockNumberToKey(targetBlockNum))
	if err != nil {
		return err
	}

	lastBlock, err := couchAttachmentsToBlock(doc.Attachments)
	if err != nil {
		return err
	}

	return blockStore.CheckpointBlock(lastBlock)
}

// ValidateRollbackParams performs necessary validation on the input given for
// the rollback operation.
func ValidateRollbackParams(couchInstance *couchdb.CouchInstance, ledgerID string, targetBlockNum uint64) error {
	logger.Infof("Validating the rollback parameters: ledgerID [%s], block number [%d]",
		ledgerID, targetBlockNum)
	if err := validateLedgerID(couchInstance, ledgerID); err != nil {
		return err
	}
	if err := validateTargetBlkNum(couchInstance, ledgerID, targetBlockNum); err != nil {
		return err
	}

	return nil
}

func validateLedgerID(couchInstance *couchdb.CouchInstance, ledgerID string) error {
	logger.Debugf("Validating the existence of ledgerID [%s]", ledgerID)
	id := strings.ToLower(ledgerID)
	blockStoreDBName := couchdb.ConstructBlockchainDBName(id, blockStoreName)

	blockStoreDB := couchdb.CouchDatabase{CouchInstance: couchInstance, DBName: blockStoreDBName}
	exists, err := blockStoreDB.Exists()
	if err != nil {
		return err
	}
	if !exists {
		return errors.Errorf("ledgerID [%s] does not exist", ledgerID)
	}

	return nil
}

func validateTargetBlkNum(couchInstance *couchdb.CouchInstance, ledgerID string, targetBlockNum uint64) error {
	logger.Debugf("Validating the given block number [%d] against the ledger block height", targetBlockNum)
	id := strings.ToLower(ledgerID)
	blockStoreDBName := couchdb.ConstructBlockchainDBName(id, blockStoreName)

	blockStore := newCDBBlockStore(&couchdb.CouchDatabase{CouchInstance: couchInstance, DBName: blockStoreDBName}, nil, ledgerID)

	if blockStore.cpInfo.lastBlockNumber <= targetBlockNum {
		return errors.Errorf("target block number [%d] should be less than the biggest block number [%d]",
			targetBlockNum, blockStore.cpInfo.lastBlockNumber)
	}

	return nil
}
