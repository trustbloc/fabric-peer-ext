/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"github.com/hyperledger/fabric/common/ledger"
	ledger2 "github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
)

// Ledger is a struct which is used to retrieve data using query
type Ledger struct {
	QueryExecutor  *QueryExecutor
	TxSimulator    *TxSimulator
	BlockchainInfo *common.BlockchainInfo
	Error          error
	BcInfoError    error
}

// GetConfigHistoryRetriever returns the config history retriever
func (m *Ledger) GetConfigHistoryRetriever() (ledger2.ConfigHistoryRetriever, error) {
	panic("not implemented")
}

// GetBlockchainInfo returns the block chain info
func (m *Ledger) GetBlockchainInfo() (*common.BlockchainInfo, error) {
	return m.BlockchainInfo, m.BcInfoError
}

// GetBlockByNumber returns the block by number
func (m *Ledger) GetBlockByNumber(blockNumber uint64) (*common.Block, error) {
	panic("not implemented")
}

// GetBlocksIterator returns the block iterator
func (m *Ledger) GetBlocksIterator(startBlockNumber uint64) (ledger.ResultsIterator, error) {
	panic("not implemented")
}

// Close closes the ledger
func (m *Ledger) Close() {
}

// GetTransactionByID gets the transaction by id
func (m *Ledger) GetTransactionByID(txID string) (*peer.ProcessedTransaction, error) {
	panic("not implemented")
}

// GetBlockByHash returns the block by hash
func (m *Ledger) GetBlockByHash(blockHash []byte) (*common.Block, error) {
	panic("not implemented")
}

// GetBlockByTxID gets the block by transaction id
func (m *Ledger) GetBlockByTxID(txID string) (*common.Block, error) {
	panic("not implemented")
}

// GetTxValidationCodeByTxID gets the validation code
func (m *Ledger) GetTxValidationCodeByTxID(txID string) (peer.TxValidationCode, error) {
	panic("not implemented")
}

// NewTxSimulator returns the transaction simulator
func (m *Ledger) NewTxSimulator(txid string) (ledger2.TxSimulator, error) {
	return m.TxSimulator, m.Error
}

// NewQueryExecutor returns the query executor
func (m *Ledger) NewQueryExecutor() (ledger2.QueryExecutor, error) {
	return m.QueryExecutor, m.Error
}

// NewHistoryQueryExecutor returns the history query executor
func (m *Ledger) NewHistoryQueryExecutor() (ledger2.HistoryQueryExecutor, error) {
	panic("not implemented")
}

// GetPvtDataAndBlockByNum gets private data and block by block number
func (m *Ledger) GetPvtDataAndBlockByNum(blockNum uint64, filter ledger2.PvtNsCollFilter) (*ledger2.BlockAndPvtData, error) {
	panic("not implemented")
}

// GetPvtDataByNum gets private data by number
func (m *Ledger) GetPvtDataByNum(blockNum uint64, filter ledger2.PvtNsCollFilter) ([]*ledger2.TxPvtData, error) {
	panic("not implemented")
}

// CommitWithPvtData commits the private data
func (m *Ledger) CommitWithPvtData(blockAndPvtdata *ledger2.BlockAndPvtData) error {
	panic("not implemented")
}

// CommitPvtDataOfOldBlocks commits the private data of old blocks
func (m *Ledger) CommitPvtDataOfOldBlocks(blockPvtData []*ledger2.BlockPvtData) ([]*ledger2.PvtdataHashMismatch, error) {
	panic("not implemented")
}

// GetMissingPvtDataTracker returns the private data tracker
func (m *Ledger) GetMissingPvtDataTracker() (ledger2.MissingPvtDataTracker, error) {
	panic("not implemented")
}

//CheckpointBlock updates checkpoint info to given block
func (m *Ledger) CheckpointBlock(block *common.Block) error {
	panic("not implemented")
}
