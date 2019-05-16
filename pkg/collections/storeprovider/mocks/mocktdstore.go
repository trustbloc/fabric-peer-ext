/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	proto "github.com/hyperledger/fabric/protos/transientstore"
	tdapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/api"
)

// TransientDataStoreProvider implements a mock transient data store provider
type TransientDataStoreProvider struct {
	store *TransientDataStore
	err   error
}

// NewTransientDataStoreProvider creates a new provider
func NewTransientDataStoreProvider() *TransientDataStoreProvider {
	return &TransientDataStoreProvider{
		store: NewTransientDataStore(),
	}
}

// Data stores key value
func (p *TransientDataStoreProvider) Data(key *storeapi.Key, value *storeapi.ExpiringValue) *TransientDataStoreProvider {
	p.store.Data(key, value)
	return p
}

// Error stores the error
func (p *TransientDataStoreProvider) Error(err error) *TransientDataStoreProvider {
	p.err = err
	return p
}

// StoreError stores the StoreError
func (p *TransientDataStoreProvider) StoreError(err error) *TransientDataStoreProvider {
	p.store.Error(err)
	return p
}

// StoreForChannel returns the transient data store for the given channel
func (p *TransientDataStoreProvider) StoreForChannel(channelID string) tdapi.Store {
	return p.store
}

// OpenStore opens the transient data store for the given channel
func (p *TransientDataStoreProvider) OpenStore(channelID string) (tdapi.Store, error) {
	return p.store, p.err
}

// Close closes the transient data store for the given channel
func (p *TransientDataStoreProvider) Close() {
	p.store.Close()
}

// IsStoreClosed indicates whether the transient data store is closed
func (p *TransientDataStoreProvider) IsStoreClosed() bool {
	return p.store.closed
}

// TransientDataStore implements a mock transient data store
type TransientDataStore struct {
	transientData map[storeapi.Key]*storeapi.ExpiringValue
	err           error
	closed        bool
}

// NewTransientDataStore returns a mock transient data store
func NewTransientDataStore() *TransientDataStore {
	return &TransientDataStore{
		transientData: make(map[storeapi.Key]*storeapi.ExpiringValue),
	}
}

// Data sets the transient data for the given key
func (m *TransientDataStore) Data(key *storeapi.Key, value *storeapi.ExpiringValue) *TransientDataStore {
	m.transientData[storeapi.Key{Namespace: key.Namespace, Collection: key.Collection, Key: key.Key}] = value
	return m
}

// Error sets an err
func (m *TransientDataStore) Error(err error) *TransientDataStore {
	m.err = err
	return m
}

// Persist stores the private write set of a transaction along with the collection config
// in the transient store based on txid and the block height the private data was received at
func (m *TransientDataStore) Persist(txid string, privateSimulationResultsWithConfig *proto.TxPvtReadWriteSetWithConfigInfo) error {
	return m.err
}

// GetTransientData gets the value for the given transient data item
func (m *TransientDataStore) GetTransientData(key *storeapi.Key) (*storeapi.ExpiringValue, error) {
	return m.transientData[storeapi.Key{Namespace: key.Namespace, Collection: key.Collection, Key: key.Key}], m.err
}

// GetTransientDataMultipleKeys gets the values for the multiple transient data items in a single call
func (m *TransientDataStore) GetTransientDataMultipleKeys(key *storeapi.MultiKey) (storeapi.ExpiringValues, error) {
	var values storeapi.ExpiringValues
	for _, k := range key.Keys {
		value, err := m.GetTransientData(&storeapi.Key{Namespace: key.Namespace, Collection: key.Collection, Key: k})
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, m.err
}

// Close closes the store
func (m *TransientDataStore) Close() {
	m.closed = true
}
