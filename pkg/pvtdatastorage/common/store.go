/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"sync/atomic"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/ledger/rwset"
	"github.com/hyperledger/fabric/common/ledger/util/leveldbhelper"
	"github.com/hyperledger/fabric/core/ledger"
	couchdb "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"
	"github.com/hyperledger/fabric/core/ledger/pvtdatapolicy"
	"github.com/hyperledger/fabric/core/ledger/pvtdatastorage"
	"github.com/pkg/errors"
	"github.com/willf/bitset"
)

// todo add pinning script to include copied code into this file, original file from fabric is found in fabric/core/ledger/pvtdatastorage/store_imp.go
// todo below functions are originally unexported, the pinning script must capitalize these functions to export them

// DBHandle is an handle to a named db
type DBHandle interface {
	WriteBatch(batch *leveldbhelper.UpdateBatch, sync bool) error
	Delete(key []byte, sync bool) error
	Get(key []byte) ([]byte, error)
	GetIterator(startKey []byte, endKey []byte) (*leveldbhelper.Iterator, error)
	Put(key []byte, value []byte, sync bool) error
	NewUpdateBatch() *leveldbhelper.UpdateBatch
}

type DataEntry struct {
	Key   *DataKey
	Value *rwset.CollectionPvtReadWriteSet
}

type ExpiryEntry struct {
	Key   *ExpiryKey
	Value *ExpiryData
}

type ExpiryKey struct {
	ExpiringBlk   uint64
	CommittingBlk uint64
}

type NsCollBlk struct {
	Ns, Coll string
	BlkNum   uint64
}

type DataKey struct {
	NsCollBlk
	TxNum uint64
}

type MissingDataKey struct {
	NsCollBlk
}

type StoreEntries struct {
	DataEntries             []*DataEntry
	ExpiryEntries           []*ExpiryEntry
	ElgMissingDataEntries   map[MissingDataKey]*bitset.BitSet
	InelgMissingDataEntries map[MissingDataKey]*bitset.BitSet
}

type preparePvtDataDoc func(blockNum uint64, updateEntries *DataAndExpiryEntries) (*couchdb.CouchDoc, error)

type entriesForPvtDataOfOldBlocks struct {
	dataEntries                     map[DataKey]*rwset.CollectionPvtReadWriteSet
	expiryEntries                   map[ExpiryKey]*ExpiryData
	prioritizedMissingDataEntries   map[NsCollBlk]*bitset.BitSet
	deprioritizedMissingDataEntries map[NsCollBlk]*bitset.BitSet
	preparePvtDataDoc               preparePvtDataDoc
}

func newEntriesForPvtDataOfOldBlocks(preparePvtDataDoc preparePvtDataDoc) *entriesForPvtDataOfOldBlocks {
	return &entriesForPvtDataOfOldBlocks{
		dataEntries:                     make(map[DataKey]*rwset.CollectionPvtReadWriteSet),
		expiryEntries:                   make(map[ExpiryKey]*ExpiryData),
		prioritizedMissingDataEntries:   make(map[NsCollBlk]*bitset.BitSet),
		deprioritizedMissingDataEntries: make(map[NsCollBlk]*bitset.BitSet),
		preparePvtDataDoc:               preparePvtDataDoc,
	}
}

type DataAndExpiryEntries struct {
	DataEntries   map[DataKey]*rwset.CollectionPvtReadWriteSet
	ExpiryEntries map[ExpiryKey]*ExpiryData
}

func (e *entriesForPvtDataOfOldBlocks) dataAndExpiryEntriesByBlock() map[uint64]*DataAndExpiryEntries {
	m := make(map[uint64]*DataAndExpiryEntries)

	logger.Infof("Examining dataEntries: %s and ExpiringEntries: %s", e.dataEntries, e.expiryEntries)

	for key, value := range e.dataEntries {
		entries, ok := m[key.BlkNum]
		if !ok {
			entries = &DataAndExpiryEntries{
				DataEntries:   make(map[DataKey]*rwset.CollectionPvtReadWriteSet),
				ExpiryEntries: make(map[ExpiryKey]*ExpiryData),
			}
			m[key.BlkNum] = entries
		}

		entries.DataEntries[key] = value
	}

	for key, value := range e.expiryEntries {
		entries, ok := m[key.CommittingBlk]
		if !ok {
			entries = &DataAndExpiryEntries{}
			m[key.CommittingBlk] = entries
		}

		entries.ExpiryEntries[key] = value
	}

	logger.Infof("Returning dataAndExpiryEntriesByBlock: %s", m)

	return m
}

