/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	"context"

	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	proto "github.com/hyperledger/fabric/protos/transientstore"
)

// Store manages the storage of transient data.
type Store interface {
	// Persist stores the private write set of a transaction.
	Persist(txID string, privateSimulationResultsWithConfig *proto.TxPvtReadWriteSetWithConfigInfo) error

	// GetTransientData gets the value for the given transient data item
	GetTransientData(key *storeapi.Key) (*storeapi.ExpiringValue, error)

	// GetTransientDataMultipleKeys gets the values for the multiple transient data items in a single call
	GetTransientDataMultipleKeys(key *storeapi.MultiKey) (storeapi.ExpiringValues, error)

	// Close closes the store
	Close()
}

// StoreProvider is an interface to open/close a provider
type StoreProvider interface {
	// OpenStore creates a handle to the transient data store for the given ledger ID
	OpenStore(ledgerid string) (Store, error)

	// Close cleans up the provider
	Close()
}

// Retriever retrieves transient data
type Retriever interface {
	// GetTransientData gets the value for the given transient data item
	GetTransientData(ctxt context.Context, key *storeapi.Key) (*storeapi.ExpiringValue, error)

	// GetTransientDataMultipleKeys gets the values for the multiple transient data items in a single call
	GetTransientDataMultipleKeys(ctxt context.Context, key *storeapi.MultiKey) (storeapi.ExpiringValues, error)
}

// Provider provides transient data retrievers
type Provider interface {
	RetrieverForChannel(channel string) Retriever
}
