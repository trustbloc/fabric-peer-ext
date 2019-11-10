/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package support

import (
	"sync"

	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
)

// This file is temporarily added. It should be removed once all resources
// have been converted to use dependency injection.

type collectionConfigRetrieverCache struct {
	mutex      sync.RWMutex
	retrievers map[string]*CollectionConfigRetriever
}

var collConfigRetrieverCache = &collectionConfigRetrieverCache{
	retrievers: make(map[string]*CollectionConfigRetriever),
}

// CollectionConfigRetrieverForChannel returns the collection config retriever for the given channel
func CollectionConfigRetrieverForChannel(channelID string) *CollectionConfigRetriever {
	return collConfigRetrieverCache.forChannel(channelID)
}

// InitCollectionConfigRetriever initializes a collection config retriever for the given channel
func InitCollectionConfigRetriever(channelID string, ledger peerLedger, blockPublisher blockPublisher) {
	collConfigRetrieverCache.init(channelID, ledger, blockPublisher)
}

func (rc *collectionConfigRetrieverCache) init(channelID string, ledger peerLedger, blockPublisher blockPublisher) {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()

	rc.retrievers[channelID] = newCollectionConfigRetriever(channelID, ledger, blockPublisher, mspmgmt.GetIdentityDeserializer(channelID))
}

func (rc *collectionConfigRetrieverCache) forChannel(channelID string) *CollectionConfigRetriever {
	rc.mutex.RLock()
	defer rc.mutex.RUnlock()
	return rc.retrievers[channelID]
}
