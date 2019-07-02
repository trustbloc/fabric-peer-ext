/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	cb "github.com/hyperledger/fabric/protos/common"
	proto "github.com/hyperledger/fabric/protos/transientstore"
)

// DataStore implements a mock data store
type DataStore struct {
	transientData map[storeapi.Key]*storeapi.ExpiringValue
	olData        map[storeapi.Key]*storeapi.ExpiringValue
	err           error
}

// NewDataStore returns a mock transient data store
func NewDataStore() *DataStore {
	return &DataStore{
		transientData: make(map[storeapi.Key]*storeapi.ExpiringValue),
		olData:        make(map[storeapi.Key]*storeapi.ExpiringValue),
	}
}

// TransientData sets the transient data for the given key
func (m *DataStore) TransientData(key *storeapi.Key, value *storeapi.ExpiringValue) *DataStore {
	m.transientData[storeapi.Key{Namespace: key.Namespace, Collection: key.Collection, Key: key.Key}] = value
	return m
}

// Data sets the data for the given key
func (m *DataStore) Data(key *storeapi.Key, value *storeapi.ExpiringValue) *DataStore {
	m.olData[storeapi.Key{Namespace: key.Namespace, Collection: key.Collection, Key: key.Key}] = value
	return m
}

// Error sets an err
func (m *DataStore) Error(err error) *DataStore {
	m.err = err
	return m
}

// Persist stores the private write set of a transaction along with the collection config
// in the transient store based on txid and the block height the private data was received at
func (m *DataStore) Persist(txid string, privateSimulationResultsWithConfig *proto.TxPvtReadWriteSetWithConfigInfo) error {
	return m.err
}

// GetTransientData gets the value for the given transient data item
func (m *DataStore) GetTransientData(key *storeapi.Key) (*storeapi.ExpiringValue, error) {
	return m.transientData[storeapi.Key{Namespace: key.Namespace, Collection: key.Collection, Key: key.Key}], m.err
}

// GetTransientDataMultipleKeys gets the values for the multiple transient data items in a single call
func (m *DataStore) GetTransientDataMultipleKeys(key *storeapi.MultiKey) (storeapi.ExpiringValues, error) {
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

// PutData stores the key/value
func (m *DataStore) PutData(config *cb.StaticCollectionConfig, key *storeapi.Key, value *storeapi.ExpiringValue) error {
	if m.err != nil {
		return m.err
	}
	m.olData[storeapi.Key{Namespace: key.Namespace, Collection: key.Collection, Key: key.Key}] = value
	return nil
}

// GetData gets the value for the given DCAS item
func (m *DataStore) GetData(key *storeapi.Key) (*storeapi.ExpiringValue, error) {
	return m.olData[storeapi.Key{Namespace: key.Namespace, Collection: key.Collection, Key: key.Key}], m.err
}

// GetDataMultipleKeys gets the values for the multiple DCAS items in a single call
func (m *DataStore) GetDataMultipleKeys(key *storeapi.MultiKey) (storeapi.ExpiringValues, error) {
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

// Query executes the given query
func (m *DataStore) Query(key *storeapi.QueryKey) (storeapi.ResultsIterator, error) {
	panic("not implemented")
}

// Close closes the store
func (m *DataStore) Close() {
}
