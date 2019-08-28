/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/state/api"
)

// StoreProvider is a mock StoreProvider
type StoreProvider struct {
	store api.StateStore
	err   error
}

// NewStoreProvider returns a mock StoreProvider
func NewStoreProvider() *StoreProvider {
	return &StoreProvider{
		store: NewStateStore(),
	}
}

// WithStore sets the StateStore
func (m *StoreProvider) WithStore(store api.StateStore) *StoreProvider {
	m.store = store
	return m
}

// WithError injects the store provider with an error
func (m *StoreProvider) WithError(err error) *StoreProvider {
	m.err = err
	return m
}

// GetStateRetriever returns a mock StateRetriever
func (m *StoreProvider) GetStateRetriever() (api.StateRetriever, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.store, nil
}

// GetStore returns a mock StateStore
func (m *StoreProvider) GetStore() (api.StateStore, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.store, nil
}

// StateStore is a mock StateStore
type StateStore struct {
	*StateRetriever
	err error
}

// NewStateStore returns a mock StateStore
func NewStateStore() *StateStore {
	return &StateStore{
		StateRetriever: NewStateRetriever(),
	}
}

// WithStoreError injects the state store with an error
func (m *StateStore) WithStoreError(err error) *StateStore {
	m.err = err
	return m
}

// WithRetrieverError injects the state retriever with an error
func (m *StateStore) WithRetrieverError(err error) *StateStore {
	m.StateRetriever.WithError(err)
	return m
}

// PutState saves the given state
func (m *StateStore) PutState(namespace, key string, value []byte) error {
	if m.err != nil {
		return m.err
	}
	m.WithState(namespace, key, value)
	return nil
}
