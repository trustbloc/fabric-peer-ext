/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"math"

	"github.com/hyperledger/fabric/core/ledger/pvtdatastorage"

	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/pvtdatapolicy"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/willf/bitset"
)

// todo add pinning script to include copied code into this file, original file from fabric is found in fabric/core/ledger/pvtdatastorage/helper.go
// todo below functions are originally unexported, the pinning script must capitalize these functions to export them

func PrepareStoreEntries(blockNum uint64, pvtData []*ledger.TxPvtData, btlPolicy pvtdatapolicy.BTLPolicy,
	missingPvtData ledger.TxMissingPvtDataMap) (*StoreEntries, error) {
	dataEntries := prepareDataEntries(blockNum, pvtData)

	missingDataEntries := prepareMissingDataEntries(blockNum, missingPvtData)

	expiryEntries, err := prepareExpiryEntries(blockNum, dataEntries, missingDataEntries, btlPolicy)
	if err != nil {
		return nil, err
	}

	return &StoreEntries{
		DataEntries:        dataEntries,
		ExpiryEntries:      expiryEntries,
		MissingDataEntries: missingDataEntries}, nil
}

func prepareDataEntries(blockNum uint64, pvtData []*ledger.TxPvtData) []*DataEntry {
	var dataEntries []*DataEntry
	for _, txPvtdata := range pvtData {
		for _, nsPvtdata := range txPvtdata.WriteSet.NsPvtRwset {
			for _, collPvtdata := range nsPvtdata.CollectionPvtRwset {
				txnum := txPvtdata.SeqInBlock
				ns := nsPvtdata.Namespace
				coll := collPvtdata.CollectionName
				dataKey := &DataKey{NsCollBlk: NsCollBlk{Ns: ns, Coll: coll, BlkNum: blockNum}, TxNum: txnum}
				dataEntries = append(dataEntries, &DataEntry{Key: dataKey, Value: collPvtdata})
			}
		}
	}
	return dataEntries
}

func prepareMissingDataEntries(committingBlk uint64, missingPvtData ledger.TxMissingPvtDataMap) map[MissingDataKey]*bitset.BitSet {
	missingDataEntries := make(map[MissingDataKey]*bitset.BitSet)

	for txNum, missingData := range missingPvtData {
		for _, nsColl := range missingData {
			key := MissingDataKey{NsCollBlk: NsCollBlk{Ns: nsColl.Namespace, Coll: nsColl.Collection, BlkNum: committingBlk},
				IsEligible: nsColl.IsEligible}

			if _, ok := missingDataEntries[key]; !ok {
				missingDataEntries[key] = &bitset.BitSet{}
			}
			bitmap := missingDataEntries[key]

			bitmap.Set(uint(txNum))
		}
	}

	return missingDataEntries
}

// prepareExpiryEntries returns expiry entries for both private data which is present in the committingBlk
// and missing private.
func prepareExpiryEntries(committingBlk uint64, dataEntries []*DataEntry, missingDataEntries map[MissingDataKey]*bitset.BitSet,
	btlPolicy pvtdatapolicy.BTLPolicy) ([]*ExpiryEntry, error) {

	var expiryEntries []*ExpiryEntry
	mapByExpiringBlk := make(map[uint64]*ExpiryData)

	// 1. prepare expiryData for non-missing data
	for _, dataEntry := range dataEntries {
		prepareExpiryEntriesForPresentData(mapByExpiringBlk, dataEntry.Key, btlPolicy)

	}

	// 2. prepare expiryData for missing data
	for missingDataKey := range missingDataEntries {
		prepareExpiryEntriesForMissingData(mapByExpiringBlk, &missingDataKey, btlPolicy)

	}

	for expiryBlk, expiryData := range mapByExpiringBlk {
		expiryKey := &ExpiryKey{ExpiringBlk: expiryBlk, CommittingBlk: committingBlk}
		expiryEntries = append(expiryEntries, &ExpiryEntry{Key: expiryKey, Value: expiryData})
	}

	return expiryEntries, nil
}

// prepareExpiryDataForPresentData creates expiryData for non-missing pvt data
func prepareExpiryEntriesForPresentData(mapByExpiringBlk map[uint64]*ExpiryData, dataKey *DataKey, btlPolicy pvtdatapolicy.BTLPolicy) error {
	expiringBlk, err := btlPolicy.GetExpiringBlock(dataKey.Ns, dataKey.Coll, dataKey.BlkNum)
	if err != nil {
		return err
	}
	if neverExpires(expiringBlk) {
		return nil
	}

	expiryData := getOrCreateExpiryData(mapByExpiringBlk, expiringBlk)

	expiryData.AddPresentData(dataKey.Ns, dataKey.Coll, dataKey.TxNum)
	return nil
}

