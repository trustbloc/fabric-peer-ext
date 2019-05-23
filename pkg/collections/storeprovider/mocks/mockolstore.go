/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	cb "github.com/hyperledger/fabric/protos/common"
	proto "github.com/hyperledger/fabric/protos/transientstore"
	olapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/api"
)

// StoreProvider implements a mock transient data store provider
type StoreProvider struct {
	store *Store
	err   error
}

// NewOffLedgerStoreProvider returns a new  provider
func NewOffLedgerStoreProvider() *StoreProvider {
	return &StoreProvider{
		store: NewStore(),
	}
}

// Data invokes the Data of the store for the given key and value
func (p *StoreProvider) Data(key *storeapi.Key, value *storeapi.ExpiringValue) *StoreProvider {
	p.store.Data(key, value)
	return p
}

// Error stores the error
func (p *StoreProvider) Error(err error) *StoreProvider {
	p.err = err
	return p
}

// StoreError stores the storeError
func (p *StoreProvider) StoreError(err error) *StoreProvider {
	p.store.Error(err)
	return p
}

// StoreForChannel returns the transient data store for the given channel
func (p *StoreProvider) StoreForChannel(channelID string) olapi.Store {
	return p.store
}

// OpenStore opens the transient data store for the given channel
func (p *StoreProvider) OpenStore(channelID string) (olapi.Store, error) {
	return p.store, p.err
}

// Close closes the  store for the given channel
func (p *StoreProvider) Close() {
	p.store.Close()
}

// IsStoreClosed indicates whether the  store is closed
func (p *StoreProvider) IsStoreClosed() bool {
	return p.store.closed
}

// Store implements a mock store
type Store struct {
	data   map[storeapi.Key]*storeapi.ExpiringValue
	err    error
	closed bool
}

// NewStore returns a mock transient data store
func NewStore() *Store {
	return &Store{
		data: make(map[storeapi.Key]*storeapi.ExpiringValue),
	}
}

// Data sets the data for the given key
func (m *Store) Data(key *storeapi.Key, value *storeapi.ExpiringValue) *Store {
	m.data[storeapi.Key{Namespace: key.Namespace, Collection: key.Collection, Key: key.Key}] = value
	return m
}

// Error sets an err
func (m *Store) Error(err error) *Store {
	m.err = err
	return m
}

// Persist stores the private write set of a transaction along with the collection config
// in the transient store based on txid and the block height the private data was received at
func (m *Store) Persist(txid string, privateSimulationResultsWithConfig *proto.TxPvtReadWriteSetWithConfigInfo) error {
	return m.err
}

// PutData stores the key/value
func (m *Store) PutData(config *cb.StaticCollectionConfig, key *storeapi.Key, value *storeapi.ExpiringValue) error {
	if m.err != nil {
		return m.err
	}
	m.data[storeapi.Key{Namespace: key.Namespace, Collection: key.Collection, Key: key.Key}] = value
	return nil
}

// GetData gets the value for the given item
func (m *Store) GetData(key *storeapi.Key) (*storeapi.ExpiringValue, error) {
	return m.data[storeapi.Key{Namespace: key.Namespace, Collection: key.Collection, Key: key.Key}], m.err
}

// GetDataMultipleKeys gets the values for the multiple items in a single call
func (m *Store) GetDataMultipleKeys(key *storeapi.MultiKey) (storeapi.ExpiringValues, error) {
	var values storeapi.ExpiringValues
	for _, k := range key.Keys {
		value, err := m.GetData(&storeapi.Key{Namespace: key.Namespace, Collection: key.Collection, Key: k})
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, m.err
}

// Close closes the store
func (m *Store) Close() {
	m.closed = true
}
