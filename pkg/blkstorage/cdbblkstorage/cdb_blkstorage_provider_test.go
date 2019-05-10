/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cdbblkstorage

import (
	"fmt"
	"os"
	"testing"

	"github.com/golang/protobuf/proto"

	"github.com/hyperledger/fabric/common/ledger/blkstorage"
	"github.com/hyperledger/fabric/common/ledger/testutil"
	"github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/stretchr/testify/assert"
	xtestutil "github.com/trustbloc/fabric-peer-ext/pkg/testutil"
)

func TestMain(m *testing.M) {

	//setup extension test environment
	_, _, destroy := xtestutil.SetupExtTestEnv()

	code := m.Run()

	destroy()

	os.Exit(code)
}

func TestMultipleBlockStores(t *testing.T) {
	env := newTestEnv(t)
	defer env.Cleanup()

	provider := env.provider
	store1, _ := provider.OpenBlockStore("ledger1")
	defer store1.Shutdown()

	store2, _ := provider.CreateBlockStore("ledger2")
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

func checkBlocks(t *testing.T, expectedBlocks []*common.Block, store blkstorage.BlockStore) {
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
		flags := util.TxValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
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

func checkWithWrongInputs(t *testing.T, store blkstorage.BlockStore, numBlocks int) {
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
	var stores []blkstorage.BlockStore
	var allLedgerIDs []string
	for i := 0; i < numStores; i++ {
		allLedgerIDs = append(allLedgerIDs, constructLedgerid(i))
	}

	for _, id := range allLedgerIDs {
		store, _ := provider.OpenBlockStore(id)
		defer store.Shutdown()
		stores = append(stores, store)
	}

	for _, id := range allLedgerIDs {
		exists, err := provider.Exists(id)
		assert.NoError(t, err)
		assert.Equal(t, true, exists)
	}

	exists, err := provider.Exists(constructLedgerid(numStores + 1))
	assert.NoError(t, err)
	assert.Equal(t, false, exists)

}

func constructLedgerid(id int) string {
	return fmt.Sprintf("ledger_%d", id)
}
