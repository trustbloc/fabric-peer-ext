/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cdbblkstorage

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/common/ledger/blkstorage"
	"github.com/hyperledger/fabric/common/ledger/testutil"
	"github.com/hyperledger/fabric/common/metrics/disabled"
	couchdb "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"
	"github.com/hyperledger/fabric/extensions/ledger/api"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/blkstorage/cdbblkstorage/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/txflags"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
	xtestutil "github.com/trustbloc/fabric-peer-ext/pkg/testutil"
)

//go:generate counterfeiter -o ./mocks/couchdb.gen.go -fake-name CouchDB . couchDB

var cdbInstance *couchdb.CouchInstance

func TestMain(m *testing.M) {

	//setup extension test environment
	_, _, destroy := xtestutil.SetupExtTestEnv()

	couchDBConfig := xtestutil.TestLedgerConf().StateDBConfig.CouchDB
	var err error
	cdbInstance, err = couchdb.CreateCouchInstance(couchDBConfig, &disabled.Provider{})
	if err != nil {
		panic(err.Error())
	}

	code := m.Run()

	destroy()

	os.Exit(code)
}

func TestMultipleBlockStores(t *testing.T) {
	env := newTestEnv(t)
	defer env.Cleanup()

	provider := env.provider
	store1, _ := provider.Open("ledger1")
	defer store1.Shutdown()

	store2, _ := provider.Open("ledger2")
	defer store2.Shutdown()

	blocks1 := testutil.ConstructTestBlocks(t, 5)
	for _, b := range blocks1 {
		store1.AddBlock(b)
	}

	blocks2 := testutil.ConstructTestBlocks(t, 10)
	for _, b := range blocks2 {
		store2.AddBlock(b)
	}

	checkBlocks(t, blocks1, store1)
	checkBlocks(t, blocks2, store2)

	checkWithWrongInputs(t, store1, 5)
	checkWithWrongInputs(t, store2, 10)
}

func checkBlocks(t *testing.T, expectedBlocks []*common.Block, store api.BlockStore) {
	bcInfo, _ := store.GetBlockchainInfo()
	assert.Equal(t, uint64(len(expectedBlocks)), bcInfo.Height)
	assert.Equal(t, protoutil.BlockHeaderHash(expectedBlocks[len(expectedBlocks)-1].GetHeader()), bcInfo.CurrentBlockHash)

	itr, _ := store.RetrieveBlocks(0)
	for i := 0; i < len(expectedBlocks); i++ {
		block, _ := itr.Next()
		assert.True(t, proto.Equal(expectedBlocks[i], block.(*common.Block)))
	}

	for blockNum := 0; blockNum < len(expectedBlocks); blockNum++ {
		block := expectedBlocks[blockNum]
		flags := txflags.ValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
		retrievedBlock, _ := store.RetrieveBlockByNumber(uint64(blockNum))
		assert.True(t, proto.Equal(block, retrievedBlock))

		retrievedBlock, _ = store.RetrieveBlockByHash(protoutil.BlockHeaderHash(block.Header))
		assert.True(t, proto.Equal(block, retrievedBlock))

		for txNum := 0; txNum < len(block.Data.Data); txNum++ {
			txEnvBytes := block.Data.Data[txNum]
			txEnv, _ := protoutil.GetEnvelopeFromBlock(txEnvBytes)
			txid, err := extractTxID(txEnvBytes)
			assert.NoError(t, err)

			retrievedBlock, _ := store.RetrieveBlockByTxID(txid)
			assert.True(t, proto.Equal(block, retrievedBlock))

			retrievedTxEnv, _ := store.RetrieveTxByID(txid)
			assert.Equal(t, txEnv, retrievedTxEnv)

			retrievedTxEnv, _ = store.RetrieveTxByBlockNumTranNum(uint64(blockNum), uint64(txNum))
			assert.Equal(t, txEnv, retrievedTxEnv)

			retrievedTxValCode, err := store.RetrieveTxValidationCodeByTxID(txid)
			assert.NoError(t, err)
			assert.Equal(t, flags.Flag(txNum), retrievedTxValCode)
		}
	}
}

func checkWithWrongInputs(t *testing.T, store api.BlockStore, numBlocks int) {
	block, err := store.RetrieveBlockByHash([]byte("non-existent-hash"))
	assert.Nil(t, block)
	assert.Equal(t, blkstorage.ErrNotFoundInIndex, err)

	block, err = store.RetrieveBlockByTxID("non-existent-txid")
	assert.Nil(t, block)
	assert.Equal(t, blkstorage.ErrNotFoundInIndex, err)

	tx, err := store.RetrieveTxByID("non-existent-txid")
	assert.Nil(t, tx)
	assert.Equal(t, blkstorage.ErrNotFoundInIndex, err)

	tx, err = store.RetrieveTxByBlockNumTranNum(uint64(numBlocks+1), uint64(0))
	assert.Nil(t, tx)
	assert.Equal(t, blkstorage.ErrNotFoundInIndex, err)

	txCode, err := store.RetrieveTxValidationCodeByTxID("non-existent-txid")
	assert.Equal(t, peer.TxValidationCode(-1), txCode)
	assert.Equal(t, blkstorage.ErrNotFoundInIndex, err)
}

func TestBlockStoreProvider(t *testing.T) {
	env := newTestEnv(t)
	defer env.Cleanup()
	const numStores = 10

	provider := env.provider
	var stores []api.BlockStore
	var allLedgerIDs []string
	for i := 0; i < numStores; i++ {
		allLedgerIDs = append(allLedgerIDs, constructLedgerid(i))
	}

	for _, id := range allLedgerIDs {
		store, _ := provider.Open(id)
		defer store.Shutdown()
		stores = append(stores, store)
	}
}

