/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package store

import (
	"context"
	"time"

	cb "github.com/hyperledger/fabric/protos/common"
	proto "github.com/hyperledger/fabric/protos/transientstore"
	"github.com/trustbloc/fabric-peer-ext/pkg/common"
)

// ExpiringValue is holds the value and expiration time.
type ExpiringValue struct {
	Value  []byte
	Expiry time.Time
}

// ExpiringValues expiring values
type ExpiringValues []*ExpiringValue

// QueryResult holds a single item from the query result set
type QueryResult struct {
	*Key
	*ExpiringValue
}

// ResultsIterator is an iterator returned by a query
type ResultsIterator interface {
	// Next returns the next item in the result set. The result is expected to be nil when
	// the iterator gets exhausted
	Next() (*QueryResult, error)

	// Close releases resources occupied by the iterator
	Close()
}

// Store manages the storage of private data collections.
type Store interface {
	// Persist stores the private write set of a transaction.
	Persist(txid string, privateSimulationResultsWithConfig *proto.TxPvtReadWriteSetWithConfigInfo) error

	// GetTransientData gets the value for the given transient data item
	GetTransientData(key *Key) (*ExpiringValue, error)

	// GetTransientDataMultipleKeys gets the values for the multiple transient data items in a single call
	GetTransientDataMultipleKeys(key *MultiKey) (ExpiringValues, error)

	// GetData gets the value for the given item
	GetData(key *Key) (*ExpiringValue, error)

	// GetDataMultipleKeys gets the values for the multiple items in a single call
	GetDataMultipleKeys(key *MultiKey) (ExpiringValues, error)

	// Query executes the given query
	// NOTE: This function is only supported on CouchDB
	Query(key *QueryKey) (ResultsIterator, error)

	// PutData stores the key/value.
	PutData(config *cb.StaticCollectionConfig, key *Key, value *ExpiringValue) error

	// Close closes the store
	Close()
}

// Retriever retrieves private data
type Retriever interface {
	// GetTransientData gets the value for the given transient data item
	GetTransientData(ctxt context.Context, key *Key) (*ExpiringValue, error)

	// GetTransientDataMultipleKeys gets the values for the multiple transient data items in a single call
	GetTransientDataMultipleKeys(ctxt context.Context, key *MultiKey) (ExpiringValues, error)

	// GetData gets the value for the given data item
	GetData(ctxt context.Context, key *Key) (*ExpiringValue, error)

	// GetDataMultipleKeys gets the values for the multiple data items in a single call
	GetDataMultipleKeys(ctxt context.Context, key *MultiKey) (ExpiringValues, error)

	// Query returns the results of the given query
	Query(ctxt context.Context, key *QueryKey) (ResultsIterator, error)
}

// Provider provides private data retrievers
type Provider interface {
	RetrieverForChannel(channel string) Retriever
}

// Values returns the ExpiringValues as Values
func (ev ExpiringValues) Values() common.Values {
	vals := make(common.Values, len(ev))
	for i, v := range ev {
		vals[i] = v
	}
	return vals
}
