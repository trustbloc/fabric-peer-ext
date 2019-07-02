/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	"context"

	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	cb "github.com/hyperledger/fabric/protos/common"
	proto "github.com/hyperledger/fabric/protos/transientstore"
)

// Store manages the storage of private data collections.
type Store interface {
	// Persist stores the private write set of a transaction.
	Persist(txid string, privateSimulationResultsWithConfig *proto.TxPvtReadWriteSetWithConfigInfo) error

	// PutData stores the key/value.
	PutData(config *cb.StaticCollectionConfig, key *storeapi.Key, value *storeapi.ExpiringValue) error

	// GetData gets the value for the given item
	GetData(key *storeapi.Key) (*storeapi.ExpiringValue, error)

	// GetDataMultipleKeys gets the values for the multiple items in a single call
	GetDataMultipleKeys(key *storeapi.MultiKey) (storeapi.ExpiringValues, error)

	// Query executes the given query
	// NOTE: This function is only supported on CouchDB
	Query(key *storeapi.QueryKey) (storeapi.ResultsIterator, error)

	// Close closes the store
	Close()
}

// StoreProvider is an interface to open/close a store
type StoreProvider interface {
	// OpenStore creates a handle to the private data store for the given ledger ID
	OpenStore(ledgerid string) (Store, error)

	// Close cleans up the provider
	Close()
}

// Retriever retrieves data
type Retriever interface {
	// GetData gets the value for the given data item
	GetData(ctxt context.Context, key *storeapi.Key) (*storeapi.ExpiringValue, error)

	// GetDataMultipleKeys gets the values for the multiple data items in a single call
	GetDataMultipleKeys(ctxt context.Context, key *storeapi.MultiKey) (storeapi.ExpiringValues, error)

	// Query returns the results from the given query
	// NOTE: This function is only supported on CouchDB
	Query(ctxt context.Context, key *storeapi.QueryKey) (storeapi.ResultsIterator, error)
}

// Provider provides data retrievers
type Provider interface {
	RetrieverForChannel(channel string) Retriever
}