func TestBlockStoreAsCommitter(t *testing.T) {

	const ledgerID = "ledger-committer-test"

	//make sure store doesn't exists in couch db
	assert.False(t, couchdbExists(cdbInstance, couchdb.ConstructBlockchainDBName(ledgerID, blockStoreName)))
	assert.False(t, couchdbExists(cdbInstance, couchdb.ConstructBlockchainDBName(ledgerID, txnStoreName)))

	//make sure roles is committer not endorser
	if roles.IsEndorser() {
		rolesValue := make(map[roles.Role]struct{})
		rolesValue[roles.CommitterRole] = struct{}{}
		roles.SetRoles(rolesValue)
		defer func() { roles.SetRoles(nil) }()
	}

	//make sure role is 'committer' only
	assert.True(t, roles.IsCommitter())
	assert.False(t, roles.IsEndorser())

	env := newTestEnv(t)
	defer env.Cleanup()

	//create block store as committer
	provider := env.provider
	store, err := provider.Open(ledgerID)
	assert.NoError(t, err)
	defer store.Shutdown()

	//committer creates store in db, if it doesn't exists
	assert.True(t, couchdbExists(cdbInstance, couchdb.ConstructBlockchainDBName(ledgerID, blockStoreName)))
	assert.True(t, couchdbExists(cdbInstance, couchdb.ConstructBlockchainDBName(ledgerID, txnStoreName)))

	blocks := testutil.ConstructTestBlocks(t, 5)
	for _, b := range blocks {
		store.AddBlock(b)
	}

	//test block store
	checkBlocks(t, blocks, store)
	checkWithWrongInputs(t, store, 5)
}

func TestReadCheckpointInfo(t *testing.T) {
	errExpected := errors.New("injected DB error")

	doc := &couchdb.CouchDoc{}

	db := &mocks.CouchDB{}
	db.ReadDocReturnsOnCall(0, nil, "", errExpected)
	db.ReadDocReturnsOnCall(1, nil, "", errExpected)
	db.ReadDocReturnsOnCall(2, nil, "", errExpected)
	db.ReadDocReturnsOnCall(3, doc, "", nil)

	cp := &checkpoint{
		maxRetries: 3,
		db:         db,
	}

	d, err := cp.readCheckpointInfo()
	require.NoError(t, err)
	require.Equal(t, d, doc)
}

func TestBlockStoreAsEndorser(t *testing.T) {

	const ledgerID = "ledger-endorser-test"

	//make sure store doesn't exists in couch db
	assert.False(t, couchdbExists(cdbInstance, couchdb.ConstructBlockchainDBName(ledgerID, blockStoreName)))
	assert.False(t, couchdbExists(cdbInstance, couchdb.ConstructBlockchainDBName(ledgerID, txnStoreName)))

	//make sure roles is endorser not committer
	if roles.IsCommitter() {
		rolesValue := make(map[roles.Role]struct{})
		rolesValue[roles.EndorserRole] = struct{}{}
		roles.SetRoles(rolesValue)
		defer func() { roles.SetRoles(nil) }()
	}

	//make sure role is 'endorser' only
	assert.True(t, roles.IsEndorser())
	assert.False(t, roles.IsCommitter())

	env := newTestEnv(t)
	defer env.Cleanup()

	//create block store as endorser
	provider := env.provider
	store, err := provider.Open(ledgerID)
	assert.EqualError(t, err, fmt.Sprintf("DB not found: [%s]", couchdb.ConstructBlockchainDBName(ledgerID, blockStoreName)))
	assert.Nil(t, store)

	//create block store manually
	bdb, err := couchdb.CreateCouchDatabase(cdbInstance, couchdb.ConstructBlockchainDBName(ledgerID, blockStoreName))
	assert.NoError(t, err)

	//expect error for missing db index
	store, err = provider.Open(ledgerID)
	assert.EqualError(t, err, fmt.Sprintf("DB index not found: [%s]", couchdb.ConstructBlockchainDBName(ledgerID, blockStoreName)))
	assert.Nil(t, store)

	//create block store index manually
	err = createBlockStoreIndices(bdb)
	assert.NoError(t, err)

	//expect error for missing txn store
	store, err = provider.Open(ledgerID)
	assert.EqualError(t, err, fmt.Sprintf("DB not found: [%s]", couchdb.ConstructBlockchainDBName(ledgerID, txnStoreName)))
	assert.Nil(t, store)

	//create block store manually
	_, err = couchdb.CreateCouchDatabase(cdbInstance, couchdb.ConstructBlockchainDBName(ledgerID, txnStoreName))
	assert.NoError(t, err)

	//open store should be successful
	store, err = provider.Open(ledgerID)
	assert.NoError(t, err)
	assert.NotNil(t, store)
	defer store.Shutdown()

	//add block shouldn't add any new blocks to store for endorser
	blocks := testutil.ConstructTestBlocks(t, 5)
	for _, b := range blocks {
		store.AddBlock(b)
	}

	itr, err := store.RetrieveBlocks(0)
	assert.NoError(t, err)

	var blksCount int
	for {
		blk, err := itr.Next()
		if err != nil && blk == nil {
			break
		}
		blksCount++
	}
	assert.Empty(t, blksCount)

}

//couchdbExists checks if given dbname exists in couchdb
func couchdbExists(cdbInstance *couchdb.CouchInstance, dbName string) bool {

	//create new store db
	cdb, err := couchdb.NewCouchDatabase(cdbInstance, dbName)
	if err != nil {
		panic(err.Error())
	}

	//check if store db exists
	dbExists, err := cdb.ExistsWithRetry()
	if err != nil {
		panic(err.Error())
	}

	return dbExists
}

func constructLedgerid(id int) string {
	return fmt.Sprintf("ledger_%d", id)
}
