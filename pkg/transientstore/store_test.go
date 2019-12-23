/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transientstore

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/bluele/gcache"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/ledger/rwset"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric-protos-go/transientstore"
	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/util"
	origts "github.com/hyperledger/fabric/core/transientstore"
	viper "github.com/spf13/viper2015"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cm "github.com/trustbloc/fabric-peer-ext/pkg/transientstore/common"
)

// test file fully copied from: github.com/hyperledger/fabric/core/transientstore/store_helper.go
// With the following changes:
// 1 - add the following import package to reference Fabric's original structs/interfaces:
// 		origts "github.com/hyperledger/fabric/core/transientstore"
// 2 - add the following import package to reference copied structs from Fabric:
// 		cm "github.com/trustbloc/fabric-peer-ext/pkg/transientstore/common"
//
// 3 - Since the store implementation replaced leveldb with in-memory cache, this test also replaced the following:
// 		leveldb setup (env := NewTestStoreEnv(t)) with a simple call to NewStoreProvider() and OpenStore()
//

func TestMain(m *testing.M) {
	viper.Set("peer.fileSystemPath", "/tmp/fabric/core/transientdata")
	os.Exit(m.Run())
}

func TestPurgeIndexKeyCodingEncoding(t *testing.T) {
	assert := assert.New(t)
	blkHts := []uint64{0, 10, 20000}
	txids := []string{"txid", ""}
	uuids := []string{"uuid", ""}
	for _, blkHt := range blkHts {
		for _, txid := range txids {
			for _, uuid := range uuids {
				testCase := fmt.Sprintf("blkHt=%d,txid=%s,uuid=%s", blkHt, txid, uuid)
				t.Run(testCase, func(t *testing.T) {
					t.Logf("Running test case [%s]", testCase)
					purgeIndexKey := cm.CreateCompositeKeyForPurgeIndexByHeight(blkHt, txid, uuid)
					txid1, uuid1, blkHt1, err := cm.SplitCompositeKeyOfPurgeIndexByHeight(purgeIndexKey)
					require.NoError(t, err)
					assert.Equal(txid, txid1)
					assert.Equal(uuid, uuid1)
					assert.Equal(blkHt, blkHt1)
				})
			}
		}
	}
}

func TestRWSetKeyCodingEncoding(t *testing.T) {
	assert := assert.New(t)
	blkHts := []uint64{0, 10, 20000}
	txids := []string{"txid", ""}
	uuids := []string{"uuid", ""}
	for _, blkHt := range blkHts {
		for _, txid := range txids {
			for _, uuid := range uuids {
				testCase := fmt.Sprintf("blkHt=%d,txid=%s,uuid=%s", blkHt, txid, uuid)
				t.Run(testCase, func(t *testing.T) {
					t.Logf("Running test case [%s]", testCase)
					rwsetKey := cm.CreateCompositeKeyForPvtRWSet(txid, uuid, blkHt)
					uuid1, blkHt1, err := cm.SplitCompositeKeyOfPvtRWSet(rwsetKey)
					require.NoError(t, err)
					assert.Equal(uuid, uuid1)
					assert.Equal(blkHt, blkHt1)
				})
			}
		}
	}
}

