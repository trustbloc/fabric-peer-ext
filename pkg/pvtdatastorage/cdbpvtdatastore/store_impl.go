/*
Copyright IBM Corp, SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pvtdatastorage

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/ledger/util/leveldbhelper"
	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/pvtdatapolicy"
	"github.com/hyperledger/fabric/core/ledger/pvtdatastorage"
	"github.com/hyperledger/fabric/core/ledger/util/couchdb"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/pvtdatastorage/common"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
	"github.com/willf/bitset"
)

var logger = flogging.MustGetLogger("cdbpvtdatastore")

const (
	pvtDataStoreName = "pvtdata"
)

type provider struct {
	couchInstance            *couchdb.CouchInstance
	missingKeysIndexProvider *leveldbhelper.Provider
	ledgerConfig             *ledger.Config
}

type store struct {
	ledgerid           string
	btlPolicy          pvtdatapolicy.BTLPolicy
	db                 *couchdb.CouchDatabase
	lastCommittedBlock uint64
	purgerLock         *sync.Mutex
	pendingPvtData     *pendingPvtData
	collElgProc        *common.CollElgProc
	// missing keys db
	missingKeysIndexDB *leveldbhelper.DBHandle
	isEmpty            bool
	// After committing the pvtdata of old blocks,
	// the `isLastUpdatedOldBlocksSet` is set to true.
	// Once the stateDB is updated with these pvtdata,
	// the `isLastUpdatedOldBlocksSet` is set to false.
	// isLastUpdatedOldBlocksSet is mainly used during the
	// recovery process. During the peer startup, if the
	// isLastUpdatedOldBlocksSet is set to true, the pvtdata
	// in the stateDB needs to be updated before finishing the
	// recovery operation.
	isLastUpdatedOldBlocksSet bool
	ledgerConfig              *ledger.Config
}

type pendingPvtData struct {
	PvtDataDoc         *couchdb.CouchDoc `json:"pvtDataDoc"`
	MissingDataEntries map[string]string `json:"missingDataEntries"`
	BatchPending       bool              `json:"batchPending"`
}

// lastUpdatedOldBlocksList keeps the list of last updated blocks
// and is stored as the value of lastUpdatedOldBlocksKey (defined in kv_encoding.go)
type lastUpdatedOldBlocksList []uint64

//////// Provider functions  /////////////
//////////////////////////////////////////

// NewProvider instantiates a private data storage provider backed by CouchDB
func NewProvider(conf *ledger.PrivateData, ledgerconfig *ledger.Config) (pvtdatastorage.Provider, error) {
	logger.Debugf("constructing CouchDB private data storage provider")
	couchDBConfig := ledgerconfig.StateDB.CouchDB

	return newProviderWithDBDef(couchDBConfig, conf, ledgerconfig)
}

func newProviderWithDBDef(couchDBConfig *couchdb.Config, conf *ledger.PrivateData, ledgerconfig *ledger.Config) (pvtdatastorage.Provider, error) {
	couchInstance, err := couchdb.CreateCouchInstance(couchDBConfig, &disabled.Provider{})
	if err != nil {
		return nil, errors.WithMessage(err, "obtaining CouchDB instance failed")
	}

	dbPath := conf.StorePath
	missingKeysIndexProvider := leveldbhelper.NewProvider(&leveldbhelper.Conf{DBPath: dbPath})

	return &provider{couchInstance, missingKeysIndexProvider, ledgerconfig}, nil
}

// OpenStore returns a handle to a store
func (p *provider) OpenStore(ledgerid string) (pvtdatastorage.Store, error) {
	// Create couchdb
	pvtDataStoreDBName := couchdb.ConstructBlockchainDBName(strings.ToLower(ledgerid), pvtDataStoreName)
	var db *couchdb.CouchDatabase
	var err error
	if roles.IsCommitter() {
		db, err = createPvtDataCouchDB(p.couchInstance, pvtDataStoreDBName)
		if err != nil {
			return nil, err
		}
		// Create missing pvt keys index in leveldb
		missingKeysIndexDB := p.missingKeysIndexProvider.GetDBHandle(ledgerid)

		purgerLock := &sync.Mutex{}
		s := &store{db: db, ledgerid: ledgerid,
			collElgProc:        common.NewCollElgProc(purgerLock, missingKeysIndexDB, p.ledgerConfig),
			purgerLock:         purgerLock,
			missingKeysIndexDB: missingKeysIndexDB,
			pendingPvtData:     &pendingPvtData{BatchPending: false},
			ledgerConfig:       p.ledgerConfig,
		}

		if errInitState := s.initState(); errInitState != nil {
			return nil, errInitState
		}
		s.collElgProc.LaunchCollElgProc()

		logger.Debugf("Pvtdata store opened. Initial state: isEmpty [%t], lastCommittedBlock [%d]",
			s.isEmpty, s.lastCommittedBlock)

		return s, nil
	}

	db, err = getPvtDataCouchInstance(p.couchInstance, pvtDataStoreDBName)
	if err != nil {
		return nil, err
	}
	s := &store{db: db, ledgerid: ledgerid,
		pendingPvtData: &pendingPvtData{BatchPending: false},
	}
	lastCommittedBlock, _, err := lookupLastBlock(db)
	if err != nil {
		return nil, err
	}
	s.isEmpty = true
	if lastCommittedBlock != 0 {
		s.lastCommittedBlock = lastCommittedBlock
		s.isEmpty = false
	}
	return s, nil

}

// Close closes the store
func (p *provider) Close() {
	p.missingKeysIndexProvider.Close()
}

//////// store functions  ////////////////
//////////////////////////////////////////

func (s *store) initState() error {
	var blist lastUpdatedOldBlocksList
	lastCommittedBlock, _, err := lookupLastBlock(s.db)
	if err != nil {
		return err
	}
	s.isEmpty = true
	if lastCommittedBlock != 0 {
		s.lastCommittedBlock = lastCommittedBlock
		s.isEmpty = false
	}

	if s.pendingPvtData, err = s.hasPendingCommit(); err != nil {
		return err
	}

	if blist, err = common.GetLastUpdatedOldBlocksList(s.missingKeysIndexDB); err != nil {
		return err
	}
	if len(blist) > 0 {
		s.isLastUpdatedOldBlocksSet = true
	} // false if not set

	return nil
}

func (s *store) Init(btlPolicy pvtdatapolicy.BTLPolicy) {
	s.btlPolicy = btlPolicy
}

// Prepare implements the function in the interface `Store`
func (s *store) Prepare(blockNum uint64, pvtData []*ledger.TxPvtData, missingPvtData ledger.TxMissingPvtDataMap) error {
	if !roles.IsCommitter() {
		panic("calling Prepare on a peer that is not a committer")
	}

	if s.pendingPvtData.BatchPending {
		return pvtdatastorage.NewErrIllegalCall(`A pending batch exists as as result of last invoke to "Prepare" call. Invoke "Commit" or "Rollback" on the pending batch before invoking "Prepare" function`)
	}

	expectedBlockNum := s.nextBlockNum()
	if expectedBlockNum != blockNum {
		return pvtdatastorage.NewErrIllegalCall(fmt.Sprintf("Expected block number=%d, received block number=%d", expectedBlockNum, blockNum))
	}

	storeEntries, err := common.PrepareStoreEntries(blockNum, pvtData, s.btlPolicy, missingPvtData)
	if err != nil {
		return err
	}

	pvtDataDoc, err := createPvtDataCouchDoc(storeEntries, blockNum, "")
	if err != nil {
		return err
	}

	s.pendingPvtData = &pendingPvtData{BatchPending: true}
	if pvtDataDoc != nil || len(storeEntries.MissingDataEntries) > 0 {
		s.pendingPvtData.MissingDataEntries, err = s.perparePendingMissingDataEntries(storeEntries.MissingDataEntries)
		if err != nil {
			return err
		}
		s.pendingPvtData.PvtDataDoc = pvtDataDoc
		if err := s.savePendingKey(); err != nil {
			return err
		}

	}
	logger.Debugf("Saved %d private data write sets for block [%d]", len(pvtData), blockNum)
	return nil
}

// Commit implements the function in the interface `Store`
func (s *store) Commit() error {
	if !roles.IsCommitter() {
		panic("calling Commit on a peer that is not a committer")
	}

	if !s.pendingPvtData.BatchPending {
		return pvtdatastorage.NewErrIllegalCall("No pending batch to commit")
	}

	committingBlockNum := s.nextBlockNum()
	logger.Debugf("Committing private data for block [%d]", committingBlockNum)

	var docs []*couchdb.CouchDoc
	if s.pendingPvtData.PvtDataDoc != nil {
		docs = append(docs, s.pendingPvtData.PvtDataDoc)
	}

	lastCommittedBlockDoc, err := s.prepareLastCommittedBlockDoc(committingBlockNum)
	if err != nil {
		return err
	}
	docs = append(docs, lastCommittedBlockDoc)

	_, err = s.db.BatchUpdateDocuments(docs)
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("writing private data to CouchDB failed [%d]", committingBlockNum))
	}

	batch := leveldbhelper.NewUpdateBatch()
	if len(s.pendingPvtData.MissingDataEntries) > 0 {
		for missingDataKey, missingDataValue := range s.pendingPvtData.MissingDataEntries {
			batch.Put([]byte(missingDataKey), []byte(missingDataValue))
		}
		if err := s.missingKeysIndexDB.WriteBatch(batch, true); err != nil {
			return err
		}
	}

	batch.Delete(common.PendingCommitKey)
	if err := s.missingKeysIndexDB.WriteBatch(batch, true); err != nil {
		return err
	}

	s.pendingPvtData = &pendingPvtData{BatchPending: false}
	s.isEmpty = false
	s.lastCommittedBlock = committingBlockNum

	logger.Debugf("Committed private data for block [%d]", committingBlockNum)
	s.performPurgeIfScheduled(committingBlockNum)
	return nil
}

func (s *store) prepareLastCommittedBlockDoc(committingBlockNum uint64) (*couchdb.CouchDoc, error) {
	_, rev, err := lookupLastBlock(s.db)
	if err != nil {
		return nil, err
	}
	lastCommittedBlockDoc, err := createLastCommittedBlockDoc(committingBlockNum, rev)
	if err != nil {
		return nil, err
	}
	return lastCommittedBlockDoc, nil
}

// Rollback implements the function in the interface `Store`
func (s *store) Rollback() error {
	if !roles.IsCommitter() {
		panic("calling Rollback on a peer that is not a committer")
	}

	if !s.pendingPvtData.BatchPending {
		return pvtdatastorage.NewErrIllegalCall("No pending batch to rollback")
	}

	s.pendingPvtData = &pendingPvtData{BatchPending: false}
	if err := s.missingKeysIndexDB.Delete(common.PendingCommitKey, true); err != nil {
		return err
	}
	return nil
}

// CommitPvtDataOfOldBlocks commits the pvtData (i.e., previously missing data) of old blocks.
// The parameter `blocksPvtData` refers a list of old block's pvtdata which are missing in the pvtstore.
// Given a list of old block's pvtData, `CommitPvtDataOfOldBlocks` performs the following four
// operations
// (1) construct dataEntries for all pvtData
// (2) construct update entries (i.e., dataEntries, expiryEntries, missingDataEntries, and
//     lastUpdatedOldBlocksList) from the above created data entries
// (3) create a db update batch from the update entries
// (4) commit the update entries to the pvtStore
func (s *store) CommitPvtDataOfOldBlocks(blocksPvtData map[uint64][]*ledger.TxPvtData) error {
	if s.isLastUpdatedOldBlocksSet {
		return pvtdatastorage.NewErrIllegalCall(`The lastUpdatedOldBlocksList is set. It means that the
		stateDB may not be in sync with the pvtStore`)
	}

	batch := leveldbhelper.NewUpdateBatch()
	docs := make([]*couchdb.CouchDoc, 0)
	// create a list of blocks' pvtData which are being stored. If this list is
	// found during the recovery, the stateDB may not be in sync with the pvtData
	// and needs recovery. In a normal flow, once the stateDB is synced, the
	// block list would be deleted.
	updatedBlksListMap := make(map[uint64]bool)
	// (1) construct dataEntries for all pvtData
	entries := common.ConstructDataEntriesFromBlocksPvtData(blocksPvtData)

	for blockNum, value := range entries {
		// (2) construct update entries (i.e., dataEntries, expiryEntries, missingDataEntries) from the above created data entries
		logger.Debugf("Constructing pvtdatastore entries for pvtData of [%d] old blocks", len(blocksPvtData))
		updateEntries, err := common.ConstructUpdateEntriesFromDataEntries(value, s.btlPolicy, s.getExpiryDataOfExpiryKey, s.getBitmapOfMissingDataKey)
		if err != nil {
			return err
		}
		// (3) create a db update batch from the update entries
		logger.Debug("Constructing update batch from pvtdatastore entries")
		batch, err = common.ConstructUpdateBatchFromUpdateEntries(updateEntries, batch)
		if err != nil {
			return err
		}
		pvtDataDoc, err := s.preparePvtDataDoc(blockNum, updateEntries)
		if err != nil {
			return err
		}
		if pvtDataDoc != nil {
			docs = append(docs, pvtDataDoc)
		}
		updatedBlksListMap[blockNum] = true
	}
	if err := s.addLastUpdatedOldBlocksList(batch, updatedBlksListMap); err != nil {
		return err
	}
	// (4) commit the update entries to the pvtStore
	logger.Debug("Committing the update batch to pvtdatastore")
	if _, err := s.db.BatchUpdateDocuments(docs); err != nil {
		return err
	}
	if err := s.missingKeysIndexDB.WriteBatch(batch, true); err != nil {
		return err
	}
	s.isLastUpdatedOldBlocksSet = true

	return nil
}

// GetLastUpdatedOldBlocksPvtData implements the function in the interface `Store`
func (s *store) GetLastUpdatedOldBlocksPvtData() (map[uint64][]*ledger.TxPvtData, error) {
	if !s.isLastUpdatedOldBlocksSet {
		return nil, nil
	}

	updatedBlksList, err := common.GetLastUpdatedOldBlocksList(s.missingKeysIndexDB)
	if err != nil {
		return nil, err
	}

	blksPvtData := make(map[uint64][]*ledger.TxPvtData)
	for _, blkNum := range updatedBlksList {
		if blksPvtData[blkNum], err = s.GetPvtDataByBlockNum(blkNum, nil); err != nil {
			return nil, err
		}
	}
	return blksPvtData, nil
}

// ResetLastUpdatedOldBlocksList implements the function in the interface `Store`
func (s *store) ResetLastUpdatedOldBlocksList() error {
	if err := common.ResetLastUpdatedOldBlocksList(s.missingKeysIndexDB); err != nil {
		return err
	}
	s.isLastUpdatedOldBlocksSet = false
	return nil
}

// GetPvtDataByBlockNum implements the function in the interface `Store`.
// If the store is empty or the last committed block number is smaller then the
// requested block number, an 'ErrOutOfRange' is thrown
func (s *store) GetPvtDataByBlockNum(blockNum uint64, filter ledger.PvtNsCollFilter) ([]*ledger.TxPvtData, error) {
	logger.Debugf("Get private data for block [%d], filter=%#v", blockNum, filter)

	if err := s.checkLastCommittedBlock(blockNum); err != nil {
		return nil, err
	}

	blockPvtDataResponse, err := retrieveBlockPvtData(s.db, blockNumberToKey(blockNum))
	if err != nil {
		_, ok := err.(*NotFoundInIndexErr)
		if ok {
			return nil, nil
		}
		return nil, err
	}

	var sortedKeys []string
	for key := range blockPvtDataResponse.Data {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	return s.getBlockPvtData(blockPvtDataResponse.Data, filter, blockNum, sortedKeys)

}

func (s *store) checkLastCommittedBlock(blockNum uint64) error {
	if roles.IsCommitter() {
		if s.isEmpty {
			return pvtdatastorage.NewErrOutOfRange("The store is empty")
		}
		if blockNum > s.lastCommittedBlock {
			return pvtdatastorage.NewErrOutOfRange(fmt.Sprintf("Last committed block=%d, block requested=%d", s.lastCommittedBlock, blockNum))
		}
	} else {
		lastCommittedBlock, _, err := lookupLastBlock(s.db)
		if err != nil {
			return err
		}
		if lastCommittedBlock == 0 {
			return pvtdatastorage.NewErrOutOfRange("The store is empty")
		}
		if blockNum > lastCommittedBlock {
			return pvtdatastorage.NewErrOutOfRange(fmt.Sprintf("Last committed block=%d, block requested=%d", s.lastCommittedBlock, blockNum))
		}
	}
	return nil
}

// ProcessCollsEligibilityEnabled implements the function in the interface `Store`
func (s *store) ProcessCollsEligibilityEnabled(committingBlk uint64, nsCollMap map[string][]string) error {
	return common.ProcessCollsEligibilityEnabled(committingBlk, nsCollMap, s.collElgProc, s.missingKeysIndexDB)
}

// LastCommittedBlockHeight implements the function in the interface `Store`
func (s *store) LastCommittedBlockHeight() (uint64, error) {
	if s.isEmpty {
		return 0, nil
	}
	return s.lastCommittedBlock + 1, nil
}

// HasPendingBatch implements the function in the interface `Store`
func (s *store) HasPendingBatch() (bool, error) {
	return s.pendingPvtData.BatchPending, nil
}

// IsEmpty implements the function in the interface `Store`
func (s *store) IsEmpty() (bool, error) {
	return s.isEmpty, nil
}

// Shutdown implements the function in the interface `Store`
func (s *store) Shutdown() {
	// do nothing
}

func (s *store) preparePvtDataDoc(blockNum uint64, updateEntries *common.EntriesForPvtDataOfOldBlocks) (*couchdb.CouchDoc, error) {
	dataEntries, expiryEntries, rev, err := s.retrieveBlockPvtEntries(blockNum)
	if err != nil {
		return nil, err
	}
	pvtDataDoc, err := createPvtDataCouchDoc(s.prepareStoreEntries(updateEntries, dataEntries, expiryEntries), blockNum, rev)
	if err != nil {
		return nil, err
	}
	return pvtDataDoc, nil
}

func (s *store) retrieveBlockPvtEntries(blockNum uint64) ([]*common.DataEntry, []*common.ExpiryEntry, string, error) {
	rev := ""
	var dataEntries []*common.DataEntry
	var expiryEntries []*common.ExpiryEntry
	blockPvtDataResponse, err := retrieveBlockPvtData(s.db, blockNumberToKey(blockNum))
	if err != nil {
		_, ok := err.(*NotFoundInIndexErr)
		if ok {
			return nil, nil, "", nil
		}
		return nil, nil, "", err
	}

	if blockPvtDataResponse != nil {
		rev = blockPvtDataResponse.Rev
		for key := range blockPvtDataResponse.Data {
			dataKeyBytes, errDecodeString := hex.DecodeString(key)
			if errDecodeString != nil {
				return nil, nil, "", errDecodeString
			}
			dataKey := common.DecodeDatakey(dataKeyBytes)
			dataValue, err := common.DecodeDataValue(blockPvtDataResponse.Data[key])
			if err != nil {
				return nil, nil, "", err
			}
			dataEntries = append(dataEntries, &common.DataEntry{Key: dataKey, Value: dataValue})
		}
		for key := range blockPvtDataResponse.Expiry {
			expiryKeyBytes, err := hex.DecodeString(key)
			if err != nil {
				return nil, nil, "", err
			}
			expiryKey := common.DecodeExpiryKey(expiryKeyBytes)
			expiryValue, err := common.DecodeExpiryValue(blockPvtDataResponse.Expiry[key])
			if err != nil {
				return nil, nil, "", err
			}
			expiryEntries = append(expiryEntries, &common.ExpiryEntry{Key: expiryKey, Value: expiryValue})
		}
	}
	return dataEntries, expiryEntries, rev, nil
}

func (s *store) addLastUpdatedOldBlocksList(batch *leveldbhelper.UpdateBatch, updatedBlksListMap map[uint64]bool) error {
	var updatedBlksList lastUpdatedOldBlocksList
	for blkNum := range updatedBlksListMap {
		updatedBlksList = append(updatedBlksList, blkNum)
	}

	// better to store as sorted list
	sort.SliceStable(updatedBlksList, func(i, j int) bool {
		return updatedBlksList[i] < updatedBlksList[j]
	})

	buf := proto.NewBuffer(nil)
	if err := buf.EncodeVarint(uint64(len(updatedBlksList))); err != nil {
		return err
	}
	for _, blkNum := range updatedBlksList {
		if err := buf.EncodeVarint(blkNum); err != nil {
			return err
		}
	}

	batch.Put(common.LastUpdatedOldBlocksKey, buf.Bytes())
	return nil
}

func (s *store) getBlockPvtData(results map[string][]byte, filter ledger.PvtNsCollFilter, blockNum uint64, sortedKeys []string) ([]*ledger.TxPvtData, error) {
	var blockPvtdata []*ledger.TxPvtData
	var currentTxNum uint64
	var currentTxWsetAssember *common.TxPvtdataAssembler
	firstItr := true

	for _, key := range sortedKeys {
		dataKeyBytes, err := hex.DecodeString(key)
		if err != nil {
			return nil, err
		}
		if common.V11Format(dataKeyBytes) {
			return v11RetrievePvtdata(results, filter)
		}
		dataValueBytes := results[key]
		dataKey := common.DecodeDatakey(dataKeyBytes)
		expired, err := s.checkIsExpired(dataKey, filter, s.lastCommittedBlock)
		if err != nil {
			return nil, err
		}
		if expired {
			continue
		}

		dataValue, err := common.DecodeDataValue(dataValueBytes)
		if err != nil {
			return nil, err
		}

		if firstItr {
			currentTxNum = dataKey.TxNum
			currentTxWsetAssember = common.NewTxPvtdataAssembler(blockNum, currentTxNum)
			firstItr = false
		}

		if dataKey.TxNum != currentTxNum {
			blockPvtdata = append(blockPvtdata, currentTxWsetAssember.GetTxPvtdata())
			currentTxNum = dataKey.TxNum
			currentTxWsetAssember = common.NewTxPvtdataAssembler(blockNum, currentTxNum)
		}
		currentTxWsetAssember.Add(dataKey.Ns, dataValue)
	}
	if currentTxWsetAssember != nil {
		blockPvtdata = append(blockPvtdata, currentTxWsetAssember.GetTxPvtdata())
	}
	return blockPvtdata, nil
}

func (s *store) checkIsExpired(dataKey *common.DataKey, filter ledger.PvtNsCollFilter, lastCommittedBlock uint64) (bool, error) {
	expired, err := common.IsExpired(dataKey.NsCollBlk, s.btlPolicy, lastCommittedBlock)
	if err != nil {
		return false, err
	}
	if expired || !common.PassesFilter(dataKey, filter) {
		return true, nil
	}
	return false, nil
}

// InitLastCommittedBlock implements the function in the interface `Store`
func (s *store) InitLastCommittedBlock(blockNum uint64) error {
	if !(s.isEmpty && !s.pendingPvtData.BatchPending) {
		return pvtdatastorage.NewErrIllegalCall("The private data store is not empty. InitLastCommittedBlock() function call is not allowed")
	}
	s.isEmpty = false
	s.lastCommittedBlock = blockNum

	_, rev, err := lookupLastBlock(s.db)
	if err != nil {
		return err
	}
	lastCommittedBlockDoc, err := createLastCommittedBlockDoc(s.lastCommittedBlock, rev)
	if err != nil {
		return err
	}
	_, err = s.db.BatchUpdateDocuments([]*couchdb.CouchDoc{lastCommittedBlockDoc})
	if err != nil {
		return err
	}

	logger.Debugf("InitLastCommittedBlock set to block [%d]", blockNum)
	return nil
}

//GetMissingPvtDataInfoForMostRecentBlocks implements the function in the interface `Store`
func (s *store) GetMissingPvtDataInfoForMostRecentBlocks(maxBlock int) (ledger.MissingPvtDataInfo, error) {
	return common.GetMissingPvtDataInfoForMostRecentBlocks(maxBlock, s.lastCommittedBlock, s.btlPolicy, s.missingKeysIndexDB)
}

func v11RetrievePvtdata(pvtDataResults map[string][]byte, filter ledger.PvtNsCollFilter) ([]*ledger.TxPvtData, error) {
	var blkPvtData []*ledger.TxPvtData
	for key, val := range pvtDataResults {
		pvtDatum, err := common.V11DecodeKV([]byte(key), val, filter)
		if err != nil {
			return nil, err
		}
		blkPvtData = append(blkPvtData, pvtDatum)
	}
	return blkPvtData, nil
}

func (s *store) getExpiryDataOfExpiryKey(expiryKey *common.ExpiryKey) (*common.ExpiryData, error) {
	var expiryEntriesMap map[string][]byte
	var err error
	if expiryEntriesMap, err = s.getExpiryEntriesDB(expiryKey.CommittingBlk); err != nil {
		return nil, err
	}
	v := expiryEntriesMap[hex.EncodeToString(common.EncodeExpiryKey(expiryKey))]
	if v == nil {
		return nil, nil
	}
	return common.DecodeExpiryValue(v)
}

func (s *store) getExpiryEntriesDB(blockNum uint64) (map[string][]byte, error) {
	blockPvtData, err := retrieveBlockPvtData(s.db, blockNumberToKey(blockNum))
	if err != nil {
		return nil, err
	}
	return blockPvtData.Expiry, nil
}

func (s *store) getBitmapOfMissingDataKey(missingDataKey *common.MissingDataKey) (*bitset.BitSet, error) {
	var v []byte
	var err error
	if v, err = s.missingKeysIndexDB.Get(common.EncodeMissingDataKey(missingDataKey)); err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	return common.DecodeMissingDataValue(v)
}

func (s *store) prepareStoreEntries(updateEntries *common.EntriesForPvtDataOfOldBlocks, dataEntries []*common.DataEntry, expiryEntries []*common.ExpiryEntry) *common.StoreEntries {
	if dataEntries == nil {
		dataEntries = make([]*common.DataEntry, 0)
	}
	if expiryEntries == nil {
		expiryEntries = make([]*common.ExpiryEntry, 0)
	}
	for k, v := range updateEntries.DataEntries {
		k := k
		dataEntries = append(dataEntries, &common.DataEntry{Key: &k, Value: v})
	}
	for k, v := range updateEntries.ExpiryEntries {
		k := k
		expiryEntries = append(expiryEntries, &common.ExpiryEntry{Key: &k, Value: v})
	}
	return &common.StoreEntries{DataEntries: dataEntries, ExpiryEntries: expiryEntries}
}

func (s *store) hasPendingCommit() (*pendingPvtData, error) {
	var v []byte
	var err error
	if v, err = s.missingKeysIndexDB.Get(common.PendingCommitKey); err != nil {
		return nil, err
	}
	if v != nil {
		var pPvtData pendingPvtData
		if err := json.Unmarshal(v, &pPvtData); err != nil {
			return nil, err
		}
		return &pPvtData, nil
	}
	return &pendingPvtData{BatchPending: false}, nil

}

func (s *store) savePendingKey() error {
	bytes, err := json.Marshal(s.pendingPvtData)
	if err != nil {
		return err
	}
	if err := s.missingKeysIndexDB.Put(common.PendingCommitKey, bytes, true); err != nil {
		return err
	}
	return nil
}

func (s *store) perparePendingMissingDataEntries(mssingDataEntries map[common.MissingDataKey]*bitset.BitSet) (map[string]string, error) {
	pendingMissingDataEntries := make(map[string]string)
	for missingDataKey, missingDataValue := range mssingDataEntries {
		missingDataKey := missingDataKey
		keyBytes := common.EncodeMissingDataKey(&missingDataKey)
		valBytes, err := common.EncodeMissingDataValue(missingDataValue)
		if err != nil {
			return nil, err
		}
		pendingMissingDataEntries[string(keyBytes)] = string(valBytes)
	}
	return pendingMissingDataEntries, nil
}

func (s *store) nextBlockNum() uint64 {
	if s.isEmpty {
		return 0
	}
	return s.lastCommittedBlock + 1
}

func (s *store) performPurgeIfScheduled(latestCommittedBlk uint64) {
	if latestCommittedBlk%uint64(s.ledgerConfig.PrivateData.PurgeInterval) != 0 {
		return
	}
	go func() {
		s.purgerLock.Lock()
		logger.Debugf("Purger started: Purging expired private data till block number [%d]", latestCommittedBlk)
		defer s.purgerLock.Unlock()
		err := s.purgeExpiredData(latestCommittedBlk)
		if err != nil {
			logger.Warningf("Could not purge data from pvtdata store:%s", err)
		}
		logger.Debug("Purger finished")
	}()
}

func (s *store) purgeExpiredData(maxBlkNum uint64) error {
	pvtData, err := retrieveBlockExpiryData(s.db, blockNumberToKey(maxBlkNum))
	if err != nil {
		return err
	}
	if len(pvtData) == 0 {
		return nil
	}

	docs, batch, err := s.prepareExpiredData(pvtData, maxBlkNum)
	if err != nil {
		return err
	}
	if len(docs) > 0 {
		_, err := s.db.BatchUpdateDocuments(docs)
		if err != nil {
			return errors.WithMessage(err, fmt.Sprintf("BatchUpdateDocuments failed for [%d] documents", len(docs)))
		}
	}
	if err := s.missingKeysIndexDB.WriteBatch(batch, false); err != nil {
		return err
	}

	logger.Infof("[%s] - Entries purged from private data storage till block number [%d]", s.ledgerid, maxBlkNum)
	return nil
}

func (s *store) prepareExpiredData(pvtData []*blockPvtDataResponse, maxBlkNum uint64) ([]*couchdb.CouchDoc, *leveldbhelper.UpdateBatch, error) {
	batch := leveldbhelper.NewUpdateBatch()
	var docs []*couchdb.CouchDoc
	for _, data := range pvtData {
		expBlkNums := make([]string, 0)
		for key, value := range data.Expiry {
			expiryKeyBytes, err := hex.DecodeString(key)
			if err != nil {
				return nil, nil, err
			}
			expiryKey := common.DecodeExpiryKey(expiryKeyBytes)
			if expiryKey.ExpiringBlk <= maxBlkNum {
				expiryValue, err := common.DecodeExpiryValue(value)
				if err != nil {
					return nil, nil, err
				}
				dataKeys, missingDataKeys := common.DeriveKeys(&common.ExpiryEntry{Key: expiryKey, Value: expiryValue})
				for _, dataKey := range dataKeys {
					delete(data.Data, hex.EncodeToString(common.EncodeDataKey(dataKey)))
				}
				for _, missingDataKey := range missingDataKeys {
					batch.Delete(common.EncodeMissingDataKey(missingDataKey))
				}
			} else {
				expBlkNums = append(expBlkNums, blockNumberToKey(expiryKey.ExpiringBlk))
			}
		}
		if len(data.Data) == 0 {
			data.Deleted = true
		}
		data.ExpiringBlkNums = expBlkNums
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			return nil, nil, err
		}
		docs = append(docs, &couchdb.CouchDoc{JSONValue: jsonBytes})
	}

	return docs, batch, nil

}