func (e *entriesForPvtDataOfOldBlocks) addDataAndExpiryEntriesTo(batch *couchDocs) error {
	for blockNum, entries := range e.dataAndExpiryEntriesByBlock() {
		doc, err := e.preparePvtDataDoc(blockNum, entries)
		if err != nil {
			return err
		}

		batch.Put(doc)
	}

	return nil
}

func (e *entriesForPvtDataOfOldBlocks) addElgPrioMissingDataEntriesTo(batch *leveldbhelper.UpdateBatch) error {
	var key, val []byte
	var err error

	for nsCollBlk, missingData := range e.prioritizedMissingDataEntries {
		missingKey := &MissingDataKey{
			NsCollBlk: nsCollBlk,
		}
		key = EncodeElgPrioMissingDataKey(missingKey)

		if missingData.None() {
			batch.Delete(key)
			continue
		}

		if val, err = EncodeMissingDataValue(missingData); err != nil {
			return errors.Wrap(err, "error while encoding missing data bitmap")
		}
		batch.Put(key, val)
	}
	return nil
}

func (e *entriesForPvtDataOfOldBlocks) addElgDeprioMissingDataEntriesTo(batch *leveldbhelper.UpdateBatch) error {
	var key, val []byte
	var err error

	for nsCollBlk, missingData := range e.deprioritizedMissingDataEntries {
		missingKey := &MissingDataKey{
			NsCollBlk: nsCollBlk,
		}
		key = EncodeElgDeprioMissingDataKey(missingKey)

		if missingData.None() {
			batch.Delete(key)
			continue
		}

		if val, err = EncodeMissingDataValue(missingData); err != nil {
			return errors.Wrap(err, "error while encoding missing data bitmap")
		}
		batch.Put(key, val)
	}
	return nil
}

type ExpiryData pvtdatastorage.ExpiryData

func NewExpiryData() *ExpiryData {
	return &ExpiryData{Map: make(map[string]*pvtdatastorage.Collections)}
}

func (e *ExpiryData) getOrCreateCollections(ns string) *pvtdatastorage.Collections {
	collections, ok := e.Map[ns]
	if !ok {
		collections = &pvtdatastorage.Collections{
			Map:            make(map[string]*pvtdatastorage.TxNums),
			MissingDataMap: make(map[string]bool)}
		e.Map[ns] = collections
	} else {
		// due to protobuf encoding/decoding, the previously
		// initialized map could be a nil now due to 0 length.
		// Hence, we need to reinitialize the map.
		if collections.Map == nil {
			collections.Map = make(map[string]*pvtdatastorage.TxNums)
		}
		if collections.MissingDataMap == nil {
			collections.MissingDataMap = make(map[string]bool)
		}
	}
	return collections
}

func (e *ExpiryData) AddPresentData(ns, coll string, txNum uint64) {
	collections := e.getOrCreateCollections(ns)

	txNums, ok := collections.Map[coll]
	if !ok {
		txNums = &pvtdatastorage.TxNums{}
		collections.Map[coll] = txNums
	}
	txNums.List = append(txNums.List, txNum)
}

func (e *ExpiryData) AddMissingData(ns, coll string) {
	collections := e.getOrCreateCollections(ns)
	collections.MissingDataMap[coll] = true
}

func (e *ExpiryData) Reset() {
	*e = ExpiryData{}
}
func (e *ExpiryData) String() string {
	return proto.CompactTextString(e)
}

func (*ExpiryData) ProtoMessage() {
}

func ConstructDataEntriesFromBlocksPvtData(blocksPvtData map[uint64][]*ledger.TxPvtData) map[uint64][]*DataEntry {
	// construct dataEntries for all pvtData
	dataEntries := make(map[uint64][]*DataEntry)
	for blkNum, pvtData := range blocksPvtData {
		// prepare the dataEntries for the pvtData
		dataEntries[blkNum] = prepareDataEntries(blkNum, pvtData)
	}
	return dataEntries
}