func TestTransientStorePersistAndRetrieve(t *testing.T) {
	assert := assert.New(t)
	var err error
	testStoreProvider := NewStoreProvider()
	testStore, err := testStoreProvider.OpenStore("TestStore")
	assert.NoError(err)

	txid := "txid-1"
	samplePvtRWSetWithConfig := samplePvtData(t)

	// Create private simulation results for txid-1
	var endorsersResults []*origts.EndorserPvtSimulationResults

	// Results produced by endorser 1
	endorser0SimulationResults := &origts.EndorserPvtSimulationResults{
		ReceivedAtBlockHeight:          10,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	endorsersResults = append(endorsersResults, endorser0SimulationResults)

	// Results produced by endorser 2
	endorser1SimulationResults := &origts.EndorserPvtSimulationResults{
		ReceivedAtBlockHeight:          10,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	endorsersResults = append(endorsersResults, endorser1SimulationResults)

	// Persist simulation results into  store
	for i := 0; i < len(endorsersResults); i++ {
		err = testStore.Persist(txid, endorsersResults[i].ReceivedAtBlockHeight,
			endorsersResults[i].PvtSimulationResultsWithConfig)
		assert.NoError(err)
	}

	// Retrieve simulation results of txid-1 from  store
	iter, err := testStore.GetTxPvtRWSetByTxid(txid, nil)
	assert.NoError(err)

	var actualEndorsersResults []*origts.EndorserPvtSimulationResults
	for {
		result, err := iter.Next()
		assert.NoError(err)
		if result == nil {
			break
		}
		actualEndorsersResults = append(actualEndorsersResults, result)
	}
	iter.Close()
	sortResults(endorsersResults)
	sortResults(actualEndorsersResults)
	assert.Equal(endorsersResults, actualEndorsersResults)
}

func TestTransientStorePersistAndRetrieveBothOldAndNewProto(t *testing.T) {
	assert := assert.New(t)
	var err error

	testStoreProvider := NewStoreProvider()
	testStore, err := testStoreProvider.OpenStore("TestStore")
	assert.NoError(err)

	txid := "txid-1"
	var receivedAtBlockHeight uint64 = 10

	// Create and persist private simulation results with old proto for txid-1
	samplePvtRWSet := samplePvtData(t)
	err = testStore.Persist(txid, receivedAtBlockHeight, samplePvtRWSet)
	assert.NoError(err)

	// Create and persist private simulation results with new proto for txid-1
	samplePvtRWSetWithConfig := samplePvtData(t)
	err = testStore.Persist(txid, receivedAtBlockHeight, samplePvtRWSetWithConfig)
	assert.NoError(err)

	// Construct the expected results
	var expectedEndorsersResults []*origts.EndorserPvtSimulationResults

	endorser0SimulationResults := &origts.EndorserPvtSimulationResults{
		ReceivedAtBlockHeight:          receivedAtBlockHeight,
		PvtSimulationResultsWithConfig: samplePvtRWSet,
	}
	expectedEndorsersResults = append(expectedEndorsersResults, endorser0SimulationResults)

	endorser1SimulationResults := &origts.EndorserPvtSimulationResults{
		ReceivedAtBlockHeight:          receivedAtBlockHeight,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	expectedEndorsersResults = append(expectedEndorsersResults, endorser1SimulationResults)

	// Retrieve simulation results of txid-1 from  store
	iter, err := testStore.GetTxPvtRWSetByTxid(txid, nil)
	assert.NoError(err)

	var actualEndorsersResults []*origts.EndorserPvtSimulationResults
	for {
		result, err := iter.Next()
		assert.NoError(err)
		if result == nil {
			break
		}
		actualEndorsersResults = append(actualEndorsersResults, result)
	}
	iter.Close()
	sortResults(expectedEndorsersResults)
	sortResults(actualEndorsersResults)
	assert.Equal(expectedEndorsersResults, actualEndorsersResults)
}

func TestTransientStorePurgeByTxids(t *testing.T) {
	assert := assert.New(t)
	var err error

	testStoreProvider := NewStoreProvider()
	testStore, err := testStoreProvider.OpenStore("TestStore")
	assert.NoError(err)

	var txids []string
	var endorsersResults []*origts.EndorserPvtSimulationResults

	samplePvtRWSetWithConfig := samplePvtData(t)

	// Create two private write set entry for txid-1
	txids = append(txids, "txid-1")
	endorser0SimulationResults := &origts.EndorserPvtSimulationResults{
		ReceivedAtBlockHeight:          10,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	endorsersResults = append(endorsersResults, endorser0SimulationResults)

	txids = append(txids, "txid-1")
	endorser1SimulationResults := &origts.EndorserPvtSimulationResults{
		ReceivedAtBlockHeight:          11,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	endorsersResults = append(endorsersResults, endorser1SimulationResults)

	// Create one private write set entry for txid-2
	txids = append(txids, "txid-2")
	endorser2SimulationResults := &origts.EndorserPvtSimulationResults{
		ReceivedAtBlockHeight:          11,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	endorsersResults = append(endorsersResults, endorser2SimulationResults)

	// Create three private write set entry for txid-3
	txids = append(txids, "txid-3")
	endorser3SimulationResults := &origts.EndorserPvtSimulationResults{
		ReceivedAtBlockHeight:          12,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	endorsersResults = append(endorsersResults, endorser3SimulationResults)

	txids = append(txids, "txid-3")
	endorser4SimulationResults := &origts.EndorserPvtSimulationResults{
		ReceivedAtBlockHeight:          12,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	endorsersResults = append(endorsersResults, endorser4SimulationResults)

	txids = append(txids, "txid-3")
	endorser5SimulationResults := &origts.EndorserPvtSimulationResults{
		ReceivedAtBlockHeight:          13,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	endorsersResults = append(endorsersResults, endorser5SimulationResults)

	for i := 0; i < len(txids); i++ {
		err = testStore.Persist(txids[i], endorsersResults[i].ReceivedAtBlockHeight,
			endorsersResults[i].PvtSimulationResultsWithConfig)
		assert.NoError(err)
	}

	// Retrieve simulation results of txid-2 from  store
	iter, err := testStore.GetTxPvtRWSetByTxid("txid-2", nil)
	assert.NoError(err)

	// Expected results for txid-2
	var expectedEndorsersResults []*origts.EndorserPvtSimulationResults
	expectedEndorsersResults = append(expectedEndorsersResults, endorser2SimulationResults)

	// Check whether actual results and expected results are same
	var actualEndorsersResults []*origts.EndorserPvtSimulationResults
	for true {
		result, err := iter.Next()
		assert.NoError(err)
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

	assert.Equal(len(expectedEndorsersResults), len(actualEndorsersResults))
	for i, expected := range expectedEndorsersResults {
		assert.Equal(expected.ReceivedAtBlockHeight, actualEndorsersResults[i].ReceivedAtBlockHeight)
		assert.True(proto.Equal(expected.PvtSimulationResultsWithConfig, actualEndorsersResults[i].PvtSimulationResultsWithConfig))
	}

	// Remove all private write set of txid-2 and txid-3
	toRemoveTxids := []string{"txid-2", "txid-3"}
	err = testStore.PurgeByTxids(toRemoveTxids)
	assert.NoError(err)

	for _, txid := range toRemoveTxids {

		// Check whether private write sets of txid-2 are removed
		var expectedEndorsersResults *origts.EndorserPvtSimulationResults
		expectedEndorsersResults = nil
		iter, err = testStore.GetTxPvtRWSetByTxid(txid, nil)
		assert.NoError(err)
		// Should return nil, nil
		result, err := iter.Next()
		assert.NoError(err)
		assert.Equal(expectedEndorsersResults, result)
	}

	// Retrieve simulation results of txid-1 from store
	iter, err = testStore.GetTxPvtRWSetByTxid("txid-1", nil)
	assert.NoError(err)

	// Expected results for txid-1
	expectedEndorsersResults = nil
	expectedEndorsersResults = append(expectedEndorsersResults, endorser0SimulationResults)
	expectedEndorsersResults = append(expectedEndorsersResults, endorser1SimulationResults)

	// Check whether actual results and expected results are same
	actualEndorsersResults = nil
	for true {
		result, err := iter.Next()
		assert.NoError(err)
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

	assert.Equal(len(expectedEndorsersResults), len(actualEndorsersResults))
	for i, expected := range expectedEndorsersResults {
		assert.Equal(expected.ReceivedAtBlockHeight, actualEndorsersResults[i].ReceivedAtBlockHeight)
		assert.True(proto.Equal(expected.PvtSimulationResultsWithConfig, actualEndorsersResults[i].PvtSimulationResultsWithConfig))
	}

	toRemoveTxids = []string{"txid-1"}
	err = testStore.PurgeByTxids(toRemoveTxids)
	assert.NoError(err)

	for _, txid := range toRemoveTxids {

		// Check whether private write sets of txid-1 are removed
		var expectedEndorsersResults *origts.EndorserPvtSimulationResults
		expectedEndorsersResults = nil
		iter, err = testStore.GetTxPvtRWSetByTxid(txid, nil)
		assert.NoError(err)
		// Should return nil, nil
		result, err := iter.Next()
		assert.NoError(err)
		assert.Equal(expectedEndorsersResults, result)
	}

	// There should be no entries in the  store
	_, err = testStore.GetMinTransientBlkHt()
	assert.Equal(err, ErrStoreEmpty)
}

func TestTransientStorePurgeByHeight(t *testing.T) {
	var err error
	assert := assert.New(t)

	testStoreProvider := NewStoreProvider()
	testStore, err := testStoreProvider.OpenStore("TestStore")
	assert.NoError(err)

	txid := "txid-1"
	samplePvtRWSetWithConfig := samplePvtData(t)

	// Create private simulation results for txid-1
	var endorsersResults []*origts.EndorserPvtSimulationResults

	// Results produced by endorser 1
	endorser0SimulationResults := &origts.EndorserPvtSimulationResults{
		ReceivedAtBlockHeight:          10,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	endorsersResults = append(endorsersResults, endorser0SimulationResults)

	// Results produced by endorser 2
	endorser1SimulationResults := &origts.EndorserPvtSimulationResults{
		ReceivedAtBlockHeight:          11,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	endorsersResults = append(endorsersResults, endorser1SimulationResults)

	// Results produced by endorser 3
	endorser2SimulationResults := &origts.EndorserPvtSimulationResults{
		ReceivedAtBlockHeight:          12,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	endorsersResults = append(endorsersResults, endorser2SimulationResults)

	// Results produced by endorser 3
	endorser3SimulationResults := &origts.EndorserPvtSimulationResults{
		ReceivedAtBlockHeight:          12,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	endorsersResults = append(endorsersResults, endorser3SimulationResults)

	// Results produced by endorser 4
	endorser4SimulationResults := &origts.EndorserPvtSimulationResults{
		ReceivedAtBlockHeight:          13,
		PvtSimulationResultsWithConfig: samplePvtRWSetWithConfig,
	}
	endorsersResults = append(endorsersResults, endorser4SimulationResults)

	// Persist simulation results into  store
	for i := 0; i < 5; i++ {
		err = testStore.Persist(txid, endorsersResults[i].ReceivedAtBlockHeight,
			endorsersResults[i].PvtSimulationResultsWithConfig)
		assert.NoError(err)
	}

	// Retain results generate at block height greater than or equal to 12
	minTransientBlkHtToRetain := uint64(12)
	err = testStore.PurgeBelowHeight(minTransientBlkHtToRetain)
	assert.NoError(err)

	// Retrieve simulation results of txid-1 from  store
	iter, err := testStore.GetTxPvtRWSetByTxid(txid, nil)
	assert.NoError(err)

	// Expected results for txid-1
	var expectedEndorsersResults []*origts.EndorserPvtSimulationResults
	expectedEndorsersResults = append(expectedEndorsersResults, endorser2SimulationResults) //endorsed at height 12
	expectedEndorsersResults = append(expectedEndorsersResults, endorser3SimulationResults) //endorsed at height 12
	expectedEndorsersResults = append(expectedEndorsersResults, endorser4SimulationResults) //endorsed at height 13

	// Check whether actual results and expected results are same
	var actualEndorsersResults []*origts.EndorserPvtSimulationResults
	for true {
		result, err := iter.Next()
		assert.NoError(err)
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

	assert.Equal(len(expectedEndorsersResults), len(actualEndorsersResults))
	for i, expected := range expectedEndorsersResults {
		assert.Equal(expected.ReceivedAtBlockHeight, actualEndorsersResults[i].ReceivedAtBlockHeight)
		assert.True(proto.Equal(expected.PvtSimulationResultsWithConfig, actualEndorsersResults[i].PvtSimulationResultsWithConfig))
	}

	// Get the minimum block height remaining in transient store
	var actualMinTransientBlkHt uint64
	actualMinTransientBlkHt, err = testStore.GetMinTransientBlkHt()
	assert.NoError(err)
	assert.Equal(minTransientBlkHtToRetain, actualMinTransientBlkHt)

	// Retain results at block height greater than or equal to 15
	minTransientBlkHtToRetain = uint64(15)
	err = testStore.PurgeBelowHeight(minTransientBlkHtToRetain)
	assert.NoError(err)

	// There should be no entries in the  store
	actualMinTransientBlkHt, err = testStore.GetMinTransientBlkHt()
	assert.Equal(err, ErrStoreEmpty)

	// Retain results at block height greater than or equal to 15
	minTransientBlkHtToRetain = uint64(15)
	err = testStore.PurgeBelowHeight(minTransientBlkHtToRetain)
	// Should not return any error
	assert.NoError(err)
}

func TestTransientStoreRetrievalWithFilter(t *testing.T) {
	var err error

	testStoreProvider := NewStoreProvider()
	store, err := testStoreProvider.OpenStore("TestStore")
	assert.NoError(t, err)

	samplePvtSimResWithConfig := samplePvtData(t)

	testTxid := "testTxid"
	numEntries := 5
	for i := 0; i < numEntries; i++ {
		store.Persist(testTxid, uint64(i), samplePvtSimResWithConfig)
	}

	filter := ledger.NewPvtNsCollFilter()
	filter.Add("ns-1", "coll-1")
	filter.Add("ns-2", "coll-2")

	itr, err := store.GetTxPvtRWSetByTxid(testTxid, filter)
	assert.NoError(t, err)

	var actualRes []*origts.EndorserPvtSimulationResults
	for {
		res, err := itr.Next()
		if res == nil || err != nil {
			assert.NoError(t, err)
			break
		}
		actualRes = append(actualRes, res)
	}

	// prepare the trimmed pvtrwset manually - retain only "ns-1/coll-1" and "ns-2/coll-2"
	expectedSimulationRes := samplePvtSimResWithConfig
	expectedSimulationRes.GetPvtRwset().NsPvtRwset[0].CollectionPvtRwset = expectedSimulationRes.GetPvtRwset().NsPvtRwset[0].CollectionPvtRwset[0:1]
	expectedSimulationRes.GetPvtRwset().NsPvtRwset[1].CollectionPvtRwset = expectedSimulationRes.GetPvtRwset().NsPvtRwset[1].CollectionPvtRwset[1:]
	expectedSimulationRes.CollectionConfigs, err = cm.TrimPvtCollectionConfigs(expectedSimulationRes.CollectionConfigs, filter)
	assert.NoError(t, err)
	for ns, colName := range map[string]string{"ns-1": "coll-1", "ns-2": "coll-2"} {
		config := expectedSimulationRes.CollectionConfigs[ns]
		assert.NotNil(t, config)
		ns1Config := config.Config
		assert.Equal(t, len(ns1Config), 1)
		ns1ColConfig := ns1Config[0].GetStaticCollectionConfig()
		assert.NotNil(t, ns1ColConfig.Name, colName)
	}

	var expectedRes []*origts.EndorserPvtSimulationResults
	for i := 0; i < numEntries; i++ {
		expectedRes = append(expectedRes, &origts.EndorserPvtSimulationResults{uint64(i), expectedSimulationRes})
	}

	// Note that the ordering of actualRes and expectedRes is dependent on the uuid. Hence, we are sorting
	// expectedRes and actualRes.
	sortResults(expectedRes)
	sortResults(actualRes)
	assert.Equal(t, len(expectedRes), len(actualRes))
	for i, expected := range expectedRes {
		assert.Equal(t, expected.ReceivedAtBlockHeight, actualRes[i].ReceivedAtBlockHeight)
		assert.True(t, proto.Equal(expected.PvtSimulationResultsWithConfig, actualRes[i].PvtSimulationResultsWithConfig))
	}

}

func sortResults(res []*origts.EndorserPvtSimulationResults) {
	// Results are sorted by ascending order of received at block height. When the block
	// heights are same, we sort by comparing the hash of private write set.
	var sortCondition = func(i, j int) bool {
		if res[i].ReceivedAtBlockHeight == res[j].ReceivedAtBlockHeight {
			res_i, _ := proto.Marshal(res[i].PvtSimulationResultsWithConfig)
			res_j, _ := proto.Marshal(res[j].PvtSimulationResultsWithConfig)
			// if hashes are same, any order would work.
			return string(util.ComputeHash(res_i)) < string(util.ComputeHash(res_j))
		}
		return res[i].ReceivedAtBlockHeight < res[j].ReceivedAtBlockHeight
	}
	sort.SliceStable(res, sortCondition)
}

func samplePvtDatax(t *testing.T) *rwset.TxPvtReadWriteSet {
	pvtWriteSet := &rwset.TxPvtReadWriteSet{DataModel: rwset.TxReadWriteSet_KV}
	pvtWriteSet.NsPvtRwset = []*rwset.NsPvtReadWriteSet{
		{
			Namespace: "ns-1",
			CollectionPvtRwset: []*rwset.CollectionPvtReadWriteSet{
				{
					CollectionName: "coll-1",
					Rwset:          []byte("RandomBytes-PvtRWSet-ns1-coll1"),
				},
				{
					CollectionName: "coll-2",
					Rwset:          []byte("RandomBytes-PvtRWSet-ns1-coll2"),
				},
			},
		},

		{
			Namespace: "ns-2",
			CollectionPvtRwset: []*rwset.CollectionPvtReadWriteSet{
				{
					CollectionName: "coll-1",
					Rwset:          []byte("RandomBytes-PvtRWSet-ns2-coll1"),
				},
				{
					CollectionName: "coll-2",
					Rwset:          []byte("RandomBytes-PvtRWSet-ns2-coll2"),
				},
			},
		},
	}
	return pvtWriteSet
}

func samplePvtData(t *testing.T) *transientstore.TxPvtReadWriteSetWithConfigInfo {
	pvtWriteSet := samplePvtDatax(t)
	pvtRWSetWithConfigInfo := &transientstore.TxPvtReadWriteSetWithConfigInfo{
		PvtRwset: pvtWriteSet,
		CollectionConfigs: map[string]*pb.CollectionConfigPackage{
			"ns-1": {
				Config: []*pb.CollectionConfig{
					sampleCollectionConfigPackage("coll-1"),
					sampleCollectionConfigPackage("coll-2"),
				},
			},
			"ns-2": {
				Config: []*pb.CollectionConfig{
					sampleCollectionConfigPackage("coll-1"),
					sampleCollectionConfigPackage("coll-2"),
				},
			},
		},
	}
	return pvtRWSetWithConfigInfo
}

func createCollectionConfig(collectionName string, signaturePolicyEnvelope *common.SignaturePolicyEnvelope,
	requiredPeerCount int32, maximumPeerCount int32,
) *pb.CollectionConfig {
	signaturePolicy := &pb.CollectionPolicyConfig_SignaturePolicy{
		SignaturePolicy: signaturePolicyEnvelope,
	}
	accessPolicy := &pb.CollectionPolicyConfig{
		Payload: signaturePolicy,
	}

	return &pb.CollectionConfig{
		Payload: &pb.CollectionConfig_StaticCollectionConfig{
			StaticCollectionConfig: &pb.StaticCollectionConfig{
				Name:              collectionName,
				MemberOrgsPolicy:  accessPolicy,
				RequiredPeerCount: requiredPeerCount,
				MaximumPeerCount:  maximumPeerCount,
			},
		},
	}
}

func sampleCollectionConfigPackage(colName string) *pb.CollectionConfig {
	var signers = [][]byte{[]byte("signer0"), []byte("signer1")}
	policyEnvelope := cauthdsl.Envelope(cauthdsl.Or(cauthdsl.SignedBy(0), cauthdsl.SignedBy(1)), signers)

	var requiredPeerCount, maximumPeerCount int32
	requiredPeerCount = 1
	maximumPeerCount = 2

	return createCollectionConfig(colName, policyEnvelope, requiredPeerCount, maximumPeerCount)
}

func TestBadUpdateTxidCache(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
			return
		}
	}()

	s := newStore()
	s.cache = &mockBadCache{}
	s.txidCache = &mockBadCache{}

	s.updateTxidCache("000", 0)
}

func TestBadGetTxPvtRWSetFromCache(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	s := newStore()
	s.cache = &mockBadCache{}

	s.getTxPvtRWSetFromCache("000")
}

func TestBadGetTxidsFromBlockHeightCache(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	s := newStore()
	s.blockHeightCache = &mockBadCache{}

	s.getTxidsFromBlockHeightCache(0)
}

func TestBadGetBlockHeightKeysFromTxidCache(t *testing.T) {
	s := newStore()
	s.txidCache = &mockBadCache{}

	v, e := s.getBlockHeightKeysFromTxidCache([]string{"0", "1"})
	assert.Error(t, e)
	assert.Empty(t, v)
	e = s.purgeBlockHeightCacheByTxids([]string{"0", "1"})
	assert.Error(t, e)
}

func TestBadGetTxPvtRWSetByTxid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	s := newStore()
	s.cache = &mockBadCache{}

	s.GetTxPvtRWSetByTxid("0", nil)
}

func TestEmptyGetTxPvtRWSetByTxid(t *testing.T) {
	s := newStore()

	v, e := s.GetTxPvtRWSetByTxid("0", nil)
	assert.NoError(t, e)
	assert.Empty(t, v)
}

func TestGetBlockHeightKeysFromTxidCache(t *testing.T) {
	s := newStore()
	v, e := s.getBlockHeightKeysFromTxidCache([]string{"0"})
	assert.NoError(t, e)
	assert.Empty(t, v)
	s.setTxidToBlockHeightCache("0", 1)
}

func TestPurgeTxRWSetCacheByBlockHeight(t *testing.T) {
	txID := "tx1"

	s := newStore()

	keyBytes := cm.CreateCompositeKeyForPvtRWSet(txID, "uuid", 1000)
	key := hex.EncodeToString(keyBytes)

	t.Run("Invalid hex key", func(t *testing.T) {
		s.cache.Set(txID, &pvtRWSetMap{
			m: map[string]string{"key1": "val1"},
		})
		err := s.purgeTxRWSetCacheByBlockHeight([]string{txID}, 1000)
		require.Error(t, err)
		require.Contains(t, err.Error(), "encoding/hex: invalid byte")
	})

	t.Run("splitCompositeKeyOfPvtRWSet error", func(t *testing.T) {
		errExpected := errors.New("split composite key error")
		restore := splitCompositeKeyOfPvtRWSet
		splitCompositeKeyOfPvtRWSet = func([]byte) (uuid string, blockHeight uint64, err error) {
			return "", 0, errExpected
		}
		defer func() { splitCompositeKeyOfPvtRWSet = restore }()

		s.cache.Set(txID, &pvtRWSetMap{
			m: map[string]string{key: "val1"},
		})
		err := s.purgeTxRWSetCacheByBlockHeight([]string{txID}, 1000)
		require.EqualError(t, err, errExpected.Error())
	})
}

type mockBadCache struct {
	gcache.Cache // necessary for unexported get() interface function
}

func (mockBadCache) Set(interface{}, interface{}) error {
	return errors.New("bad set error")
}

func (mockBadCache) SetWithExpire(interface{}, interface{}, time.Duration) error {
	panic("implement me")
}

func (mockBadCache) Get(interface{}) (interface{}, error) {
	return nil, errors.New("bad error")
}

func (mockBadCache) GetIFPresent(interface{}) (interface{}, error) {
	panic("implement me")
}

func (mockBadCache) GetALL() map[interface{}]interface{} {
	panic("implement me")
}

func (mockBadCache) Has(interface{}) bool {
	panic("implement me")
}

func (mockBadCache) Remove(interface{}) bool {
	panic("implement me")
}

func (mockBadCache) Purge() {
	panic("implement me")
}

func (mockBadCache) Keys() []interface{} {
	panic("implement me")
}

func (mockBadCache) Len() int {
	panic("implement me")
}

func (mockBadCache) HitCount() uint64 {
	panic("implement me")
}

func (mockBadCache) MissCount() uint64 {
	panic("implement me")
}

func (mockBadCache) LookupCount() uint64 {
	panic("implement me")
}

func (mockBadCache) HitRate() float64 {
	panic("implement me")
}

func TestRwsetScanner_Next(t *testing.T) {
	rs := RwsetScanner{}
	n, e := rs.Next()
	assert.NoError(t, e)
	assert.Empty(t, n)

	rs = RwsetScanner{
		next: 1,
		results: []keyValue{
			{"k1", "v1"},
			{"k2", "v2"},
		},
	}
	n, e = rs.Next()
	assert.Error(t, e)
	assert.Empty(t, n)

	rs = RwsetScanner{
		next: 1,
		results: []keyValue{
			{"01000100020003", "v1"},
			{"02000100020003", "v2"},
		},
	}
	n, e = rs.Next()
	assert.Error(t, e)
	assert.Empty(t, n)
}
