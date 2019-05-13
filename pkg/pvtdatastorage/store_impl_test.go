/*
Copyright IBM Corp, SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pvtdatastorage

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	btltestutil "github.com/hyperledger/fabric/core/ledger/pvtdatapolicy/testutil"
	"github.com/hyperledger/fabric/core/ledger/pvtdatastorage"
	"github.com/hyperledger/fabric/core/ledger/util/couchdb"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	xtestutil "github.com/trustbloc/fabric-peer-ext/pkg/testutil"
)

var couchDBDef *couchdb.CouchDBDef

func TestMain(m *testing.M) {
	//setup extension test environment
	_, _, destroy := xtestutil.SetupExtTestEnv()

	viper.Set("peer.fileSystemPath", "/tmp/fabric/core/ledger/pvtdatastore")

	// Create CouchDB definition from config parameters
	couchDBDef = couchdb.GetCouchDBDefinition()

	code := m.Run()
	destroy()
	os.Exit(code)
}

func TestEmptyStore(t *testing.T) {
	env := NewTestStoreEnv(t, "testempty", nil, couchDBDef)
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

	env := NewTestStoreEnv(t, "testbasiccommitandretrieval", btlPolicy, couchDBDef)
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
	env := NewTestStoreEnv(t, "teststate", btlPolicy, couchDBDef)
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
	env := NewTestStoreEnv(t, "testinitlastcommittedblock", nil, couchDBDef)
	req := require.New(t)
	store := env.TestStore
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

func testLastCommittedBlockHeight(expectedBlockHt uint64, req *require.Assertions, store pvtdatastorage.Store) {
	blkHt, err := store.LastCommittedBlockHeight()
	req.NoError(err)
	req.Equal(expectedBlockHt, blkHt)
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
