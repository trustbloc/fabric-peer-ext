/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package store

import (
	"context"
	"time"

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

// Store manages the storage of private data collections.
type Store interface {
	// Persist stores the private write set of a transaction.
	Persist(txid string, privateSimulationResultsWithConfig *proto.TxPvtReadWriteSetWithConfigInfo) error

	// GetTransientData gets the value for the given transient data item
	GetTransientData(key *Key) (*ExpiringValue, error)

	// GetTransientDataMultipleKeys gets the values for the multiple transient data items in a single call
	GetTransientDataMultipleKeys(key *MultiKey) (ExpiringValues, error)

	// Close closes the store
	Close()
}

// Retriever retrieves private data
type Retriever interface {
	// GetTransientData gets the value for the given transient data item
	GetTransientData(ctxt context.Context, key *Key) (*ExpiringValue, error)

	// GetTransientDataMultipleKeys gets the values for the multiple transient data items in a single call
	GetTransientDataMultipleKeys(ctxt context.Context, key *MultiKey) (ExpiringValues, error)
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