func GetLastUpdatedOldBlocksList(missingKeysIndexDB DBHandle) ([]uint64, error) {
	var v []byte
	var err error
	if v, err = missingKeysIndexDB.Get(LastUpdatedOldBlocksKey); err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}

	var updatedBlksList []uint64
	buf := proto.NewBuffer(v)
	numBlks, err := buf.DecodeVarint()
	if err != nil {
		return nil, err
	}
	for i := 0; i < int(numBlks); i++ {
		blkNum, err := buf.DecodeVarint()
		if err != nil {
			return nil, err
		}
		updatedBlksList = append(updatedBlksList, blkNum)
	}
	return updatedBlksList, nil
}

func ResetLastUpdatedOldBlocksList(missingKeysIndexDB DBHandle) error {
	batch := missingKeysIndexDB.NewUpdateBatch()
	batch.Delete(LastUpdatedOldBlocksKey)
	if err := missingKeysIndexDB.WriteBatch(batch, true); err != nil {
		return err
	}
	return nil
}

// GetMissingPvtDataInfoForMostRecentBlocks
func GetMissingPvtDataInfoForMostRecentBlocks(group []byte, maxBlock int, lastCommittedBlk uint64, btlPolicy pvtdatapolicy.BTLPolicy, missingKeysIndexDB DBHandle) (ledger.MissingPvtDataInfo, error) {
	// we assume that this function would be called by the gossip only after processing the
	// last retrieved missing pvtdata info and committing the same.
	if maxBlock < 1 {
		return nil, nil
	}

	missingPvtDataInfo := make(ledger.MissingPvtDataInfo)
	numberOfBlockProcessed := 0
	lastProcessedBlock := uint64(0)
	isMaxBlockLimitReached := false
	// as we are not acquiring a read lock, new blocks can get committed while we
	// construct the MissingPvtDataInfo. As a result, lastCommittedBlock can get
	// changed. To ensure consistency, we atomically load the lastCommittedBlock value
	lastCommittedBlock := atomic.LoadUint64(&lastCommittedBlk)

	startKey, endKey := createRangeScanKeysForElgMissingData(lastCommittedBlock, group)
	dbItr, err := missingKeysIndexDB.GetIterator(startKey, endKey)
	if err != nil {
		return nil, err
	}
	defer dbItr.Release()

	for dbItr.Next() {
		missingDataKeyBytes := dbItr.Key()
		missingDataKey := DecodeElgMissingDataKey(missingDataKeyBytes)

		if isMaxBlockLimitReached && (missingDataKey.BlkNum != lastProcessedBlock) {
			// esnures that exactly maxBlock number
			// of blocks' entries are processed
			break
		}

		// check whether the entry is expired. If so, move to the next item.
		// As we may use the old lastCommittedBlock value, there is a possibility that
		// this missing data is actually expired but we may get the stale information.
		// Though it may leads to extra work of pulling the expired data, it will not
		// affect the correctness. Further, as we try to fetch the most recent missing
		// data (less possibility of expiring now), such scenario would be rare. In the
		// best case, we can load the latest lastCommittedBlock value here atomically to
		// make this scenario very rare.
		lastCommittedBlock = atomic.LoadUint64(&lastCommittedBlk)
		expired, err := IsExpired(missingDataKey.NsCollBlk, btlPolicy, lastCommittedBlock)
		if err != nil {
			return nil, err
		}
		if expired {
			continue
		}

		// check for an existing entry for the blkNum in the MissingPvtDataInfo.
		// If no such entry exists, create one. Also, keep track of the number of
		// processed block due to maxBlock limit.
		if _, ok := missingPvtDataInfo[missingDataKey.BlkNum]; !ok {
			numberOfBlockProcessed++
			if numberOfBlockProcessed == maxBlock {
				isMaxBlockLimitReached = true
				// as there can be more than one entry for this block,
				// we cannot `break` here
				lastProcessedBlock = missingDataKey.BlkNum
			}
		}

		valueBytes := dbItr.Value()
		bitmap, err := DecodeMissingDataValue(valueBytes)
		if err != nil {
			return nil, err
		}

		// for each transaction which misses private data, make an entry in missingBlockPvtDataInfo
		for index, isSet := bitmap.NextSet(0); isSet; index, isSet = bitmap.NextSet(index + 1) {
			txNum := uint64(index)
			missingPvtDataInfo.Add(missingDataKey.BlkNum, txNum, missingDataKey.Ns, missingDataKey.Coll)
		}
	}

	return missingPvtDataInfo, nil
}

