package cdbblkstorage

import (
	"testing"

	"github.com/hyperledger/fabric/common/ledger/testutil"
	"github.com/stretchr/testify/assert"
)

func TestWrongBlockNumber(t *testing.T) {
	env := newTestEnv(t)
	defer env.Cleanup()

	provider := env.provider
	store, _ := provider.OpenBlockStore("testLedger-1")
	defer store.Shutdown()

	blocks := testutil.ConstructTestBlocks(t, 5)
	for i := 0; i < 3; i++ {
		err := store.AddBlock(blocks[i])
		assert.NoError(t, err)
	}
	err := store.AddBlock(blocks[4])
	assert.Error(t, err, "Error should have been thrown when adding block number 4 while block number 3 is expected")
}
