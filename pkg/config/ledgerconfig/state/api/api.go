/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	"github.com/hyperledger/fabric/protos/ledger/queryresult"
)

const (
	// ConfigCCEventName is the name of the chaincode event that is published to
	// indicate that the configuration has changed
	ConfigCCEventName = "config-update"
)

// StateRetriever retrieves ledger state
type StateRetriever interface {
	GetState(namespace, key string) ([]byte, error)
	GetStateByPartialCompositeKey(namespace, objectType string, attributes []string) (ResultsIterator, error)
	Done()
}

// ResultsIterator iterates through the results of a range query
type ResultsIterator interface {
	Next() (*queryresult.KV, error)
	HasNext() bool
	Close() error
}

// RetrieverProvider returns a State Retriever
type RetrieverProvider interface {
	GetStateRetriever() (StateRetriever, error)
}

// StateStore extends the StateRetriever and adds functions to save ledger state
type StateStore interface {
	StateRetriever
	PutState(namespace, key string, value []byte) error
}

// StoreProvider returns a State Store
type StoreProvider interface {
	RetrieverProvider
	GetStore() (StateStore, error)
}
