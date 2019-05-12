/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"sync/atomic"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/ledger/util/leveldbhelper"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/pvtdatapolicy"
	"github.com/hyperledger/fabric/core/ledger/pvtdatastorage"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/willf/bitset"
)

// todo add pinning script to include copied code into this file, original file from fabric is found in fabric/core/ledger/pvtdatastorage/store_imp.go
// todo below functions are originally unexported, the pinning script must capitalize these functions to export them

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
	IsEligible bool
}

type StoreEntries struct {
	DataEntries        []*DataEntry
	ExpiryEntries      []*ExpiryEntry
	MissingDataEntries map[MissingDataKey]*bitset.BitSet
}

type EntriesForPvtDataOfOldBlocks struct {
	// for each <ns, coll, blkNum, txNum>, store the dataEntry, i.e., pvtData
	DataEntries map[DataKey]*rwset.CollectionPvtReadWriteSet
	// store the retrieved (& updated) expiryData in expiryEntries
	ExpiryEntries map[ExpiryKey]*ExpiryData
	// for each <ns, coll, blkNum>, store the retrieved (& updated) bitmap in the missingDataEntries
	MissingDataEntries map[NsCollBlk]*bitset.BitSet
}

func (updateEntries *EntriesForPvtDataOfOldBlocks) AddDataEntry(dataEntry *DataEntry) {
	dataKey := DataKey{NsCollBlk: dataEntry.Key.NsCollBlk, TxNum: dataEntry.Key.TxNum}
	updateEntries.DataEntries[dataKey] = dataEntry.Value
}

func (updateEntries *EntriesForPvtDataOfOldBlocks) UpdateAndAddExpiryEntry(expiryEntry *ExpiryEntry, dataKey *DataKey) {
	txNum := dataKey.TxNum
	nsCollBlk := dataKey.NsCollBlk
	// update
	expiryEntry.Value.AddPresentData(nsCollBlk.Ns, nsCollBlk.Coll, txNum)
	// we cannot delete entries from MissingDataMap as
	// we keep only one entry per missing <ns-col>
	// irrespective of the number of txNum.

	// add
	expiryKey := ExpiryKey{expiryEntry.Key.ExpiringBlk, expiryEntry.Key.CommittingBlk}
	updateEntries.ExpiryEntries[expiryKey] = expiryEntry.Value
}

