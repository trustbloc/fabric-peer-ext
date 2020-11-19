/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cdbblkstorage

import (
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/common/ledger/blkstorage"
	"github.com/hyperledger/fabric/common/ledger/snapshot"
	"github.com/pkg/errors"
)

type store struct {
	ledgerID   string
	blockStore couchDB
	txnStore   couchDB
	attachTxn  bool
}

func newStore(ledgerID string, blockStore, txnStore couchDB) *store {
	return &store{
		ledgerID:   ledgerID,
		blockStore: blockStore,
		txnStore:   txnStore,
		attachTxn:  false,
	}
}

func (s *store) RetrieveBlockByHash(blockHash []byte) (*common.Block, error) {
	logger.Infof("[%s] Retrieving block from store for hash", s.ledgerID)

	blockHashHex := hex.EncodeToString(blockHash)
	const queryFmt = `
	{
		"selector": {
			"` + blockHeaderField + `.` + blockHashField + `": {
				"$eq": "%s"
			}
		},
		"use_index": ["_design/` + blockHashIndexDoc + `", "` + blockHashIndexName + `"]
	}`

	block, err := retrieveBlockQuery(s.blockStore, fmt.Sprintf(queryFmt, blockHashHex))
	if err != nil {
		// note: allow ErrNotFoundInIndex to pass through
		return nil, err
	}

	return block, nil
}

func (s *store) RetrieveBlockByNumber(blockNum uint64) (*common.Block, error) {
	logger.Debugf("[%s] Retrieving block [%d] from store", s.ledgerID, blockNum)

	id := blockNumberToKey(blockNum)

	doc, _, err := s.blockStore.ReadDoc(id)
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("retrieval of block [%d] from couchDB failed", blockNum))
	}
	if doc == nil {
		return nil, blkstorage.ErrNotFoundInIndex
	}

	block, err := couchDocToBlock(doc)
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("unmarshal of block [%d] from couchDB failed", blockNum))
	}

	return block, nil
}

func (s *store) RetrieveTxByID(txID string) (*common.Envelope, error) {
	logger.Debugf("[%s] Retrieving transaction [%s] from store", s.ledgerID, txID)

	doc, _, err := s.txnStore.ReadDoc(txID)
	if err != nil {
		// note: allow ErrNotFoundInIndex to pass through
		logger.Debugf("[%s] Error retrieving transaction [%s] from store: %s", s.ledgerID, txID, err)

		return nil, err
	}
	if doc == nil {
		logger.Debugf("[%s] Transaction [%s] not found", s.ledgerID, txID)

		return nil, blkstorage.ErrNotFoundInIndex
	}

	// If this transaction includes the envelope as a valid attachment then can return immediately.
	if len(doc.Attachments) > 0 {
		attachedEnv, e := couchAttachmentsToTxnEnvelope(doc.Attachments)
		if e == nil {
			return attachedEnv, nil
		}
		logger.Debugf("transaction [%s] has attachment but failed to be extracted into envelope", err)
	}

	// Otherwise, we need to extract the transaction from the block document.
	block, err := s.RetrieveBlockByTxID(txID)
	if err != nil {
		// note: allow ErrNotFoundInIndex to pass through
		return nil, err
	}

	return extractTxnEnvelopeFromBlock(block, txID)
}

// RetrieveBlockByTxID returns a block for a given transaction ID
func (s *store) RetrieveBlockByTxID(txID string) (*common.Block, error) {
	logger.Debugf("[%s] Retrieving block from store for transaction ID [%s]", s.ledgerID, txID)

	blockHash, err := s.retrieveBlockHashByTxID(txID)
	if err != nil {
		return nil, err
	}

	return s.RetrieveBlockByHash(blockHash)
}

// RetrieveTxValidationCodeByTxID returns a TX validation code for a given transaction ID
func (s *store) RetrieveTxValidationCodeByTxID(txID string) (peer.TxValidationCode, error) {
	logger.Debugf("[%s] Retrieving TxValidationCode from store for transaction ID [%s]", s.ledgerID, txID)

	jsonResult, err := retrieveJSONQuery(s.txnStore, txID)
	if err != nil {
		// note: allow ErrNotFoundInIndex to pass through
		return peer.TxValidationCode(-1), err
	}

	txnValidationCodeStoredUT, ok := jsonResult[txnValidationCode]
	if !ok {
		return peer.TxValidationCode_INVALID_OTHER_REASON, errors.Errorf("validation code was not found for transaction ID [%s]", txID)
	}

	txnValidationCodeStored, ok := txnValidationCodeStoredUT.(string)
	if !ok {
		return peer.TxValidationCode_INVALID_OTHER_REASON, errors.Errorf("validation code has invalid type for transaction ID [%s]", txID)
	}

	const sizeOfTxValidationCode = 32
	txnValidationCode, err := strconv.ParseInt(txnValidationCodeStored, txnValidationCodeBase, sizeOfTxValidationCode)
	if err != nil {
		return peer.TxValidationCode_INVALID_OTHER_REASON, errors.Wrapf(err, "validation code was invalid for transaction ID [%s]", txID)
	}

	return peer.TxValidationCode(txnValidationCode), nil
}

