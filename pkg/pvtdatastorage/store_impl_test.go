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
	"github.com/hyperledger/fabric/core/ledger/pvtdatapolicy"
	btltestutil "github.com/hyperledger/fabric/core/ledger/pvtdatapolicy/testutil"
	"github.com/hyperledger/fabric/core/ledger/pvtdatastorage"
	"github.com/hyperledger/fabric/core/ledger/util/couchdb"
	"github.com/hyperledger/fabric/extensions/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	xtestutil "github.com/trustbloc/fabric-peer-ext/pkg/testutil"
)

var couchDBConfig *couchdb.Config

type mockProvider struct {
	openStoreValue pvtdatastorage.Store
	openStoreErr   error
}

func (m mockProvider) OpenStore(id string) (pvtdatastorage.Store, error) {
	return m.openStoreValue, m.openStoreErr
}

func (m mockProvider) Close() {

}

type mockStore struct {
	isEmptyValue                                bool
	isEmptyErr                                  error
	lastCommittedBlockHeightValue               uint64
	lastCommittedBlockHeightErr                 error
	initLastCommittedBlockErr                   error
	commitErr                                   error
	rollbackErr                                 error
	getMissingPvtDataInfoForMostRecentBlocksErr error
	processCollsEligibilityEnabledErr           error
	commitPvtDataOfOldBlocksErr                 error
	getLastUpdatedOldBlocksPvtDataErr           error
	resetLastUpdatedOldBlocksListErr            error
}

func (m mockStore) Init(btlPolicy pvtdatapolicy.BTLPolicy) {

}

func (m mockStore) InitLastCommittedBlock(blockNum uint64) error {
	return m.initLastCommittedBlockErr
}

func (m mockStore) GetPvtDataByBlockNum(blockNum uint64, filter ledger.PvtNsCollFilter) ([]*ledger.TxPvtData, error) {
	return nil, nil
}

func (m mockStore) GetMissingPvtDataInfoForMostRecentBlocks(maxBlock int) (ledger.MissingPvtDataInfo, error) {
	return nil, m.getMissingPvtDataInfoForMostRecentBlocksErr
}

func (m mockStore) Prepare(blockNum uint64, pvtData []*ledger.TxPvtData, missingPvtData ledger.TxMissingPvtDataMap) error {
	return nil
}

func (m mockStore) Commit() error {
	return m.commitErr
}

func (m mockStore) Rollback() error {
	return m.rollbackErr
}

func (m mockStore) ProcessCollsEligibilityEnabled(committingBlk uint64, nsCollMap map[string][]string) error {
	return m.processCollsEligibilityEnabledErr
}

func (m mockStore) CommitPvtDataOfOldBlocks(blocksPvtData map[uint64][]*ledger.TxPvtData) error {
	return m.commitPvtDataOfOldBlocksErr
}

func (m mockStore) GetLastUpdatedOldBlocksPvtData() (map[uint64][]*ledger.TxPvtData, error) {
	return nil, m.getLastUpdatedOldBlocksPvtDataErr
}

func (m mockStore) ResetLastUpdatedOldBlocksList() error {
	return m.resetLastUpdatedOldBlocksListErr
}
func (m mockStore) IsEmpty() (bool, error) {
	return m.isEmptyValue, m.isEmptyErr
}
func (m mockStore) LastCommittedBlockHeight() (uint64, error) {
	return m.lastCommittedBlockHeightValue, m.lastCommittedBlockHeightErr
}
func (m mockStore) HasPendingBatch() (bool, error) {
	return false, nil
}
func (m mockStore) Shutdown() {

}

func TestOpenStore(t *testing.T) {
	t.Run("test error from storageProvider OpenStore", func(t *testing.T) {
		removeStorePath()
		conf := testutil.TestLedgerConf().PrivateData
		testStoreProvider := NewProvider(conf, testutil.TestLedgerConf())
		testStoreProvider.storageProvider = &mockProvider{openStoreErr: fmt.Errorf("openStore error")}
		_, err := testStoreProvider.OpenStore("")
		require.Error(t, err)
		require.Contains(t, err.Error(), "openStore error")
	})

	t.Run("test error from cacheProvider OpenStore", func(t *testing.T) {
		removeStorePath()
		conf := testutil.TestLedgerConf().PrivateData
		testStoreProvider := NewProvider(conf, testutil.TestLedgerConf())
		testStoreProvider.storageProvider = &mockProvider{}
		testStoreProvider.cacheProvider = &mockProvider{openStoreErr: fmt.Errorf("openStore error")}
		_, err := testStoreProvider.OpenStore("")
		require.Error(t, err)
		require.Contains(t, err.Error(), "openStore error")
	})

}

