/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cdbblkstorage

import (
	"bytes"
	"fmt"
	"math"
	"path/filepath"
	"sync"

	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/common/ledger/snapshot"
	couchdb "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/pkg/errors"

	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

const (
	snapshotFileFormat       = byte(1)
	snapshotDataFileName     = "txids.data"
	snapshotMetadataFileName = "txids.metadata"
)

// cdbBlockStore ...
type cdbBlockStore struct {
	ledgerID   string
	cpInfo     *checkpointInfo
	cpInfoCond *sync.Cond
	cp         *checkpoint
	store      *store
	cache      *blockCache
}

type config struct {
	blockByNumSize  uint
	blockByHashSize uint
}

type option func(cfg *config)

// withBlockByNumCacheSize sets the size of the block-by-number cache
// A size of zero disables the cache
func withBlockByNumCacheSize(size uint) option {
	return func(cfg *config) {
		cfg.blockByNumSize = size
	}
}

// withBlockByHashCacheSize sets the size of the block-by-hash cache
// A size of zero disables the cache
func withBlockByHashCacheSize(size uint) option {
	return func(cfg *config) {
		cfg.blockByHashSize = size
	}
}

// newCDBBlockStore constructs block store based on CouchDB
func newCDBBlockStore(blockStore *couchdb.CouchDatabase, txnStore *couchdb.CouchDatabase, ledgerID string, opts ...option) *cdbBlockStore {
	cp := newCheckpoint(blockStore)

	store := newStore(ledgerID, blockStore, txnStore)

	cdbBlockStore := &cdbBlockStore{
		ledgerID: ledgerID,
		store:    store,
		cp:       cp,
	}

	// cp = checkpointInfo, retrieve from the database the last block number that was written to that db.
	cpInfo := cdbBlockStore.cp.getCheckpointInfo()
	err := cdbBlockStore.cp.saveCurrentInfo(cpInfo)
	if err != nil {
		panic(fmt.Sprintf("Could not save cpInfo info to db: %s", err))
	}

	// Update the manager with the checkpoint info and the file writer
	cdbBlockStore.cpInfo = cpInfo

	// Create a checkpoint condition (event) variable, for the  goroutine waiting for
	// or announcing the occurrence of an event.
	cdbBlockStore.cpInfoCond = sync.NewCond(&sync.Mutex{})

	var bcInfo *common.BlockchainInfo

	if cpInfo.isChainEmpty {
		bcInfo = &common.BlockchainInfo{}
	} else {
		logger.Debugf("[%s] Loading block %d from database", ledgerID, cpInfo.lastBlockNumber)

		//If start up is a restart of an existing storage, update BlockchainInfo for external API's
		lastBlock, err := store.RetrieveBlockByNumber(cpInfo.lastBlockNumber)
		if err != nil {
			panic(fmt.Sprintf("Could not load block %d from database: %s", cpInfo.lastBlockNumber, err))
		}

		lastBlockHeader := lastBlock.GetHeader()

		bcInfo = &common.BlockchainInfo{
			Height:            lastBlockHeader.GetNumber() + 1,
			CurrentBlockHash:  protoutil.BlockHeaderHash(lastBlockHeader),
			PreviousBlockHash: lastBlockHeader.GetPreviousHash(),
		}
	}

	cdbBlockStore.cache = newCache(ledgerID, store, bcInfo, resolveOptions(opts))

	return cdbBlockStore
}

// AddBlock adds a new block
func (s *cdbBlockStore) AddBlock(block *common.Block) error {
	if !roles.IsCommitter() {
		// Nothing to do if not a committer
		return nil
	}

	err := s.validateBlock(block)
	if err != nil {
		return err
	}

	err = s.store.store(block)
	if err != nil {
		return err
	}

	return s.CheckpointBlock(block, noOp)
}

//validateBlock validates block before adding to store
func (s *cdbBlockStore) validateBlock(block *common.Block) error {
	s.cpInfoCond.L.Lock()
	defer s.cpInfoCond.L.Unlock()

	if s.cpInfo.isChainEmpty {
		//chain is empty, no need to validate, first block it is.
		return nil
	}
	if block.Header.Number != s.cpInfo.lastBlockNumber+1 {
		return errors.Errorf(
			"block number should have been %d but was %d",
			s.cpInfo.lastBlockNumber+1, block.Header.Number,
		)
	}

	// Add the previous hash check - Though, not essential but may not be a bad idea to
	// verify the field `block.Header.PreviousHash` present in the block.
	// This check is a simple bytes comparison and hence does not cause any observable performance penalty
	// and may help in detecting a rare scenario if there is any bug in the ordering service.
	if !bytes.Equal(block.Header.PreviousHash, s.cpInfo.currentHash) {
		return errors.Errorf(
			"unexpected Previous block hash. Expected PreviousHash = [%x], PreviousHash referred in the latest block= [%x]",
			s.cpInfo.currentHash, block.Header.PreviousHash,
		)
	}

	return nil
}

func noOp() {
}

func (s *cdbBlockStore) CheckpointBlock(block *common.Block, notify func()) error {
	//Update the checkpoint info with the results of adding the new block
	newCPInfo := &checkpointInfo{
		isChainEmpty:    false,
		lastBlockNumber: block.Header.Number,
		currentHash:     protoutil.BlockHeaderHash(block.Header),
	}

	s.cache.put(block)

	if roles.IsCommitter() {
		//save the checkpoint information in the database
		err := s.cp.saveCurrentInfo(newCPInfo)
		if err != nil {
			return errors.WithMessage(err, "adding cpInfo to couchDB failed")
		}
	}

	bcInfo := &common.BlockchainInfo{
		Height:            newCPInfo.lastBlockNumber + 1,
		CurrentBlockHash:  newCPInfo.currentHash,
		PreviousBlockHash: block.Header.PreviousHash,
	}

	s.cache.setBlockchainInfo(bcInfo)

	notify()

	//update the checkpoint info (for storage) and the blockchain info (for APIs) in the manager
	s.updateCheckpoint(newCPInfo)

	return nil
}

// GetBlockchainInfo returns the current info about blockchain
func (s *cdbBlockStore) GetBlockchainInfo() (*common.BlockchainInfo, error) {
	return s.cache.getBlockchainInfo(), nil
}

// RetrieveBlocks returns an iterator that can be used for iterating over a range of blocks
func (s *cdbBlockStore) RetrieveBlocks(startNum uint64) (ledger.ResultsIterator, error) {
	return newBlockItr(s, startNum), nil
}

// RetrieveBlockByHash returns the block for given block-hash
func (s *cdbBlockStore) RetrieveBlockByHash(blockHash []byte) (*common.Block, error) {
	return s.cache.getBlockByHash(blockHash)
}

// RetrieveBlockByNumber returns the block at a given blockchain height
func (s *cdbBlockStore) RetrieveBlockByNumber(blockNum uint64) (*common.Block, error) {
	// interpret math.MaxUint64 as a request for last block
	if blockNum == math.MaxUint64 {
		bcinfo, err := s.GetBlockchainInfo()
		if err != nil {
			return nil, errors.WithMessage(err, "retrieval of blockchain info failed")
		}
		blockNum = bcinfo.Height - 1
	}

	return s.cache.getBlockByNumber(blockNum)
}

// RetrieveTxByID returns a transaction for given transaction id
func (s *cdbBlockStore) RetrieveTxByID(txID string) (*common.Envelope, error) {
	return s.store.RetrieveTxByID(txID)
}

// RetrieveTxByBlockNumTranNum returns a transaction for given block ID and transaction ID
func (s *cdbBlockStore) RetrieveTxByBlockNumTranNum(blockNum uint64, tranNum uint64) (*common.Envelope, error) {
	block, err := s.cache.getBlockByNumber(blockNum)
	if err != nil {
		// note: allow ErrNotFoundInIndex to pass through
		return nil, err
	}

	return extractEnvelopeFromBlock(block, tranNum)
}

// RetrieveBlockByTxID returns a block for a given transaction ID
func (s *cdbBlockStore) RetrieveBlockByTxID(txID string) (*common.Block, error) {
	return s.store.RetrieveBlockByTxID(txID)
}

// RetrieveTxValidationCodeByTxID returns a TX validation code for a given transaction ID
func (s *cdbBlockStore) RetrieveTxValidationCodeByTxID(txID string) (peer.TxValidationCode, error) {
	return s.store.RetrieveTxValidationCodeByTxID(txID)
}

// ExportTxIds creates two files in the specified dir and returns a map that contains
// the mapping between the names of the files and their hashes.
// Technically, the TxIDs appear in the sort order of radix-sort/shortlex. However,
// since practically all the TxIDs are of same length, so the sort order would be the lexical sort order
func (s *cdbBlockStore) ExportTxIds(dir string, newHashFunc snapshot.NewHashFunc) (map[string][]byte, error) {
	logger.Infof("[%s] Exporting transaction IDs to directory [%s]", s.ledgerID, dir)

	dataHash, numTxIDs, err := s.store.exportTxIDs(dir, newHashFunc)
	if err != nil {
		return nil, err
	}

	if dataHash == nil {
		return nil, nil
	}

	// create the metadata file
	metadataFileName := filepath.Join(dir, snapshotMetadataFileName)
	metadataFile, err := snapshot.CreateFile(metadataFileName, snapshotFileFormat, newHashFunc)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = metadataFile.Close(); err != nil {
			logger.Warnf("Error closing metadataFile: %s", err)
		}
	}()

	logger.Infof("[%s] Created file [%s]", s.ledgerID, metadataFileName)

	if err = metadataFile.EncodeUVarint(numTxIDs); err != nil {
		return nil, err
	}

	metadataHash, err := metadataFile.Done()
	if err != nil {
		return nil, err
	}

	return map[string][]byte{
		snapshotDataFileName:     dataHash,
		snapshotMetadataFileName: metadataHash,
	}, nil
}

// Shutdown closes the storage instance
func (s *cdbBlockStore) Shutdown() {
}

func (s *cdbBlockStore) updateCheckpoint(cpInfo *checkpointInfo) {
	s.cpInfoCond.L.Lock()
	defer s.cpInfoCond.L.Unlock()
	s.cpInfo = cpInfo

	logger.Debugf("[%s] Broadcasting checkpoint info for block [%d]", s.ledgerID, cpInfo.lastBlockNumber)

	s.cpInfoCond.Broadcast()
}

func resolveOptions(opts []option) config {
	var cfg config

	for _, opt := range opts {
		opt(&cfg)
	}

	return cfg
}
