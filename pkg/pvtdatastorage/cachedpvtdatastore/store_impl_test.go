/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cachedpvtdatastore

import (
	"fmt"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/ledger/rwset"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	btltestutil "github.com/hyperledger/fabric/core/ledger/pvtdatapolicy/testutil"
	"github.com/hyperledger/fabric/core/ledger/pvtdatastorage"
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
	req.NoError(store.Commit(0, nil, nil))

	// pvt data with block 1 - commit
	req.NoError(store.Commit(1, testData, blk1MissingData))

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
	req.NoError(store.Commit(2, testData, blk2MissingData))
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

	req.NoError(store.Commit(0, testData, nil))

	_, ok := store.Commit(2, testData, nil).(*pvtdatastorage.ErrIllegalCall)
	req.True(ok)
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