// prepareExpiryDataForMissingData creates expiryData for missing pvt data
func prepareExpiryEntriesForMissingData(mapByExpiringBlk map[uint64]*ExpiryData, missingKey *MissingDataKey, btlPolicy pvtdatapolicy.BTLPolicy) error {
	expiringBlk, err := btlPolicy.GetExpiringBlock(missingKey.Ns, missingKey.Coll, missingKey.BlkNum)
	if err != nil {
		return err
	}
	if neverExpires(expiringBlk) {
		return nil
	}

	expiryData := getOrCreateExpiryData(mapByExpiringBlk, expiringBlk)

	expiryData.AddMissingData(missingKey.Ns, missingKey.Coll)
	return nil
}

func getOrCreateExpiryData(mapByExpiringBlk map[uint64]*ExpiryData, expiringBlk uint64) *ExpiryData {
	expiryData, ok := mapByExpiringBlk[expiringBlk]
	if !ok {
		expiryData = NewExpiryData()
		mapByExpiringBlk[expiringBlk] = expiryData
	}
	return expiryData
}

func PassesFilter(dataKey *DataKey, filter ledger.PvtNsCollFilter) bool {
	return filter == nil || filter.Has(dataKey.Ns, dataKey.Coll)
}

func IsExpired(key NsCollBlk, btl pvtdatapolicy.BTLPolicy, latestBlkNum uint64) (bool, error) {
	expiringBlk, err := btl.GetExpiringBlock(key.Ns, key.Coll, key.BlkNum)
	if err != nil {
		return false, err
	}

	return latestBlkNum >= expiringBlk, nil
}

// DeriveKeys constructs dataKeys and missingDataKey from an expiryEntry
func DeriveKeys(expiryEntry *ExpiryEntry) (dataKeys []*DataKey, missingDataKeys []*MissingDataKey) {
	for ns, colls := range expiryEntry.Value.Map {
		// 1. constructs dataKeys of expired existing pvt data
		for coll, txNums := range colls.Map {
			for _, txNum := range txNums.List {
				dataKeys = append(dataKeys,
					&DataKey{NsCollBlk{ns, coll, expiryEntry.Key.CommittingBlk}, txNum})
			}
		}
		// 2. constructs missingDataKeys of expired missing pvt data
		for coll := range colls.MissingDataMap {
			// one key for eligible entries and another for ieligible entries
			missingDataKeys = append(missingDataKeys,
				&MissingDataKey{NsCollBlk{ns, coll, expiryEntry.Key.CommittingBlk}, true})
			missingDataKeys = append(missingDataKeys,
				&MissingDataKey{NsCollBlk{ns, coll, expiryEntry.Key.CommittingBlk}, false})

		}
	}
	return
}

func newCollElgInfo(nsCollMap map[string][]string) *pvtdatastorage.CollElgInfo {
	m := &pvtdatastorage.CollElgInfo{NsCollMap: map[string]*pvtdatastorage.CollNames{}}
	for ns, colls := range nsCollMap {
		collNames, ok := m.NsCollMap[ns]
		if !ok {
			collNames = &pvtdatastorage.CollNames{}
			m.NsCollMap[ns] = collNames
		}
		collNames.Entries = colls
	}
	return m
}

type TxPvtdataAssembler struct {
	blockNum, txNum uint64
	txWset          *rwset.TxPvtReadWriteSet
	currentNsWSet   *rwset.NsPvtReadWriteSet
	firstCall       bool
}

func NewTxPvtdataAssembler(blockNum, txNum uint64) *TxPvtdataAssembler {
	return &TxPvtdataAssembler{blockNum, txNum, &rwset.TxPvtReadWriteSet{}, nil, true}
}

func (a *TxPvtdataAssembler) Add(ns string, collPvtWset *rwset.CollectionPvtReadWriteSet) {
	// start a NsWset
	if a.firstCall {
		a.currentNsWSet = &rwset.NsPvtReadWriteSet{Namespace: ns}
		a.firstCall = false
	}

	// if a new ns started, add the existing NsWset to TxWset and start a new one
	if a.currentNsWSet.Namespace != ns {
		a.txWset.NsPvtRwset = append(a.txWset.NsPvtRwset, a.currentNsWSet)
		a.currentNsWSet = &rwset.NsPvtReadWriteSet{Namespace: ns}
	}
	// add the collWset to the current NsWset
	a.currentNsWSet.CollectionPvtRwset = append(a.currentNsWSet.CollectionPvtRwset, collPvtWset)
}

func (a *TxPvtdataAssembler) done() {
	if a.currentNsWSet != nil {
		a.txWset.NsPvtRwset = append(a.txWset.NsPvtRwset, a.currentNsWSet)
	}
	a.currentNsWSet = nil
}

func (a *TxPvtdataAssembler) GetTxPvtdata() *ledger.TxPvtData {
	a.done()
	return &ledger.TxPvtData{SeqInBlock: a.txNum, WriteSet: a.txWset}
}

func neverExpires(expiringBlkNum uint64) bool {
	return expiringBlkNum == math.MaxUint64
}
