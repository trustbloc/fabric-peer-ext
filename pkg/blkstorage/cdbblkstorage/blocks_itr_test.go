/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cdbblkstorage

import (
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/common/ledger/testutil"

	"github.com/stretchr/testify/assert"
)

func TestBlocksItr(t *testing.T) {

	env := newTestEnv(t)
	defer env.Cleanup()

	const numberOfBlks = 5
	const startBlk = 3

	provider := env.provider
	store, _ := provider.Open("ledger-1")
	defer store.Shutdown()
	cdbstore := store.(*cdbBlockStore)

	blocks := testutil.ConstructTestBlocks(t, numberOfBlks)
	testAppendBlocks(cdbstore, blocks)

	//get iterator
	itr := newBlockItr(cdbstore, 0)
	for i := 0; i < numberOfBlks; i++ {
		blk, err := itr.Next()
		assert.NoError(t, err)
		assert.NotNil(t, blk)
		assert.Equal(t, uint64(i), blk.(*common.Block).Header.Number)
	}
	itr.Close()

	itr = newBlockItr(cdbstore, startBlk)
	for i := startBlk; i < numberOfBlks; i++ {
		blk, err := itr.Next()
		assert.NoError(t, err)
		assert.NotNil(t, blk)
		assert.Equal(t, uint64(i), blk.(*common.Block).Header.Number)
	}
	itr.Close()

	blk, err := itr.Next()
	assert.NoError(t, err)
	assert.Nil(t, blk)
}

func TestBlocksItrBlockingNext(t *testing.T) {
	env := newTestEnv(t)
	defer env.Cleanup()

	const numberOfBlks = 10

	provider := env.provider
	store, _ := provider.Open("testLedger")
	defer store.Shutdown()
	cdbstore := store.(*cdbBlockStore)

	blocks := testutil.ConstructTestBlocks(t, numberOfBlks)
	testAppendBlocks(cdbstore, blocks[:5])

	itr := newBlockItr(cdbstore, 1)
	defer itr.Close()

	readyChan := make(chan struct{})
	doneChan := make(chan bool)

	go testIterateAndVerify(t, itr, blocks[1:], 4, readyChan, doneChan)
	<-readyChan
	testAppendBlocks(cdbstore, blocks[5:7])
	time.Sleep(time.Millisecond * 10)
	testAppendBlocks(cdbstore, blocks[7:])
	<-doneChan
}

func TestRaceToDeadlock(t *testing.T) {
	env := newTestEnv(t)
	defer env.Cleanup()

	const numberOfBlks = 5

	provider := env.provider
	store, _ := provider.Open("testLedger")
	defer store.Shutdown()

	blocks := testutil.ConstructTestBlocks(t, numberOfBlks)
	testAppendBlocks(store.(*cdbBlockStore), blocks)

	for i := 0; i < 1000; i++ {
		itr, err := store.RetrieveBlocks(0)
		if err != nil {
			panic(err)
		}
		go func() {
			itr.Next()
		}()
		itr.Close()
	}

	for i := 0; i < 1000; i++ {
		itr, err := store.RetrieveBlocks(0)
		if err != nil {
			panic(err)
		}
		go func() {
			itr.Close()
		}()
		itr.Next()
	}
}

func TestBlockItrCloseWithoutRetrieve(t *testing.T) {
	env := newTestEnv(t)
	defer env.Cleanup()

	const numberOfBlks = 5

	provider := env.provider
	store, _ := provider.Open("testLedger")
	defer store.Shutdown()

	blocks := testutil.ConstructTestBlocks(t, numberOfBlks)
	testAppendBlocks(store.(*cdbBlockStore), blocks)

	itr, err := store.RetrieveBlocks(2)
	assert.NoError(t, err)
	itr.Close()

	assert.True(t, itr.(*blocksItr).closeMarker)
}

func TestCloseMultipleItrsWaitForFutureBlock(t *testing.T) {
	env := newTestEnv(t)
	defer env.Cleanup()

	const numberOfBlks = 10

	provider := env.provider
	store, _ := provider.Open("testLedger-2")
	defer store.Shutdown()

	blocks := testutil.ConstructTestBlocks(t, numberOfBlks)

	testAppendBlocks(store.(*cdbBlockStore), blocks[:5])

	wg := &sync.WaitGroup{}
	wg.Add(2)
	itr1, err := store.RetrieveBlocks(7)
	assert.NoError(t, err)
	// itr1 does not retrieve any block because it closes before new blocks are added
	go iterateInBackground(t, itr1, 9, wg, []uint64{})

	itr2, err := store.RetrieveBlocks(8)
	assert.NoError(t, err)
	// itr2 retrieves two blocks 8 and 9. Because it started waiting for 8 and quits at 9
	go iterateInBackground(t, itr2, 9, wg, []uint64{8, 9})

	// sleep for the background iterators to get started
	time.Sleep(2 * time.Second)
	itr1.Close()
	testAppendBlocks(store.(*cdbBlockStore), blocks[5:])
	wg.Wait()
}

func iterateInBackground(t *testing.T, itr ledger.ResultsIterator, quitAfterBlkNum uint64, wg *sync.WaitGroup, expectedBlockNums []uint64) {
	defer wg.Done()
	retrievedBlkNums := []uint64{}
	defer func() { assert.Equal(t, expectedBlockNums, retrievedBlkNums) }()

	for {
		blk, err := itr.Next()
		assert.NoError(t, err)
		if blk == nil {
			return
		}
		blkNum := blk.(*common.Block).Header.Number
		retrievedBlkNums = append(retrievedBlkNums, blkNum)
		t.Logf("blk.Num=%d", blk.(*common.Block).Header.Number)
		if blkNum == quitAfterBlkNum {
			return
		}
	}
}

func testIterateAndVerify(t *testing.T, itr *blocksItr, blocks []*common.Block, readyAt int, readyChan chan<- struct{}, doneChan chan bool) {
	blocksIterated := 0
	for {
		t.Logf("blocksIterated: %v", blocksIterated)
		block, err := itr.Next()
		assert.NoError(t, err)
		assert.True(t, proto.Equal(blocks[blocksIterated], block.(*common.Block)))
		blocksIterated++
		if blocksIterated == readyAt {
			close(readyChan)
		}
		if blocksIterated == len(blocks) {
			break
		}
	}
	doneChan <- true
}

func testAppendBlocks(store *cdbBlockStore, blocks []*common.Block) {
	for _, b := range blocks {
		store.AddBlock(b)
	}
}
