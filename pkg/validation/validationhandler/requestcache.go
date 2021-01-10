/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validationhandler

import (
	"sync"

	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/trustbloc/fabric-peer-ext/pkg/gossip/appdata"
)

type request struct {
	block     *common.Block
	responder appdata.Responder
}

type requestCache struct {
	channelID string
	requests  []*request
	mutex     sync.RWMutex
}

func newRequestCache(channelID string) *requestCache {
	return &requestCache{
		channelID: channelID,
	}
}

// Add adds the given block to the cache
func (b *requestCache) Add(block *common.Block, responder appdata.Responder) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.requests = append(b.requests, &request{
		block:     block,
		responder: responder,
	})
}

// Remove removes the given block number and also all previous blocks
// Returns the request for the given number or nil if not found
func (b *requestCache) Remove(blockNum uint64) *request {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	var req *request
	removeToIndex := -1
	for i := 0; i < len(b.requests); i++ {
		r := b.requests[i]
		if r.block.Header.Number <= blockNum {
			removeToIndex = i
			if r.block.Header.Number == blockNum {
				req = r
				break
			}
		}
	}

	if removeToIndex >= 0 {
		// Since blocks are added in order, remove all blocks before removeToIndex
		b.requests = b.requests[removeToIndex+1:]
	}

	return req
}

// Size returns the size of the cache
func (b *requestCache) Size() int {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return len(b.requests)
}
