/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cdbblkstorage

import (
	"sync"

	"github.com/hyperledger/fabric/common/ledger"
)

// blocksItr - an iterator for iterating over a sequence of blocks
type blocksItr struct {
	cdbBlockStore        *cdbBlockStore
	maxBlockNumAvailable uint64
	blockNumToRetrieve   uint64
	closeMarker          bool
	closeMarkerLock      *sync.Mutex
}

func newBlockItr(cdbBlockStore *cdbBlockStore, startBlockNum uint64) *blocksItr {
	return &blocksItr{cdbBlockStore, cdbBlockStore.cpInfo.lastBlockNumber, startBlockNum, false, &sync.Mutex{}}
}

func (itr *blocksItr) waitForBlock(blockNum uint64) uint64 {
	itr.cdbBlockStore.cpInfoCond.L.Lock()
	defer itr.cdbBlockStore.cpInfoCond.L.Unlock()
	for itr.cdbBlockStore.cpInfo.lastBlockNumber < blockNum && !itr.shouldClose() {
		logger.Debugf("Going to wait for newer blocks. maxAvailaBlockNumber=[%d], waitForBlockNum=[%d]",
			itr.cdbBlockStore.cpInfo.lastBlockNumber, blockNum)
		itr.cdbBlockStore.cpInfoCond.Wait()
		logger.Debugf("Came out of wait. maxAvailaBlockNumber=[%d]", itr.cdbBlockStore.cpInfo.lastBlockNumber)
	}
	return itr.cdbBlockStore.cpInfo.lastBlockNumber
}

func (itr *blocksItr) shouldClose() bool {
	itr.closeMarkerLock.Lock()
	defer itr.closeMarkerLock.Unlock()
	return itr.closeMarker
}

// Next moves the cursor to next block and returns true iff the iterator is not exhausted
func (itr *blocksItr) Next() (ledger.QueryResult, error) {
	if itr.maxBlockNumAvailable < itr.blockNumToRetrieve {
		itr.maxBlockNumAvailable = itr.waitForBlock(itr.blockNumToRetrieve)
	}
	itr.closeMarkerLock.Lock()
	defer itr.closeMarkerLock.Unlock()
	if itr.closeMarker {
		return nil, nil
	}

	nextBlock, err := itr.cdbBlockStore.RetrieveBlockByNumber(itr.blockNumToRetrieve)
	if err != nil {
		return nil, err
	}
	itr.blockNumToRetrieve++
	return nextBlock, nil
}

// Close releases any resources held by the iterator
func (itr *blocksItr) Close() {
	itr.cdbBlockStore.cpInfoCond.L.Lock()
	defer itr.cdbBlockStore.cpInfoCond.L.Unlock()
	itr.closeMarkerLock.Lock()
	defer itr.closeMarkerLock.Unlock()
	itr.closeMarker = true
	itr.cdbBlockStore.cpInfoCond.Broadcast()
}