// ProcessCollsEligibilityEnabled
func ProcessCollsEligibilityEnabled(committingBlk uint64, nsCollMap map[string][]string, collElgProcSync *CollElgProcSync, missingKeysIndexDB DBHandle) error {
	key := encodeCollElgKey(committingBlk)
	m := newCollElgInfo(nsCollMap)
	val, err := encodeCollElgVal(m)
	if err != nil {
		return err
	}
	batch := missingKeysIndexDB.NewUpdateBatch()
	batch.Put(key, val)
	if err = missingKeysIndexDB.WriteBatch(batch, true); err != nil {
		return err
	}
	collElgProcSync.notify()
	return nil
}

type getExpiryDataFromEntriesOrStore func(expKey ExpiryKey) (*ExpiryData, error)

type OldBlockDataProcessor struct {
	entries                         *entriesForPvtDataOfOldBlocks
	btlPolicy                       pvtdatapolicy.BTLPolicy
	getExpiryDataFromEntriesOrStore getExpiryDataFromEntriesOrStore
	missingKeysIndexDB              DBHandle
}

func NewOldBlockDataProcessor(
	btlPolicy pvtdatapolicy.BTLPolicy,
	getExpiryDataFromEntriesOrStore getExpiryDataFromEntriesOrStore,
	preparePvtDataDoc preparePvtDataDoc,
	missingKeysIndexDB DBHandle) *OldBlockDataProcessor {
	return &OldBlockDataProcessor{
		entries:                         newEntriesForPvtDataOfOldBlocks(preparePvtDataDoc),
		btlPolicy:                       btlPolicy,
		getExpiryDataFromEntriesOrStore: getExpiryDataFromEntriesOrStore,
		missingKeysIndexDB:              missingKeysIndexDB,
	}
}

func (p *OldBlockDataProcessor) PrepareDataAndExpiryEntries(blocksPvtData map[uint64][]*ledger.TxPvtData) error {
	var dataEntries []*DataEntry
	var expData *ExpiryData

	for blkNum, pvtData := range blocksPvtData {
		dataEntries = append(dataEntries, prepareDataEntries(blkNum, pvtData)...)
	}

	for _, dataEntry := range dataEntries {
		nsCollBlk := dataEntry.Key.NsCollBlk
		txNum := dataEntry.Key.TxNum

		expKey, err := p.constructExpiryKey(dataEntry)
		if err != nil {
			return err
		}

		if neverExpires(expKey.ExpiringBlk) {
			p.entries.dataEntries[*dataEntry.Key] = dataEntry.Value
			continue
		}

		if expData, err = p.getExpiryDataFromEntriesOrStore(expKey); err != nil {
			return err
		}
		if expData == nil {
			// if expiryData is not available, it means that
			// the pruge scheduler removed these entries and the
			// associated data entry is no longer needed. Note
			// that the associated missingData entry would also
			// be not present. Hence, we can skip this data entry.
			continue
		}
		expData.AddPresentData(nsCollBlk.Ns, nsCollBlk.Coll, txNum)

		p.entries.dataEntries[*dataEntry.Key] = dataEntry.Value
		p.entries.expiryEntries[expKey] = expData
	}
	return nil
}

func (p *OldBlockDataProcessor) PrepareMissingDataEntriesToReflectReconciledData() error {
	for dataKey := range p.entries.dataEntries {
		key := dataKey.NsCollBlk
		txNum := uint(dataKey.TxNum)

		prioMissingData, err := p.getPrioMissingDataFromEntriesOrStore(key)
		if err != nil {
			return err
		}
		if prioMissingData != nil && prioMissingData.Test(txNum) {
			p.entries.prioritizedMissingDataEntries[key] = prioMissingData.Clear(txNum)
			continue
		}

		deprioMissingData, err := p.getDeprioMissingDataFromEntriesOrStore(key)
		if err != nil {
			return err
		}
		if deprioMissingData != nil && deprioMissingData.Test(txNum) {
			p.entries.deprioritizedMissingDataEntries[key] = deprioMissingData.Clear(txNum)
		}
	}

	return nil
}

