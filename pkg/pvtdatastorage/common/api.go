/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	xstorageapi "github.com/hyperledger/fabric/extensions/storage/api"
)

// StoreExt extends PrivateDataStore with additional functions
type StoreExt interface {
	xstorageapi.PrivateDataStore
	UpdateLastCommittedBlockNum(blockNum uint64)
}

// Provider manages a private data store
type Provider interface {
	OpenStore(ledgerID string) (StoreExt, error)
	Close()
}

// CacheProvider manages a private data cache
type CacheProvider interface {
	Create(ledgerID string, lastCommittedBlockNum uint64) StoreExt
}