func (updateEntries *EntriesForPvtDataOfOldBlocks) UpdateAndAddMissingDataEntry(missingData *bitset.BitSet, dataKey *DataKey) {

	txNum := dataKey.TxNum
	nsCollBlk := dataKey.NsCollBlk
	// update
	missingData.Clear(uint(txNum))
	// add
	updateEntries.MissingDataEntries[nsCollBlk] = missingData
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

func ConstructUpdateEntriesFromDataEntries(dataEntries []*DataEntry, btlPolicy pvtdatapolicy.BTLPolicy,
	getExpiryDataOfExpiryKey func(*ExpiryKey) (*ExpiryData, error), getBitmapOfMissingDataKey func(*MissingDataKey) (*bitset.BitSet, error)) (*EntriesForPvtDataOfOldBlocks, error) {
	updateEntries := &EntriesForPvtDataOfOldBlocks{
		DataEntries:        make(map[DataKey]*rwset.CollectionPvtReadWriteSet),
		ExpiryEntries:      make(map[ExpiryKey]*ExpiryData),
		MissingDataEntries: make(map[NsCollBlk]*bitset.BitSet)}

	// for each data entry, first, get the expiryData and missingData from the pvtStore.
	// Second, update the expiryData and missingData as per the data entry. Finally, add
	// the data entry along with the updated expiryData and missingData to the update entries
	for _, dataEntry := range dataEntries {
		// get the expiryBlk number to construct the expiryKey
		expiryKey, err := constructExpiryKeyFromDataEntry(dataEntry, btlPolicy)
		if err != nil {
			return nil, err
		}

		// get the existing expiryData ntry
		var expiryData *ExpiryData
		if !neverExpires(expiryKey.ExpiringBlk) {
			if expiryData, err = getExpiryDataFromUpdateEntriesOrStore(updateEntries, expiryKey, getExpiryDataOfExpiryKey); err != nil {
				return nil, err
			}
			if expiryData == nil {
				// data entry is already expired
				// and purged (a rare scenario)
				continue
			}
		}

		// get the existing missingData entry
		var missingData *bitset.BitSet
		nsCollBlk := dataEntry.Key.NsCollBlk
		if missingData, err = getMissingDataFromUpdateEntriesOrStore(updateEntries, nsCollBlk, getBitmapOfMissingDataKey); err != nil {
			return nil, err
		}
		if missingData == nil {
			// data entry is already expired
			// and purged (a rare scenario)
			continue
		}

		updateEntries.AddDataEntry(dataEntry)
		if expiryData != nil { // would be nill for the never expiring entry
			expiryEntry := &ExpiryEntry{Key: &expiryKey, Value: expiryData}
			updateEntries.UpdateAndAddExpiryEntry(expiryEntry, dataEntry.Key)
		}
		updateEntries.UpdateAndAddMissingDataEntry(missingData, dataEntry.Key)
	}
	return updateEntries, nil
}

func ConstructUpdateBatchFromUpdateEntries(updateEntries *EntriesForPvtDataOfOldBlocks, batch *leveldbhelper.UpdateBatch) (*leveldbhelper.UpdateBatch, error) {
	// add the following four types of entries to the update batch: (1) updated missing data entries

	// (1) add updated missingData to the batch
	if err := addUpdatedMissingDataEntriesToUpdateBatch(batch, updateEntries); err != nil {
		return nil, err
	}

	return batch, nil
}

func constructExpiryKeyFromDataEntry(dataEntry *DataEntry, btlPolicy pvtdatapolicy.BTLPolicy) (ExpiryKey, error) {
	// get the expiryBlk number to construct the expiryKey
	nsCollBlk := dataEntry.Key.NsCollBlk
	expiringBlk, err := btlPolicy.GetExpiringBlock(nsCollBlk.Ns, nsCollBlk.Coll, nsCollBlk.BlkNum)
	if err != nil {
		return ExpiryKey{}, err
	}
	return ExpiryKey{ExpiringBlk: expiringBlk, CommittingBlk: nsCollBlk.BlkNum}, nil
}

func getExpiryDataFromUpdateEntriesOrStore(updateEntries *EntriesForPvtDataOfOldBlocks, expiryKey ExpiryKey, getExpiryDataOfExpiryKey func(*ExpiryKey) (*ExpiryData, error)) (*ExpiryData, error) {
	expiryData, ok := updateEntries.ExpiryEntries[expiryKey]
	if !ok {
		var err error
		expiryData, err = getExpiryDataOfExpiryKey(&expiryKey)
		if err != nil {
			return nil, err
		}
	}
	return expiryData, nil
}

func getMissingDataFromUpdateEntriesOrStore(updateEntries *EntriesForPvtDataOfOldBlocks, nsCollBlk NsCollBlk, getBitmapOfMissingDataKey func(*MissingDataKey) (*bitset.BitSet, error)) (*bitset.BitSet, error) {
	missingData, ok := updateEntries.MissingDataEntries[nsCollBlk]
	if !ok {
		var err error
		missingDataKey := &MissingDataKey{NsCollBlk: nsCollBlk, IsEligible: true}
		missingData, err = getBitmapOfMissingDataKey(missingDataKey)
		if err != nil {
			return nil, err
		}
	}
	return missingData, nil
}

func addUpdatedMissingDataEntriesToUpdateBatch(batch *leveldbhelper.UpdateBatch, entries *EntriesForPvtDataOfOldBlocks) error {
	var keyBytes, valBytes []byte
	var err error
	for nsCollBlk, missingData := range entries.MissingDataEntries {
		keyBytes = EncodeMissingDataKey(&MissingDataKey{nsCollBlk, true})
		// if the missingData is empty, we need to delete the missingDataKey
		if missingData.None() {
			batch.Delete(keyBytes)
			continue
		}
		if valBytes, err = EncodeMissingDataValue(missingData); err != nil {
			return err
		}
		batch.Put(keyBytes, valBytes)
	}
	return nil
}

func GetLastUpdatedOldBlocksList(missingKeysIndexDB *leveldbhelper.DBHandle) ([]uint64, error) {
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

func ResetLastUpdatedOldBlocksList(missingKeysIndexDB *leveldbhelper.DBHandle) error {
	batch := leveldbhelper.NewUpdateBatch()
	batch.Delete(LastUpdatedOldBlocksKey)
	if err := missingKeysIndexDB.WriteBatch(batch, true); err != nil {
		return err
	}
	return nil
}

// GetMissingPvtDataInfoForMostRecentBlocks
func GetMissingPvtDataInfoForMostRecentBlocks(maxBlock int, lastCommittedBlk uint64, btlPolicy pvtdatapolicy.BTLPolicy, missingKeysIndexDB *leveldbhelper.DBHandle) (ledger.MissingPvtDataInfo, error) {
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

	startKey, endKey := createRangeScanKeysForEligibleMissingDataEntries(lastCommittedBlock)
	dbItr := missingKeysIndexDB.GetIterator(startKey, endKey)
	defer dbItr.Release()

	for dbItr.Next() {
		missingDataKeyBytes := dbItr.Key()
		missingDataKey := decodeMissingDataKey(missingDataKeyBytes)

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
func ProcessCollsEligibilityEnabled(committingBlk uint64, nsCollMap map[string][]string, collElgProcSync *CollElgProc, missingKeysIndexDB *leveldbhelper.DBHandle) error {
	key := encodeCollElgKey(committingBlk)
	m := newCollElgInfo(nsCollMap)
	val, err := encodeCollElgVal(m)
	if err != nil {
		return err
	}
	batch := leveldbhelper.NewUpdateBatch()
	batch.Put(key, val)
	if err = missingKeysIndexDB.WriteBatch(batch, true); err != nil {
		return err
	}
	collElgProcSync.notify()
	return nil
}
