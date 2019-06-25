/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cachedpvtdatastore

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/bluele/gcache"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	btltestutil "github.com/hyperledger/fabric/core/ledger/pvtdatapolicy/testutil"
	"github.com/hyperledger/fabric/core/ledger/pvtdatastorage"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/pvtdatastorage/common"
)

func TestCommitPvtDataOfOldBlocks(t *testing.T) {
	env := NewTestStoreEnv(t, "ledger", nil)
	store := env.TestStore
	err := store.CommitPvtDataOfOldBlocks(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not supported")
}

func TestGetMissingPvtDataInfoForMostRecentBlocks(t *testing.T) {
	env := NewTestStoreEnv(t, "ledger", nil)
	store := env.TestStore
	_, err := store.GetMissingPvtDataInfoForMostRecentBlocks(0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not supported")
}

func TestGetLastUpdatedOldBlocksPvtData(t *testing.T) {
	env := NewTestStoreEnv(t, "ledger", nil)
	store := env.TestStore
	_, err := store.GetLastUpdatedOldBlocksPvtData()
	require.Error(t, err)
	require.Contains(t, err.Error(), "not supported")
}

func TestResetLastUpdatedOldBlocksList(t *testing.T) {
	env := NewTestStoreEnv(t, "ledger", nil)
	store := env.TestStore
	err := store.ResetLastUpdatedOldBlocksList()
	require.Error(t, err)
	require.Contains(t, err.Error(), "not supported")
}

func TestProcessCollsEligibilityEnabled(t *testing.T) {
	env := NewTestStoreEnv(t, "ledger", nil)
	store := env.TestStore
	err := store.ProcessCollsEligibilityEnabled(0, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not supported")
}

func TestShutdown(t *testing.T) {
	env := NewTestStoreEnv(t, "ledger", nil)
	store := env.TestStore
	store.Shutdown()
}

func TestGetPvtDataByBlockNum(t *testing.T) {
	env := NewTestStoreEnv(t, "ledger", nil)
	cacheStore := env.TestStore
	s := cacheStore.(*store)
	s.isEmpty = true
	_, err := s.GetPvtDataByBlockNum(0, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "The store is empty")
}

func TestV11RetrievePvtdata(t *testing.T) {
	t.Run("test empty data entry", func(t *testing.T) {
		r, err := v11RetrievePvtdata([]*common.DataEntry{}, nil)
		require.NoError(t, err)
		require.Empty(t, r)
	})

	t.Run("test error from EncodeDataValue", func(t *testing.T) {
		_, err := v11RetrievePvtdata([]*common.DataEntry{{Value: nil}}, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "EncodeDataValue failed")
	})

	t.Run("test success", func(t *testing.T) {
		r, err := v11RetrievePvtdata([]*common.DataEntry{{Key: &common.DataKey{}, Value: &rwset.CollectionPvtReadWriteSet{}}}, nil)
		require.NoError(t, err)
		require.Equal(t, len(r), 1)
	})
}

func TestEmptyStore(t *testing.T) {
	env := NewTestStoreEnv(t, "testemptystore", nil)
	req := require.New(t)
	store := env.TestStore
	testEmpty(true, req, store)
	testPendingBatch(false, req, store)
}

func TestStoreBasicCommitAndRetrieval(t *testing.T) {
	btlPolicy := btltestutil.SampleBTLPolicy(
		map[[2]string]uint64{
			{"ns-1", "coll-1"}: 0,
			{"ns-1", "coll-2"}: 0,
			{"ns-2", "coll-1"}: 0,
			{"ns-2", "coll-2"}: 0,
			{"ns-3", "coll-1"}: 0,
			{"ns-4", "coll-1"}: 0,
			{"ns-4", "coll-2"}: 0,
		},
	)

	env := NewTestStoreEnv(t, "teststorebasiccommitandretrieval", btlPolicy)
	req := require.New(t)
	store := env.TestStore
	testData := []*ledger.TxPvtData{
		produceSamplePvtdata(t, 2, []string{"ns-1:coll-1", "ns-1:coll-2", "ns-2:coll-1", "ns-2:coll-2"}),
		produceSamplePvtdata(t, 4, []string{"ns-1:coll-1", "ns-1:coll-2", "ns-2:coll-1", "ns-2:coll-2"}),
	}

	// construct missing data for block 1
	blk1MissingData := make(ledger.TxMissingPvtDataMap)

	// eligible missing data in tx1
	blk1MissingData.Add(1, "ns-1", "coll-1", true)
	blk1MissingData.Add(1, "ns-1", "coll-2", true)
	blk1MissingData.Add(1, "ns-2", "coll-1", true)
	blk1MissingData.Add(1, "ns-2", "coll-2", true)
	// eligible missing data in tx2
	blk1MissingData.Add(2, "ns-3", "coll-1", true)
	// ineligible missing data in tx4
	blk1MissingData.Add(4, "ns-4", "coll-1", false)
	blk1MissingData.Add(4, "ns-4", "coll-2", false)

	// construct missing data for block 2
	blk2MissingData := make(ledger.TxMissingPvtDataMap)
	// eligible missing data in tx1
	blk2MissingData.Add(1, "ns-1", "coll-1", true)
	blk2MissingData.Add(1, "ns-1", "coll-2", true)
	// eligible missing data in tx3
	blk2MissingData.Add(3, "ns-1", "coll-1", true)

	// no pvt data with block 0
	req.NoError(store.Prepare(0, nil, nil))
	req.NoError(store.Commit())

	// pvt data with block 1 - commit
	req.NoError(store.Prepare(1, testData, blk1MissingData))
	req.NoError(store.Commit())

	// pvt data with block 2 - rollback
	req.NoError(store.Prepare(2, testData, nil))
	req.NoError(store.Rollback())

	// pvt data retrieval for block 0 should return nil
	var nilFilter ledger.PvtNsCollFilter
	retrievedData, err := store.GetPvtDataByBlockNum(0, nilFilter)
	req.NoError(err)
	req.Nil(retrievedData)

	// pvt data retrieval for block 1 should return full pvtdata
	retrievedData, err = store.GetPvtDataByBlockNum(1, nilFilter)
	req.NoError(err)
	for i, data := range retrievedData {
		req.Equal(data.SeqInBlock, testData[i].SeqInBlock)
		req.True(proto.Equal(data.WriteSet, testData[i].WriteSet))
	}

	// pvt data retrieval for block 1 with filter should return filtered pvtdata
	filter := ledger.NewPvtNsCollFilter()
	filter.Add("ns-1", "coll-1")
	filter.Add("ns-2", "coll-2")
	retrievedData, err = store.GetPvtDataByBlockNum(1, filter)
	expectedRetrievedData := []*ledger.TxPvtData{
		produceSamplePvtdata(t, 2, []string{"ns-1:coll-1", "ns-2:coll-2"}),
		produceSamplePvtdata(t, 4, []string{"ns-1:coll-1", "ns-2:coll-2"}),
	}
	for i, data := range retrievedData {
		req.Equal(data.SeqInBlock, expectedRetrievedData[i].SeqInBlock)
		req.True(proto.Equal(data.WriteSet, expectedRetrievedData[i].WriteSet))
	}

	// pvt data retrieval for block 2 should return ErrOutOfRange
	retrievedData, err = store.GetPvtDataByBlockNum(2, nilFilter)
	_, ok := err.(*pvtdatastorage.ErrOutOfRange)
	req.True(ok)
	req.Nil(retrievedData)

	// pvt data with block 2 - commit
	req.NoError(store.Prepare(2, testData, blk2MissingData))
	req.NoError(store.Commit())
}

func TestStoreState(t *testing.T) {
	btlPolicy := btltestutil.SampleBTLPolicy(
		map[[2]string]uint64{
			{"ns-1", "coll-1"}: 0,
			{"ns-1", "coll-2"}: 0,
		},
	)
	env := NewTestStoreEnv(t, "teststorestate", btlPolicy)
	req := require.New(t)
	store := env.TestStore
	testData := []*ledger.TxPvtData{
		produceSamplePvtdata(t, 0, []string{"ns-1:coll-1", "ns-1:coll-2"}),
	}
	_, ok := store.Prepare(1, testData, nil).(*pvtdatastorage.ErrIllegalCall)
	req.True(ok)

	req.Nil(store.Prepare(0, testData, nil))
	req.NoError(store.Commit())

	req.Nil(store.Prepare(1, testData, nil))
	_, ok = store.Prepare(2, testData, nil).(*pvtdatastorage.ErrIllegalCall)
	req.True(ok)
}

func TestInitLastCommittedBlock(t *testing.T) {
	env := NewTestStoreEnv(t, "teststorestate", nil)
	req := require.New(t)
	store := env.TestStore

	testLastCommittedBlockHeight(0, req, store)
	existingLastBlockNum := uint64(25)
	req.NoError(store.InitLastCommittedBlock(existingLastBlockNum))

	testEmpty(false, req, store)
	testPendingBatch(false, req, store)
	testLastCommittedBlockHeight(existingLastBlockNum+1, req, store)

	env.CloseAndReopen()
	testEmpty(false, req, store)
	testPendingBatch(false, req, store)
	testLastCommittedBlockHeight(existingLastBlockNum+1, req, store)

	err := store.InitLastCommittedBlock(30)
	_, ok := err.(*pvtdatastorage.ErrIllegalCall)
	req.True(ok)
}

func TestRollBack(t *testing.T) {
	btlPolicy := btltestutil.SampleBTLPolicy(
		map[[2]string]uint64{
			{"ns-1", "coll-1"}: 0,
			{"ns-1", "coll-2"}: 0,
		},
	)
	env := NewTestStoreEnv(t, "testrollback", btlPolicy)
	req := require.New(t)
	store := env.TestStore
	req.NoError(store.Prepare(0, nil, nil))
	req.NoError(store.Commit())

	pvtdata := []*ledger.TxPvtData{
		produceSamplePvtdata(t, 0, []string{"ns-1:coll-1", "ns-1:coll-2"}),
		produceSamplePvtdata(t, 5, []string{"ns-1:coll-1", "ns-1:coll-2"}),
	}
	missingData := make(ledger.TxMissingPvtDataMap)
	missingData.Add(1, "ns-1", "coll-1", true)
	missingData.Add(5, "ns-1", "coll-1", true)
	missingData.Add(5, "ns-2", "coll-2", false)

	for i := 1; i <= 9; i++ {
		req.NoError(store.Prepare(uint64(i), pvtdata, missingData))
		req.NoError(store.Commit())
	}

	datakeyTx0 := &common.DataKey{
		NsCollBlk: common.NsCollBlk{Ns: "ns-1", Coll: "coll-1"},
		TxNum:     0,
	}
	datakeyTx5 := &common.DataKey{
		NsCollBlk: common.NsCollBlk{Ns: "ns-1", Coll: "coll-1"},
		TxNum:     5,
	}
	eligibleMissingdatakey := &common.MissingDataKey{
		NsCollBlk:  common.NsCollBlk{Ns: "ns-1", Coll: "coll-1"},
		IsEligible: true,
	}

	// test store state before preparing for block 10
	testPendingBatch(false, req, store)
	testLastCommittedBlockHeight(10, req, store)

	// prepare for block 10 and test store for presence of datakeys and eligibile missingdatakeys
	req.NoError(store.Prepare(10, pvtdata, missingData))
	testPendingBatch(true, req, store)
	testLastCommittedBlockHeight(10, req, store)

	datakeyTx0.BlkNum = 10
	datakeyTx5.BlkNum = 10
	req.True(testPendingDataKeyExists(t, store, datakeyTx0))
	req.True(testPendingDataKeyExists(t, store, datakeyTx5))

	// rollback last prepared block and test store for absence of datakeys and eligibile missingdatakeys
	err := store.Rollback()
	req.NoError(err)
	testPendingBatch(false, req, store)
	testLastCommittedBlockHeight(10, req, store)
	req.False(testPendingDataKeyExists(t, store, datakeyTx0))
	req.False(testPendingDataKeyExists(t, store, datakeyTx5))

	// For previously committed blocks the datakeys and eligibile missingdatakeys should still be present
	for i := 1; i <= 9; i++ {
		datakeyTx0.BlkNum = uint64(i)
		datakeyTx5.BlkNum = uint64(i)
		eligibleMissingdatakey.BlkNum = uint64(i)
		req.True(testDataKeyExists(t, store, datakeyTx0))
		req.True(testDataKeyExists(t, store, datakeyTx5))
	}
}

func testLastCommittedBlockHeight(expectedBlockHt uint64, req *require.Assertions, store pvtdatastorage.Store) {
	blkHt, err := store.LastCommittedBlockHeight()
	req.NoError(err)
	req.Equal(expectedBlockHt, blkHt)
}

func testDataKeyExists(t *testing.T, s pvtdatastorage.Store, dataKey *common.DataKey) bool {

	value, err := s.(*store).cache.Get(dataKey.BlkNum)
	if err != nil {
		if err != gcache.KeyNotFoundError {
			panic(fmt.Sprintf("Get must never return an error other than KeyNotFoundError err:%s", err))
		}
		return false
	}

	dataEntries := value.([]*common.DataEntry)
	for _, value := range dataEntries {
		if bytes.Equal(common.EncodeDataKey(value.Key), common.EncodeDataKey(dataKey)) {
			return true
		}
	}
	return false
}

func testPendingDataKeyExists(t *testing.T, s pvtdatastorage.Store, dataKey *common.DataKey) bool {
	if s.(*store).pendingPvtData.dataEntries == nil {
		return false
	}
	for _, value := range s.(*store).pendingPvtData.dataEntries {
		if bytes.Equal(common.EncodeDataKey(value.Key), common.EncodeDataKey(dataKey)) {
			return true
		}
	}
	return false
}

func testEmpty(expectedEmpty bool, req *require.Assertions, store pvtdatastorage.Store) {
	isEmpty, err := store.IsEmpty()
	req.NoError(err)
	req.Equal(expectedEmpty, isEmpty)
}

func testPendingBatch(expectedPending bool, req *require.Assertions, store pvtdatastorage.Store) {
	hasPendingBatch, err := store.HasPendingBatch()
	req.NoError(err)
	req.Equal(expectedPending, hasPendingBatch)
}

func produceSamplePvtdata(t *testing.T, txNum uint64, nsColls []string) *ledger.TxPvtData {
	builder := rwsetutil.NewRWSetBuilder()
	for _, nsColl := range nsColls {
		nsCollSplit := strings.Split(nsColl, ":")
		ns := nsCollSplit[0]
		coll := nsCollSplit[1]
		builder.AddToPvtAndHashedWriteSet(ns, coll, fmt.Sprintf("key-%s-%s", ns, coll), []byte(fmt.Sprintf("value-%s-%s", ns, coll)))
	}
	simRes, err := builder.GetTxSimulationResults()
	require.NoError(t, err)
	return &ledger.TxPvtData{SeqInBlock: txNum, WriteSet: simRes.PvtSimulationResults}
}
