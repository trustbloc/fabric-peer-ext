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
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/ledger/util/leveldbhelper"
	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/core/ledger"
	couchdb "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"
	"github.com/hyperledger/fabric/core/ledger/pvtdatapolicy"
	"github.com/hyperledger/fabric/core/ledger/pvtdatastorage"
	xstorageapi "github.com/hyperledger/fabric/extensions/storage/api"
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
	pvtDataConfig            *pvtdatastorage.PrivateDataConfig
}

type store struct {
	ledgerid           string
	btlPolicy          pvtdatapolicy.BTLPolicy
	db                 couchDB
	lastCommittedBlock uint64
	purgerLock         *sync.Mutex
	collElgProcSync    *common.CollElgProcSync
	// missing keys db
	missingKeysIndexDB common.DBHandle
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
	pvtDataConfig             *pvtdatastorage.PrivateDataConfig

	deprioritizedDataReconcilerInterval time.Duration
	accessDeprioMissingDataAfter        time.Time
}

type pendingPvtData struct {
	PvtDataDoc              *couchdb.CouchDoc `json:"pvtDataDoc"`
	ElgMissingDataEntries   map[string]string `json:"elgMissingDataEntries"`
	InelgMissingDataEntries map[string]string `json:"inelgMissingDataEntries"`
	BatchPending            bool              `json:"batchPending"`
}

// lastUpdatedOldBlocksList keeps the list of last updated blocks
// and is stored as the value of lastUpdatedOldBlocksKey (defined in kv_encoding.go)
type lastUpdatedOldBlocksList []uint64

//////// Provider functions  /////////////
//////////////////////////////////////////

// NewProvider instantiates a private data storage provider backed by CouchDB
func NewProvider(conf *pvtdatastorage.PrivateDataConfig, ledgerconfig *ledger.Config) (xstorageapi.PrivateDataProvider, error) {
	logger.Debugf("constructing CouchDB private data storage provider")
	couchDBConfig := ledgerconfig.StateDBConfig.CouchDB

	return newProviderWithDBDef(couchDBConfig, conf)
}

func newProviderWithDBDef(couchDBConfig *ledger.CouchDBConfig, conf *pvtdatastorage.PrivateDataConfig) (xstorageapi.PrivateDataProvider, error) {
	couchInstance, err := couchdb.CreateCouchInstance(couchDBConfig, &disabled.Provider{})
	if err != nil {
		return nil, errors.WithMessage(err, "obtaining CouchDB instance failed")
	}

	dbPath := conf.StorePath
	missingKeysIndexProvider, err := leveldbhelper.NewProvider(&leveldbhelper.Conf{DBPath: dbPath})
	if err != nil {
		return nil, err
	}

	return &provider{
		couchInstance:            couchInstance,
		missingKeysIndexProvider: missingKeysIndexProvider,
		pvtDataConfig:            conf,
	}, nil
}

