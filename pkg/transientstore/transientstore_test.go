/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transientstore

import (
	"fmt"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/ledger"
	origts "github.com/hyperledger/fabric/core/transientstore"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

const (
	txID1 = "txid1"
	txID2 = "txid2"
	txID3 = "txid3"
	txID4 = "txid4"

	ns1 = "ns1"

	coll0 = "coll0"
	coll1 = "coll1"
	coll2 = "coll2"
	coll3 = "coll3"

	key1 = "key1"
	key2 = "key2"
	key3 = "key3"
)

func TestTransientStore(t *testing.T) {
	s := NewStoreProvider()
	require.NotNil(t, s, "Creating new store must not return nil")

	transientStore, err := s.OpenStore("ledgerid")
	require.NoError(t, err, "Opening transient store should not throw an error")
	require.NotNil(t, transientStore, "TransientStore should not be nil")
	ts := transientStore.(*store)
	require.NotNil(t, ts.cache, "Transient store's cache should not be nil")
	require.NotNil(t, ts.blockHeightCache, "Transient store's cache should not be nil")
	require.NotNil(t, ts.txidCache, "Transient store's cache should not be nil")

	b := mocks.NewPvtReadWriteSetBuilder()
	b.Namespace(ns1).
		Collection(coll1).
		TransientConfig("OR('Org1MSP.member')", 1, 2, "1m").
		Write(key1, []byte("value")).
		WithMarshalError()

	t.Run("Test Persist nil TxPvtReadWriteSet", func(t *testing.T) {
		err = ts.Persist("12345", 1, nil)
		require.Error(t, err, "Persist() call of transient store should throw an error with nil privateSimulationResults")
	})

	err = ts.Persist(txID1, 500, b.Build())
	require.NoError(t, err, "Persist() call of transient store should not throw an error")
	require.True(t, ts.cache.Has(txID1), "Transient store cache should have txId: %s following Persist(\"%s\", %d, &rwset.TxPvtReadWriteSet{}) cache content: %+v", txID1, txID1, 500, ts.cache.GetALL())
	require.True(t, ts.blockHeightCache.Len() == 1, "Transient store blockHeightCache should have 1 item following Persist() call")
	require.True(t, ts.blockHeightCache.Has(uint64(500)), "Transient store blockHeightCache should have blockHeight: %d following Persist(\"%s\", %d, &rwset.TxPvtReadWriteSet{}) cache content: %+v", 500, txID1, 500, ts.blockHeightCache.GetALL())

	t.Run("Test GetTxPvtRWSetByTxid with TxPvtReadWriteSet", func(t *testing.T) {
		var nilFilter ledger.PvtNsCollFilter
		scanner, err := ts.GetTxPvtRWSetByTxid(txID1, nilFilter)
		require.NoError(t, err, "GetTxPvtRWSetByTxid() call should not return error")
		require.NotNil(t, scanner, "GetTxPvtRWSetByTxid() call should not return a nil scanner")
		res, err := scanner.Next()
		require.NoError(t, err, "scanner.Next() call should not return error")
		require.NotNil(t, res, "scanner.Next() call should not return a nil EndorserPvtSimulationResults")
		resWithConfig, err := scanner.Next()
		require.NoError(t, err, "scanner.Next() call should return error for txid \"%s\"", txID1)
		require.Nil(t, resWithConfig, "scanner.Next() call should return a nil EndorserPvtSimulationResults for txid \"%s\" as it does not contain config in the results", txID1)
		scanner.Close()
	})

	value1One := []byte("value1One")
	value1Two := []byte("value1Two")

	value2One := []byte("value2One")
	value3One := []byte("value3One")

	b = mocks.NewPvtReadWriteSetBuilder()
	ns1Builder := b.Namespace(ns1)
	coll0Builder := ns1Builder.Collection(coll0)
	coll0Builder.
		TransientConfig("OR('Org1MSP.member')", 1, 2, "1m").
		Write(key1, value1One).
		Write(key2, value1Two).
		Write(key3, value3One)
	coll1Builder := ns1Builder.Collection(coll1)
	coll1Builder.
		TransientConfig("OR('Org1MSP.member')", 1, 2, "1m").
		Write(key1, value1One).
		Write(key2, value1Two)
	coll2Builder := ns1Builder.Collection(coll2)
	coll2Builder.
		TransientConfig("OR('Org1MSP.member')", 1, 2, "1m").
		Write(key1, value2One).
		Delete(key2)
	coll3Builder := ns1Builder.Collection(coll3)
	coll3Builder.
		StaticConfig("OR('Org1MSP.member')", 1, 2, 100).
		Write(key1, value3One)

	err = ts.Persist(txID2, 501, nil)
	require.Error(t, err, "Persist() call of transient store with nil config should throw an error")

	txPvtReadWriteSetWithConfigInfo := b.Build()
	err = ts.Persist(txID2, 501, txPvtReadWriteSetWithConfigInfo)
	require.NoError(t, err, "Persist() call of transient store should not throw an error")
	require.True(t, ts.cache.Has(txID2), "Transient store cache should have txId: %s following Persist(\"%s\", %d, &pb.TxPvtReadWriteSetWithConfigInfo{}) cache content: %+v", txID2, txID2, 501, ts.cache.GetALL())
	require.True(t, ts.blockHeightCache.Has(uint64(501)), "Transient store blockHeightCache should have blockHeight: %d following Persist(\"%s\", %d, &pb.TxPvtReadWriteSetWithConfigInfo{}) cache content: %+v", 501, txID2, 501, ts.blockHeightCache.GetALL())

	err = ts.Persist(txID3, 502, txPvtReadWriteSetWithConfigInfo)
	require.NoError(t, err, "Persist() call of transient store should not throw an error")
	require.True(t, ts.cache.Has(txID3), "Transient store cache should have txId: %s following Persist(\"%s\", %d, &pb.TxPvtReadWriteSetWithConfigInfo{}) cache content: %+v", txID3, txID3, 502, ts.cache.GetALL())
	require.True(t, ts.blockHeightCache.Has(uint64(502)), "Transient store blockHeightCache should have blockHeight: %d following Persist(\"%s\", %d, &pb.TxPvtReadWriteSetWithConfigInfo{}) cache content: %+v", 502, txID3, 502, ts.blockHeightCache.GetALL())

	err = ts.Persist(txID4, 503, txPvtReadWriteSetWithConfigInfo)
	require.NoError(t, err, "Persist() call of transient store should not throw an error")
	require.True(t, ts.cache.Has(txID4), "Transient store cache should have txId: %s following Persist(\"%s\", %d, &pb.TxPvtReadWriteSetWithConfigInfo{}) cache content: %+v", txID4, txID4, 503, ts.cache.GetALL())
	require.True(t, ts.blockHeightCache.Has(uint64(503)), "Transient store blockHeightCache should have blockHeight: %d following Persist(\"%s\", %d, &pb.TxPvtReadWriteSetWithConfigInfo{}) cache content: %+v", 503, txID4, 503, ts.blockHeightCache.GetALL())

	t.Run("Test GetTxPvtRWSetByTxid with TxPvtReadWriteSetWithConfigInfo", func(t *testing.T) {
		filter := make(ledger.PvtNsCollFilter)
		filter.Add(ns1, coll0)
		scanner, err := ts.GetTxPvtRWSetByTxid(txID2, filter)
		require.NoError(t, err, "GetTxPvtRWSetByTxid() call should not return error")
		require.NotNil(t, scanner, "GetTxPvtRWSetByTxid() call should not return a nil scanner")
		resWithConfig, err := scanner.Next()
		require.NoError(t, err, "scanner.Next() call should return error for txid \"%s\"", txID2)
		require.NotNil(t, resWithConfig, "scanner.Next() call should return a nil EndorserPvtSimulationResults for txid \"%s\" as it does contain config in the results", txID2)
	})

	t.Run("Test GetMinTransientBlkHt, PurgeByTxids and PurgeByHeight", func(t *testing.T) {
		minBlkHgt, err := ts.GetMinTransientBlkHt()
		require.NoError(t, err, "GetMinTransientBlkHt() call of transient store should not throw an error")
		require.Equal(t, uint64(500), minBlkHgt, "GetMinTransientBlkHt() call did not return expected minBlockHeight of 500")

		txToPurge := []string{txID3, txID4}
		err = ts.PurgeByTxids(txToPurge)
		require.NoError(t, err, "PurgeByTxids() call of transient store should not throw an error when purging transactions %d and %d", txToPurge)
		pvtRWSetData := ts.getTxPvtRWSetFromCache(txID3)
		blkHgt := ts.getTxidsFromBlockHeightCache(502)
		require.Empty(t, pvtRWSetData.m, "After purging 2 transactions, fetching pvt data should return empty set")
		require.Empty(t, blkHgt.m, "After purging 2 transactions, fetching block should return empty block height")

		err = ts.PurgeBelowHeight(minBlkHgt + 1) // purge minBlkHgt by setting 1 above the current minimum height
		require.NoError(t, err, "PurgeByHeight() call of tansient store should not throw an error when purging transactions by height %d", minBlkHgt)
		minBlkHgt, err = ts.GetMinTransientBlkHt()
		// minBlkHgt should now be 501 as 500 was purged
		require.NoError(t, err, "GetMinTransientBlkHt() call of transient store should not throw an error")
		require.Equal(t, uint64(501), minBlkHgt, "GetMinTransientBlkHt() call did not return expected minBlockHeight of 501 (block 500 was purged already)")
	})
	s.Close()     // noop
	ts.Shutdown() // noop
}

func TestPersistTransientStoreParallel(t *testing.T) {
	s := NewStoreProvider()
	require.NotNil(t, s, "Creating new store must not return nil")

	transientStore, err := s.OpenStore("ledgerid")
	require.NoError(t, err, "Opening transient store should not throw an error")
	require.NotNil(t, transientStore, "TransientStore should not be nil")
	tStore := transientStore.(*store)
	require.NotNil(t, tStore.cache, "Transient store's cache should not be nil")
	require.NotNil(t, tStore.blockHeightCache, "Transient store's cache should not be nil")
	require.NotNil(t, tStore.txidCache, "Transient store's cache should not be nil")

	samplePvtRWSetWithConfig := samplePvtData(t)
	// Create two private write set entry for txid-1
	endorser0SimulationResults := &origts.EndorserPvtSimulationResults{
		ReceivedAtBlockHeight:          10,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}

	endorser1SimulationResults := &origts.EndorserPvtSimulationResults{
		ReceivedAtBlockHeight:          11,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}

	// Create one private write set entry for txid-2
	endorser2SimulationResults := &origts.EndorserPvtSimulationResults{
		ReceivedAtBlockHeight:          11,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}

	// Create three private write set entry for txid-3
	endorser3SimulationResults := &origts.EndorserPvtSimulationResults{
		ReceivedAtBlockHeight:          12,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}

	endorser4SimulationResults := &origts.EndorserPvtSimulationResults{
		ReceivedAtBlockHeight:          12,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}

	endorser5SimulationResults := &origts.EndorserPvtSimulationResults{
		ReceivedAtBlockHeight:          13,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}

	tcCreate := []struct {
		pvtrwSet    *origts.EndorserPvtSimulationResults
		txid        string
		blockHeight uint64
	}{
		// Create two private write set entry for txid-1
		{
			endorser0SimulationResults,
			"txid-1",
			10,
		},
		{
			endorser1SimulationResults,
			"txid-1",
			11,
		},
		// Create one private write set entry for txid-2
		{
			endorser2SimulationResults,
			"txid-2",
			11,
		},
		// Create three private write set entry for txid-3
		{
			endorser3SimulationResults,
			"txid-3",
			12,
		},
		{
			endorser4SimulationResults,
			"txid-3",
			12,
		},
		{
			endorser5SimulationResults,
			"txid-3",
			13,
		},
	}
	for _, tt := range tcCreate {
		tt := tt
		t.Run(fmt.Sprintf("Testing parallel Create pvt transient data for txid: %s and block height: %d", tt.txid, tt.blockHeight), func(st *testing.T) {
			st.Parallel()
			err := tStore.Persist(tt.txid, tt.pvtrwSet.ReceivedAtBlockHeight,
				tt.pvtrwSet.PvtSimulationResultsWithConfig)
			require.NoError(st, err, "Persist should not fail")

		})
	}

}
func TestIterateTransientStoreParallel(t *testing.T) {
	s := NewStoreProvider()
	require.NotNil(t, s, "Creating new store must not return nil")

	transientStore, err := s.OpenStore("ledgerid")
	require.NoError(t, err, "Opening transient store should not throw an error")
	require.NotNil(t, transientStore, "TransientStore should not be nil")
	tStore := transientStore.(*store)
	require.NotNil(t, tStore.cache, "Transient store's cache should not be nil")
	require.NotNil(t, tStore.blockHeightCache, "Transient store's cache should not be nil")
	require.NotNil(t, tStore.txidCache, "Transient store's cache should not be nil")

	samplePvtRWSetWithConfig := samplePvtData(t)
	// Create two private write set entry for txid-1
	endorser0SimulationResults := &origts.EndorserPvtSimulationResults{
		ReceivedAtBlockHeight:          10,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}

	endorser1SimulationResults := &origts.EndorserPvtSimulationResults{
		ReceivedAtBlockHeight:          11,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}

	// Create one private write set entry for txid-2
	endorser2SimulationResults := &origts.EndorserPvtSimulationResults{
		ReceivedAtBlockHeight:          11,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}

	// Create three private write set entry for txid-3
	endorser3SimulationResults := &origts.EndorserPvtSimulationResults{
		ReceivedAtBlockHeight:          12,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}

	endorser4SimulationResults := &origts.EndorserPvtSimulationResults{
		ReceivedAtBlockHeight:          12,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}

	endorser5SimulationResults := &origts.EndorserPvtSimulationResults{
		ReceivedAtBlockHeight:          13,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	// create sample data in transientstore (not in t.Run)
	createSlice := []struct {
		pvtrwSet    *origts.EndorserPvtSimulationResults
		txid        string
		blockHeight uint64
	}{
		// Create two private write set entry for txid-1
		{
			endorser0SimulationResults,
			"txid-1",
			10,
		},
		{
			endorser1SimulationResults,
			"txid-1",
			11,
		},
		// Create one private write set entry for txid-2
		{
			endorser2SimulationResults,
			"txid-2",
			11,
		},
		// Create three private write set entry for txid-3
		{
			endorser3SimulationResults,
			"txid-3",
			12,
		},
		{
			endorser4SimulationResults,
			"txid-3",
			12,
		},
		{
			endorser5SimulationResults,
			"txid-3",
			13,
		},
	}
	// notice there is no t.Run() call here as we want sample data to be stored prior to test the iterators later in the test
	for _, tt := range createSlice {
		tt := tt
		err := tStore.Persist(tt.txid, tt.pvtrwSet.ReceivedAtBlockHeight,
			tt.pvtrwSet.PvtSimulationResultsWithConfig)
		require.NoError(t, err, "Persist should not fail")

	}

	// now start the scanner iterator test
	tcGet := []struct {
		txid string
	}{

		{"txid-1"},
		{"txid-2"},
		{"txid-3"},
	}
	for _, tt := range tcGet {
		tt := tt
		// prepare expectedEndorsersResults
		var expectedEndorsersResults []*origts.EndorserPvtSimulationResults
		switch tt.txid {
		case "txid-1":
			expectedEndorsersResults = append(expectedEndorsersResults, endorser0SimulationResults, endorser1SimulationResults)
		case "txid-2":
			expectedEndorsersResults = append(expectedEndorsersResults, endorser2SimulationResults)
		case "txid-3":
			expectedEndorsersResults = append(expectedEndorsersResults, endorser3SimulationResults, endorser4SimulationResults, endorser5SimulationResults)
		default: // default txid-1
			expectedEndorsersResults = append(expectedEndorsersResults, endorser0SimulationResults, endorser1SimulationResults)
		}

		t.Run(fmt.Sprintf("Testing parallel Get pvt transient data for txid: %s", tt.txid), func(st *testing.T) {
			st.Parallel()
			iter, err := tStore.GetTxPvtRWSetByTxid(tt.txid, nil)
			require.NoError(st, err, "GetTxPvtRWSetByTxid should not fail")

			// Check whether actual results and expected results are same
			var actualEndorsersResults []*origts.EndorserPvtSimulationResults
			for true {
				result, err := iter.Next()
				require.NoError(st, err, "Next should not fail")
				if result == nil {
					break
				}
				actualEndorsersResults = append(actualEndorsersResults, result)
			}
			iter.Close()

			// Note that the ordering of actualRes and expectedRes is dependent on the uuid. Hence, we are sorting
			// expectedRes and actualRes.
			sortResults(expectedEndorsersResults)
			sortResults(actualEndorsersResults)
			require.Equal(st, len(expectedEndorsersResults), len(actualEndorsersResults), "%s must return %d results", tt.txid, len(expectedEndorsersResults))
			for i, expected := range expectedEndorsersResults {
				require.Equal(st, expected.ReceivedAtBlockHeight, actualEndorsersResults[i].ReceivedAtBlockHeight)
				require.True(st, proto.Equal(expected.PvtSimulationResultsWithConfig, actualEndorsersResults[i].PvtSimulationResultsWithConfig))
			}

		})
	}
	//TODO add Purge parallel testing here..
}
