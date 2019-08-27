/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

// StateRetriever retrieves ledger state
type StateRetriever interface {
	GetState(namespace, key string) ([]byte, error)
	Done()
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