func TestNewPvtDataStore(t *testing.T) {
	t.Run("test error from pvtDataDBStore IsEmpty", func(t *testing.T) {
		_, err := newPvtDataStore(&mockStore{isEmptyErr: fmt.Errorf("isEmpty error")}, &mockStore{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "isEmpty error")

	})

	t.Run("test error from pvtDataDBStore LastCommittedBlockHeight", func(t *testing.T) {
		_, err := newPvtDataStore(&mockStore{isEmptyValue: false, lastCommittedBlockHeightErr: fmt.Errorf("lastCommittedBlockHeight error")}, &mockStore{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "lastCommittedBlockHeight error")
	})

	t.Run("test error from cachePvtDataStore InitLastCommittedBlock", func(t *testing.T) {
		_, err := newPvtDataStore(&mockStore{isEmptyValue: false, lastCommittedBlockHeightValue: 1},
			&mockStore{initLastCommittedBlockErr: fmt.Errorf("initLastCommittedBlock error")})
		require.Error(t, err)
		require.Contains(t, err.Error(), "initLastCommittedBlock error")
	})

}

func TestCommit(t *testing.T) {
	t.Run("test error from cachePvtDataStore commit", func(t *testing.T) {
		env := NewTestStoreEnv(t, "testerrorfromcachepvtdatastorecommit", nil, couchDBConfig)
		store := env.TestStore
		s := store.(*pvtDataStore)
		s.cachePvtDataStore = &mockStore{commitErr: fmt.Errorf("commit error")}
		err := s.Commit()
		require.Error(t, err)
		require.Contains(t, err.Error(), "commit error")

	})
}

func TestRollback(t *testing.T) {
	t.Run("test error from cachePvtDataStore rollback", func(t *testing.T) {
		env := NewTestStoreEnv(t, "testerrorfromcachepvtdatastorerollback", nil, couchDBConfig)
		store := env.TestStore
		s := store.(*pvtDataStore)
		s.cachePvtDataStore = &mockStore{rollbackErr: fmt.Errorf("rollback error")}
		err := s.Rollback()
		require.Error(t, err)
		require.Contains(t, err.Error(), "rollback error")

	})
}

func TestShutdown(t *testing.T) {
	env := NewTestStoreEnv(t, "testshutdown", nil, couchDBConfig)
	store := env.TestStore
	s := store.(*pvtDataStore)
	s.cachePvtDataStore = &mockStore{}
	s.pvtDataDBStore = &mockStore{}
	s.Shutdown()

}

func TestGetMissingPvtDataInfoForMostRecentBlocks(t *testing.T) {
	t.Run("test error from pvtDataDBStore GetMissingPvtDataInfoForMostRecentBlocks", func(t *testing.T) {
		env := NewTestStoreEnv(t, "ledger", nil, couchDBConfig)
		store := env.TestStore
		s := store.(*pvtDataStore)
		s.pvtDataDBStore = &mockStore{getMissingPvtDataInfoForMostRecentBlocksErr: fmt.Errorf("getMissingPvtDataInfoForMostRecentBlocks error")}
		_, err := s.GetMissingPvtDataInfoForMostRecentBlocks(0)
		require.Error(t, err)
		require.Contains(t, err.Error(), "getMissingPvtDataInfoForMostRecentBlocks error")

	})
}

func TestProcessCollsEligibilityEnabled(t *testing.T) {
	t.Run("test error from pvtDataDBStore ProcessCollsEligibilityEnabled", func(t *testing.T) {
		env := NewTestStoreEnv(t, "ledger", nil, couchDBConfig)
		store := env.TestStore
		s := store.(*pvtDataStore)
		s.pvtDataDBStore = &mockStore{processCollsEligibilityEnabledErr: fmt.Errorf("processCollsEligibilityEnabled error")}
		err := s.ProcessCollsEligibilityEnabled(0, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "processCollsEligibilityEnabled error")

	})
}

func TestCommitPvtDataOfOldBlocks(t *testing.T) {
	t.Run("test error from pvtDataDBStore CommitPvtDataOfOldBlocks", func(t *testing.T) {
		env := NewTestStoreEnv(t, "ledger", nil, couchDBConfig)
		store := env.TestStore
		s := store.(*pvtDataStore)
		s.pvtDataDBStore = &mockStore{commitPvtDataOfOldBlocksErr: fmt.Errorf("commitPvtDataOfOldBlocks error")}
		err := s.CommitPvtDataOfOldBlocks(nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "commitPvtDataOfOldBlocks error")

	})
}

func TestGetLastUpdatedOldBlocksPvtData(t *testing.T) {
	t.Run("test error from pvtDataDBStore GetLastUpdatedOldBlocksPvtData", func(t *testing.T) {
		env := NewTestStoreEnv(t, "ledger", nil, couchDBConfig)
		store := env.TestStore
		s := store.(*pvtDataStore)
		s.pvtDataDBStore = &mockStore{getLastUpdatedOldBlocksPvtDataErr: fmt.Errorf("getLastUpdatedOldBlocksPvtData error")}
		_, err := s.GetLastUpdatedOldBlocksPvtData()
		require.Error(t, err)
		require.Contains(t, err.Error(), "getLastUpdatedOldBlocksPvtData error")

	})
}

func TestResetLastUpdatedOldBlocksList(t *testing.T) {
	t.Run("test error from pvtDataDBStore ResetLastUpdatedOldBlocksList", func(t *testing.T) {
		env := NewTestStoreEnv(t, "ledger", nil, couchDBConfig)
		store := env.TestStore
		s := store.(*pvtDataStore)
		s.pvtDataDBStore = &mockStore{resetLastUpdatedOldBlocksListErr: fmt.Errorf("resetLastUpdatedOldBlocksList error")}
		err := s.ResetLastUpdatedOldBlocksList()
		require.Error(t, err)
		require.Contains(t, err.Error(), "resetLastUpdatedOldBlocksList error")

	})
}

func TestMain(m *testing.M) {
	//setup extension test environment
	_, _, destroy := xtestutil.SetupExtTestEnv()

	viper.Set("peer.fileSystemPath", "/tmp/fabric/core/ledger/pvtdatastore")

	// Create CouchDB definition from config parameters
	couchDBConfig = xtestutil.TestLedgerConf().StateDB.CouchDB

	code := m.Run()
	destroy()
	os.Exit(code)
}

func TestEmptyStore(t *testing.T) {
	env := NewTestStoreEnv(t, "testempty", nil, couchDBConfig)
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

	env := NewTestStoreEnv(t, "testbasiccommitandretrieval", btlPolicy, couchDBConfig)
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
	env := NewTestStoreEnv(t, "teststate", btlPolicy, couchDBConfig)
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
	env := NewTestStoreEnv(t, "testinitlastcommittedblock", nil, couchDBConfig)
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
