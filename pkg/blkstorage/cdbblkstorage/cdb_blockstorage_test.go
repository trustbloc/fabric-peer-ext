/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cdbblkstorage

import (
	"crypto/sha256"
	"errors"
	"hash"
	"os"
	"path/filepath"
	"testing"

	"github.com/hyperledger/fabric/common/ledger/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/blkstorage/cdbblkstorage/mocks"
)

func TestWrongBlockNumber(t *testing.T) {
	env := newTestEnv(t)
	defer env.Cleanup()

	provider := env.provider
	store, _ := provider.Open("testLedger-1")
	defer store.Shutdown()

	blocks := testutil.ConstructTestBlocks(t, 5)
	for i := 0; i < 3; i++ {
		err := store.AddBlock(blocks[i])
		assert.NoError(t, err)
	}
	err := store.AddBlock(blocks[4])
	assert.Error(t, err, "Error should have been thrown when adding block number 4 while block number 3 is expected")
}

func TestCheckpointBlock(t *testing.T) {
	env := newTestEnv(t)
	defer env.Cleanup()

	provider := env.provider
	store, _ := provider.Open("testLedger-2")
	defer store.Shutdown()

	blocks := testutil.ConstructTestBlocks(t, 1)
	err := store.CheckpointBlock(blocks[0])
	assert.NoError(t, err)
	assert.Equal(t, blocks[0].Header.Number, store.(*cdbBlockStore).cpInfo.lastBlockNumber)

}

func TestCheckpointBlockFailure(t *testing.T) {
	env := newTestEnv(t)
	defer env.Cleanup()

	provider := env.provider
	store, _ := provider.Open("testLedger-2")
	defer store.Shutdown()

	blocks := testutil.ConstructTestBlocks(t, 1)

	db := &mocks.CouchDB{}
	db.SaveDocReturns("", errors.New("injected DB error"))

	store.(*cdbBlockStore).cp.db = db
	err := store.CheckpointBlock(blocks[0])
	require.Error(t, err, "adding cpInfo to couchDB failed")
}

func TestCdbBlockStore_ExportTxIds(t *testing.T) {
	env := newTestEnv(t)
	defer env.Cleanup()

	provider := env.provider
	store, _ := provider.Open("testLedger-3")
	defer store.Shutdown()

	store.(*cdbBlockStore).cp.db = &mocks.CouchDB{}
	dir := os.TempDir()

	t.Run("no transactions", func(t *testing.T) {
		defer func() {
			os.RemoveAll(filepath.Join(dir, snapshotDataFileName))
			os.RemoveAll(filepath.Join(dir, snapshotMetadataFileName))
		}()

		result, err := store.ExportTxIds(dir, func() (hash.Hash, error) { return sha256.New(), nil })
		require.NoError(t, err)
		require.Empty(t, result[snapshotDataFileName])
		require.Empty(t, result[snapshotMetadataFileName])
	})

	t.Run("with transactions", func(t *testing.T) {
		defer func() {
			os.RemoveAll(filepath.Join(dir, snapshotDataFileName))
			os.RemoveAll(filepath.Join(dir, snapshotMetadataFileName))
		}()

		for _, block := range testutil.ConstructTestBlocks(t, 10) {
			require.NoError(t, store.AddBlock(block))
		}
		result, err := store.ExportTxIds(dir, func() (hash.Hash, error) { return sha256.New(), nil })
		require.NoError(t, err)
		require.NotEmpty(t, result[snapshotDataFileName])
		require.NotEmpty(t, result[snapshotMetadataFileName])
	})
}