func (p *OldBlockDataProcessor) PrepareMissingDataEntriesToReflectPriority(deprioritizedList ledger.MissingPvtDataInfo) error {
	for blkNum, blkMissingData := range deprioritizedList {
		for txNum, txMissingData := range blkMissingData {
			for _, nsColl := range txMissingData {
				key := NsCollBlk{
					Ns:     nsColl.Namespace,
					Coll:   nsColl.Collection,
					BlkNum: blkNum,
				}
				txNum := uint(txNum)

				prioMissingData, err := p.getPrioMissingDataFromEntriesOrStore(key)
				if err != nil {
					return err
				}
				if prioMissingData == nil {
					// we would reach here when either of the following happens:
					//   (1) when the purge scheduler already removed the respective
					//       missing data entry.
					//   (2) when the missing data info is already persistent in the
					//       deprioritized list. Currently, we do not have different
					//       levels of deprioritized list.
					// In both of the above case, we can continue to the next entry.
					continue
				}
				p.entries.prioritizedMissingDataEntries[key] = prioMissingData.Clear(txNum)

				deprioMissingData, err := p.getDeprioMissingDataFromEntriesOrStore(key)
				if err != nil {
					return err
				}
				if deprioMissingData == nil {
					deprioMissingData = &bitset.BitSet{}
				}
				p.entries.deprioritizedMissingDataEntries[key] = deprioMissingData.Set(txNum)
			}
		}
	}

	return nil
}

func (p *OldBlockDataProcessor) ConstructDBUpdateBatch() ([]*couchdb.CouchDoc, *leveldbhelper.UpdateBatch, error) {
	docs := &couchDocs{}
	batch := p.missingKeysIndexDB.NewUpdateBatch()

	if err := p.entries.addDataAndExpiryEntriesTo(docs); err != nil {
		return nil, nil, errors.WithMessage(err, "error while adding data and expiry entries to the update batch")
	}

	if err := p.entries.addElgPrioMissingDataEntriesTo(batch); err != nil {
		return nil, nil, errors.WithMessage(err, "error while adding eligible prioritized missing data entries to the update batch")
	}

	if err := p.entries.addElgDeprioMissingDataEntriesTo(batch); err != nil {
		return nil, nil, errors.WithMessage(err, "error while adding eligible deprioritized missing data entries to the update batch")
	}

	return docs.docs, batch, nil
}

func (p *OldBlockDataProcessor) constructExpiryKey(dataEntry *DataEntry) (ExpiryKey, error) {
	// get the expiryBlk number to construct the expiryKey
	nsCollBlk := dataEntry.Key.NsCollBlk
	expiringBlk, err := p.btlPolicy.GetExpiringBlock(nsCollBlk.Ns, nsCollBlk.Coll, nsCollBlk.BlkNum)
	if err != nil {
		return ExpiryKey{}, errors.WithMessagef(err, "error while constructing expiry data key")
	}

	return ExpiryKey{
		ExpiringBlk:   expiringBlk,
		CommittingBlk: nsCollBlk.BlkNum,
	}, nil
}

func (p *OldBlockDataProcessor) getPrioMissingDataFromEntriesOrStore(nsCollBlk NsCollBlk) (*bitset.BitSet, error) {
	missingData, ok := p.entries.prioritizedMissingDataEntries[nsCollBlk]
	if ok {
		return missingData, nil
	}

	missingKey := &MissingDataKey{
		NsCollBlk: nsCollBlk,
	}
	key := EncodeElgPrioMissingDataKey(missingKey)

	encMissingData, err := p.missingKeysIndexDB.Get(key)
	if err != nil {
		return nil, errors.Wrap(err, "error while getting missing data bitmap from the store")
	}
	if encMissingData == nil {
		return nil, nil
	}

	return DecodeMissingDataValue(encMissingData)
}

func (p *OldBlockDataProcessor) getDeprioMissingDataFromEntriesOrStore(nsCollBlk NsCollBlk) (*bitset.BitSet, error) {
	missingData, ok := p.entries.deprioritizedMissingDataEntries[nsCollBlk]
	if ok {
		return missingData, nil
	}

	missingKey := &MissingDataKey{
		NsCollBlk: nsCollBlk,
	}
	key := EncodeElgDeprioMissingDataKey(missingKey)

	encMissingData, err := p.missingKeysIndexDB.Get(key)
	if err != nil {
		return nil, errors.Wrap(err, "error while getting missing data bitmap from the store")
	}
	if encMissingData == nil {
		return nil, nil
	}

	return DecodeMissingDataValue(encMissingData)
}

type couchDocs struct {
	docs []*couchdb.CouchDoc
}

func (d *couchDocs) Put(doc *couchdb.CouchDoc) {
	d.docs = append(d.docs, doc)
}