// OpenStore returns a handle to a store
func (p *provider) OpenStore(ledgerid string) (xstorageapi.PrivateDataStore, error) {
	// Create couchdb
	pvtDataStoreDBName := couchdb.ConstructBlockchainDBName(strings.ToLower(ledgerid), pvtDataStoreName)
	if roles.IsCommitter() {
		db, err := couchdb.CreateCouchDatabase(p.couchInstance, pvtDataStoreDBName)
		if err != nil {
			return nil, errors.Wrapf(err, "createCouchDatabase failed")
		}
		err = createPvtDataCouchDB(db)
		if err != nil {
			return nil, errors.Wrapf(err, "createPvtDataCouchDB failed")
		}
		// Create missing pvt keys index in leveldb
		missingKeysIndexDB := p.missingKeysIndexProvider.GetDBHandle(ledgerid)

		purgerLock := &sync.Mutex{}
		s := &store{db: db, ledgerid: ledgerid,
			collElgProcSync:                     common.NewCollElgProcSync(purgerLock, missingKeysIndexDB, p.pvtDataConfig.BatchesInterval, p.pvtDataConfig.MaxBatchSize),
			purgerLock:                          purgerLock,
			missingKeysIndexDB:                  missingKeysIndexDB,
			pvtDataConfig:                       p.pvtDataConfig,
			deprioritizedDataReconcilerInterval: p.pvtDataConfig.DeprioritizedDataReconcilerInterval,
			accessDeprioMissingDataAfter:        time.Now().Add(p.pvtDataConfig.DeprioritizedDataReconcilerInterval),
		}

		if errInitState := s.initState(); errInitState != nil {
			return nil, errInitState
		}
		s.collElgProcSync.LaunchCollElgProc()

		logger.Debugf("Pvtdata store opened. Initial state: isEmpty [%t], lastCommittedBlock [%d]",
			s.isEmpty, s.lastCommittedBlock)

		return s, nil
	}
	db, err := couchdb.NewCouchDatabase(p.couchInstance, pvtDataStoreDBName)
	if err != nil {
		return nil, errors.Wrapf(err, "newCouchDatabase failed")
	}
	err = getPvtDataCouchInstance(db, db.DBName)
	if err != nil {
		return nil, errors.Wrapf(err, "getPvtDataCouchInstance failed")
	}
	s := &store{db: db, ledgerid: ledgerid}
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
		return errors.Wrapf(err, "lookupLastBlock failed")
	}
	s.isEmpty = true
	if lastCommittedBlock != 0 {
		s.lastCommittedBlock = lastCommittedBlock
		s.isEmpty = false
	}

	if blist, err = common.GetLastUpdatedOldBlocksList(s.missingKeysIndexDB); err != nil {
		return errors.Wrap(err, "getLastUpdatedOldBlocksList failed")
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
func (s *store) Commit(blockNum uint64, pvtData []*ledger.TxPvtData, missingPvtData ledger.TxMissingPvtDataMap) error {
	if !roles.IsCommitter() {
		panic("calling Prepare on a peer that is not a committer")
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

	pendingPvtData, err := s.prepareMissingKeys(pvtDataDoc, storeEntries)
	if err != nil {
		return err
	}
	logger.Debugf("Saved %d private data write sets for block [%d]", len(pvtData), blockNum)

	committingBlockNum := s.nextBlockNum()
	logger.Debugf("Committing private data for block [%d]", committingBlockNum)

	var docs []*couchdb.CouchDoc
	if pendingPvtData.PvtDataDoc != nil {
		docs = append(docs, pendingPvtData.PvtDataDoc)
	}

	lastCommittedBlockDoc, err := s.prepareLastCommittedBlockDoc(committingBlockNum)
	if err != nil {
		return errors.WithMessage(err, "prepareLastCommittedBlockDoc failed")
	}
	docs = append(docs, lastCommittedBlockDoc)

	_, err = s.db.BatchUpdateDocuments(docs)
	if err != nil {
		return errors.WithMessagef(err, "writing private data to CouchDB failed [%d]", committingBlockNum)
	}

	err = s.updateMissingKeys(pendingPvtData)
	if err != nil {
		return err
	}

	s.isEmpty = false
	s.lastCommittedBlock = committingBlockNum

	logger.Debugf("Committed private data for block [%d]", committingBlockNum)
	s.performPurgeIfScheduled(committingBlockNum)
	return nil
}

func (s *store) prepareMissingKeys(pvtDataDoc *couchdb.CouchDoc, storeEntries *common.StoreEntries) (*pendingPvtData, error) {
	pendingPvtData := &pendingPvtData{BatchPending: true}
	if pvtDataDoc != nil || len(storeEntries.ElgMissingDataEntries) > 0 || len(storeEntries.InelgMissingDataEntries) > 0 {
		var err error
		pendingPvtData.ElgMissingDataEntries, err = s.preparePendingMissingDataEntries(storeEntries.ElgMissingDataEntries, common.EncodeElgPrioMissingDataKey)
		if err != nil {
			return nil, err
		}

		pendingPvtData.InelgMissingDataEntries, err = s.preparePendingMissingDataEntries(storeEntries.InelgMissingDataEntries, common.EncodeInelgMissingDataKey)
		if err != nil {
			return nil, err
		}

		pendingPvtData.PvtDataDoc = pvtDataDoc
		bytes, err := json.Marshal(pendingPvtData)
		if err != nil {
			return nil, err
		}
		err = s.missingKeysIndexDB.Put(common.PendingCommitKey, bytes, true)
		if err != nil {
			return nil, errors.Wrapf(err, "put in leveldb failed for key PendingCommitKey")
		}
	}
	return pendingPvtData, nil
}

func (s *store) updateMissingKeys(pendingPvtData *pendingPvtData) error {
	batch := s.missingKeysIndexDB.NewUpdateBatch()

	for missingDataKey, missingDataValue := range pendingPvtData.ElgMissingDataEntries {
		batch.Put([]byte(missingDataKey), []byte(missingDataValue))
	}

	for missingDataKey, missingDataValue := range pendingPvtData.InelgMissingDataEntries {
		batch.Put([]byte(missingDataKey), []byte(missingDataValue))
	}

	batch.Delete(common.PendingCommitKey)
	if err := s.missingKeysIndexDB.WriteBatch(batch, true); err != nil {
		return errors.Wrap(err, "WriteBatch failed")
	}

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

// GetLastUpdatedOldBlocksPvtData implements the function in the interface `Store`
func (s *store) GetLastUpdatedOldBlocksPvtData() (map[uint64][]*ledger.TxPvtData, error) {
	if !s.isLastUpdatedOldBlocksSet {
		return nil, nil
	}

	updatedBlksList, err := common.GetLastUpdatedOldBlocksList(s.missingKeysIndexDB)
	if err != nil {
		return nil, errors.Wrapf(err, "GetLastUpdatedOldBlocksList failed")
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
		return errors.Wrapf(err, "ResetLastUpdatedOldBlocksList failed")
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
			return errors.Wrapf(err, "lookupLastBlock failed")
		}
		if lastCommittedBlock == 0 {
			return pvtdatastorage.NewErrOutOfRange("The store is empty")
		}
		if blockNum > lastCommittedBlock {
			return pvtdatastorage.NewErrOutOfRange(fmt.Sprintf("Last committed block=%d, block requested=%d", lastCommittedBlock, blockNum))
		}
	}
	return nil
}

// ProcessCollsEligibilityEnabled implements the function in the interface `Store`
func (s *store) ProcessCollsEligibilityEnabled(committingBlk uint64, nsCollMap map[string][]string) error {
	return common.ProcessCollsEligibilityEnabled(committingBlk, nsCollMap, s.collElgProcSync, s.missingKeysIndexDB)
}

// LastCommittedBlockHeight implements the function in the interface `Store`
func (s *store) LastCommittedBlockHeight() (uint64, error) {
	if s.isEmpty {
		return 0, nil
	}
	return s.lastCommittedBlock + 1, nil
}

// Shutdown implements the function in the interface `Store`
func (s *store) Shutdown() {
	// do nothing
}

func (s *store) preparePvtDataDoc(blockNum uint64, updateEntries *common.DataAndExpiryEntries) (*couchdb.CouchDoc, error) {
	dataEntries, expiryEntries, rev, err := s.retrieveBlockPvtEntries(blockNum)
	if err != nil {
		return nil, errors.WithMessage(err, "retrieveBlockPvtEntries failed")
	}
	pvtDataDoc, err := createPvtDataCouchDoc(s.prepareStoreEntries(updateEntries, dataEntries, expiryEntries), blockNum, rev)
	if err != nil {
		return nil, err
	}
	return pvtDataDoc, nil
}

func (s *store) retrieveBlockPvtEntries(blockNum uint64) ([]*common.DataEntry, []*common.ExpiryEntry, string, error) {
	rev := ""
	blockPvtDataResponse, err := retrieveBlockPvtData(s.db, blockNumberToKey(blockNum))
	if err != nil {
		_, ok := err.(*NotFoundInIndexErr)
		if ok {
			return nil, nil, "", nil
		}
		return nil, nil, "", err
	}

	if blockPvtDataResponse == nil {
		return nil, nil, rev, nil
	}

	rev = blockPvtDataResponse.Rev
	dataEntries, err := s.retrieveBlockPvtDataEntries(blockPvtDataResponse)
	if err != nil {
		return nil, nil, "", err
	}

	expiryEntries, err := s.retrieveBlockPvtExpiryEntries(blockPvtDataResponse)
	if err != nil {
		return nil, nil, "", err
	}

	return dataEntries, expiryEntries, rev, nil
}

func (s *store) retrieveBlockPvtDataEntries(blockPvtDataResponse *blockPvtDataResponse) ([]*common.DataEntry, error) {
	var dataEntries []*common.DataEntry
	for key := range blockPvtDataResponse.Data {
		dataKeyBytes, errDecodeString := hex.DecodeString(key)
		if errDecodeString != nil {
			return nil, errDecodeString
		}
		dataKey, err := common.DecodeDatakey(dataKeyBytes)
		if err != nil {
			return nil, err
		}

		dataValue, err := common.DecodeDataValue(blockPvtDataResponse.Data[key])
		if err != nil {
			return nil, err
		}
		dataEntries = append(dataEntries, &common.DataEntry{Key: dataKey, Value: dataValue})
	}
	return dataEntries, nil
}

func (s *store) retrieveBlockPvtExpiryEntries(blockPvtDataResponse *blockPvtDataResponse) ([]*common.ExpiryEntry, error) {
	var expiryEntries []*common.ExpiryEntry
	for key := range blockPvtDataResponse.Expiry {
		expiryKeyBytes, err := hex.DecodeString(key)
		if err != nil {
			return nil, err
		}
		expiryKey, err := common.DecodeExpiryKey(expiryKeyBytes)
		if err != nil {
			return nil, err
		}

		expiryValue, err := common.DecodeExpiryValue(blockPvtDataResponse.Expiry[key])
		if err != nil {
			return nil, err
		}
		expiryEntries = append(expiryEntries, &common.ExpiryEntry{Key: expiryKey, Value: expiryValue})
	}
	return expiryEntries, nil
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
	return newBlockPvtDataAssembler(results, filter, blockNum, s.lastCommittedBlock, sortedKeys, s.checkIsExpired).assemble()
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

// GetMissingPvtDataInfoForMostRecentBlocks returns the missing private data information for the
// most recent `maxBlock` blocks which miss at least a private data of a eligible collection.
func (s *store) GetMissingPvtDataInfoForMostRecentBlocks(maxBlock int) (ledger.MissingPvtDataInfo, error) {
	// we assume that this function would be called by the gossip only after processing the
	// last retrieved missing pvtdata info and committing the same.
	if maxBlock < 1 {
		return nil, nil
	}

	if time.Now().After(s.accessDeprioMissingDataAfter) {
		s.accessDeprioMissingDataAfter = time.Now().Add(s.deprioritizedDataReconcilerInterval)
		logger.Debug("fetching missing pvtdata entries from the deprioritized list")
		return common.GetMissingPvtDataInfoForMostRecentBlocks(common.ElgDeprioritizedMissingDataGroup, maxBlock, s.lastCommittedBlock, s.btlPolicy, s.missingKeysIndexDB)
	}

	logger.Debug("fetching missing pvtdata entries from the prioritized list")
	return common.GetMissingPvtDataInfoForMostRecentBlocks(common.ElgPrioritizedMissingDataGroup, maxBlock, s.lastCommittedBlock, s.btlPolicy, s.missingKeysIndexDB)
}

func (s *store) getExpiryDataOfExpiryKey(expiryKey common.ExpiryKey) (*common.ExpiryData, error) {
	var expiryEntriesMap map[string][]byte
	var err error
	if expiryEntriesMap, err = s.getExpiryEntriesDB(expiryKey.CommittingBlk); err != nil {
		return nil, errors.WithMessage(err, "getExpiryEntriesDB failed")
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
		if _, ok := err.(*NotFoundInIndexErr); ok {
			return nil, nil
		}
		return nil, err
	}
	return blockPvtData.Expiry, nil
}

func (s *store) prepareStoreEntries(updateEntries *common.DataAndExpiryEntries, dataEntries []*common.DataEntry, expiryEntries []*common.ExpiryEntry) *common.StoreEntries {
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

func (s *store) preparePendingMissingDataEntries(mssingDataEntries map[common.MissingDataKey]*bitset.BitSet, encodeKey func(key *common.MissingDataKey) []byte) (map[string]string, error) {
	pendingMissingDataEntries := make(map[string]string)
	for missingDataKey, missingDataValue := range mssingDataEntries {
		missingDataKey := missingDataKey
		keyBytes := encodeKey(&missingDataKey)
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
	if latestCommittedBlk%uint64(s.pvtDataConfig.PurgeInterval) != 0 {
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
	batch := s.missingKeysIndexDB.NewUpdateBatch()
	var docs []*couchdb.CouchDoc
	for _, data := range pvtData {
		doc, err := s.getExpiredDataDoc(data, batch, maxBlkNum)
		if err != nil {
			return nil, nil, err
		}
		docs = append(docs, doc)
	}

	return docs, batch, nil

}

func (s *store) getExpiredDataDoc(data *blockPvtDataResponse, batch *leveldbhelper.UpdateBatch, maxBlkNum uint64) (*couchdb.CouchDoc, error) {
	expBlkNums := make([]string, 0)

	for key, value := range data.Expiry {
		expiryKeyBytes, err := hex.DecodeString(key)
		if err != nil {
			return nil, err
		}

		expiryKey, err := common.DecodeExpiryKey(expiryKeyBytes)
		if err != nil {
			return nil, err
		}

		if expiryKey.ExpiringBlk <= maxBlkNum {
			expiryValue, err := common.DecodeExpiryValue(value)
			if err != nil {
				return nil, err
			}

			dataKeys, missingDataKeys := common.DeriveKeys(&common.ExpiryEntry{Key: expiryKey, Value: expiryValue})

			for _, dataKey := range dataKeys {
				delete(data.Data, hex.EncodeToString(common.EncodeDataKey(dataKey)))
			}

			for _, missingDataKey := range missingDataKeys {
				batch.Delete(common.EncodeElgPrioMissingDataKey(missingDataKey))
				batch.Delete(common.EncodeElgDeprioMissingDataKey(missingDataKey))
				batch.Delete(common.EncodeInelgMissingDataKey(missingDataKey))
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
		return nil, err
	}

	return &couchdb.CouchDoc{JSONValue: jsonBytes}, nil
}