func (s *store) retrieveBlockHashByTxID(txID string) ([]byte, error) {
	jsonResult, err := retrieveJSONQuery(s.txnStore, txID)
	if err != nil {
		// note: allow ErrNotFoundInIndex to pass through
		logger.Errorf("retrieving transaction document from DB failed : %s", err)
		return nil, err
	}

	blockHashStoredUT, ok := jsonResult[txnBlockHashField]
	if !ok {
		return nil, errors.Errorf("block hash was not found for transaction ID [%s]", txID)
	}

	blockHashStored, ok := blockHashStoredUT.(string)
	if !ok {
		return nil, errors.Errorf("block hash has invalid type for transaction ID [%s]", txID)
	}

	blockHash, err := hex.DecodeString(blockHashStored)
	if err != nil {
		return nil, errors.Wrapf(err, "block hash was invalid for transaction ID [%s]", txID)
	}

	return blockHash, nil
}

func (s *store) store(block *common.Block) error {
	err := s.storeBlock(block)
	if err != nil {
		return err
	}

	return s.storeTransactions(block)
}

func (s *store) storeBlock(block *common.Block) error {
	doc, err := blockToCouchDoc(block)
	if err != nil {
		return errors.WithMessage(err, "converting block to couchDB document failed")
	}

	logger.Debugf("[%s] Storing block [%d]", s.ledgerID, block.Header.Number)

	id := blockNumberToKey(block.GetHeader().GetNumber())

	rev, err := s.blockStore.SaveDoc(id, "", doc)
	if err != nil {
		return errors.WithMessage(err, "adding block to couchDB failed")
	}
	logger.Debugf("block added to couchDB [%d, %s]", block.GetHeader().GetNumber(), rev)

	return nil
}

func (s *store) storeTransactions(block *common.Block) error {
	docs, err := blockToTxnCouchDocs(block, s.attachTxn)
	if err != nil {
		return errors.WithMessage(err, "converting block to couchDB txn documents failed")
	}

	if len(docs) == 0 {
		return nil
	}

	logger.Debugf("[%s] Storing transactions for block [%d]", s.ledgerID, block.Header.Number)

	_, err = s.txnStore.BatchUpdateDocuments(docs)
	if err != nil {
		return errors.WithMessage(err, "adding block to couchDB failed")
	}
	logger.Debugf("block transactions added to couchDB [%d]", block.GetHeader().GetNumber())
	return nil
}

func (s *store) exportTxIDs(dir string, newHashFunc snapshot.NewHashFunc) ([]byte, uint64, error) {
	// Get everything from the DB
	// TODO: Is it practical to be returning all rows from a database?
	results, _, err := s.txnStore.QueryDocuments(`{"selector": {}}`)
	if err != nil {
		return nil, 0, err
	}

	var numTxIDs uint64 = 0
	var dataFile *snapshot.FileWriter

	for _, r := range results {
		if numTxIDs == 0 { // first iteration, create the data file
			fileName := filepath.Join(dir, snapshotDataFileName)
			dataFile, err = snapshot.CreateFile(fileName, snapshotFileFormat, newHashFunc)
			if err != nil {
				return nil, 0, err
			}

			logger.Debugf("[%s] Created file [%s]", s.ledgerID, fileName)

			defer func() {
				if err = dataFile.Close(); err != nil {
					logger.Warnf("Error closing datafile: %s", err)
				}
			}()
		}

		logger.Debugf("[%s] Adding TxID [%s]", s.ledgerID, r.ID)

		if e := dataFile.EncodeString(r.ID); e != nil {
			return nil, 0, e
		}

		numTxIDs++
	}

	if dataFile == nil {
		logger.Infof("[%s] No data file created", s.ledgerID)
		return nil, 0, nil
	}

	dataHash, err := dataFile.Done()
	if err != nil {
		return nil, 0, err
	}

	return dataHash, numTxIDs, nil
}
