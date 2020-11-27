/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"sync"

	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
)

// MockStateDB implements a mocked state database
type MockStateDB struct {
	state map[string]map[string][]byte
	err   error
	mutex sync.RWMutex
}

// NewStateDB return a new mocked state database
func NewStateDB() *MockStateDB {
	return &MockStateDB{
		state: make(map[string]map[string][]byte),
	}
}

// WithState sets the state for the given namespace and key
func (db *MockStateDB) WithState(namespace, key string, value []byte) *MockStateDB {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	ns, ok := db.state[namespace]
	if !ok {
		ns = make(map[string][]byte)
		db.state[namespace] = ns
	}

	ns[key] = value

	return db
}

// WithError injects an error
func (db *MockStateDB) WithError(err error) *MockStateDB {
	db.err = err

	return db
}

// GetState gets the value for given namespace and key. For a chaincode, the namespace corresponds to the chaincodeId
func (db *MockStateDB) GetState(namespace string, key string) ([]byte, error) {
	if db.err != nil {
		return nil, db.err
	}

	return db.state[namespace][key], nil
}

// GetStateMultipleKeys gets the values for multiple keys in a single call
func (db *MockStateDB) GetStateMultipleKeys(namespace string, keys []string) ([][]byte, error) {
	if db.err != nil {
		return nil, db.err
	}

	values := make([][]byte, len(keys))
	for i, key := range keys {
		var err error
		values[i], err = db.GetState(namespace, key)
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}

// GetStateRangeScanIterator returns an iterator that contains all the key-values between given key ranges.
func (db *MockStateDB) GetStateRangeScanIterator(namespace string, startKey string, endKey string) (statedb.ResultsIterator, error) {
	panic("not implemented")
}

// GetStateRangeScanIteratorWithPagination returns an iterator that contains all the key-values between given key ranges.
func (db *MockStateDB) GetStateRangeScanIteratorWithPagination(namespace string, startKey string, endKey string, pageSize int32) (statedb.QueryResultsIterator, error) {
	panic("not implemented")
}

// ExecuteQuery executes the given query and returns an iterator that contains results of type *VersionedKV.
func (db *MockStateDB) ExecuteQuery(namespace, query string) (statedb.ResultsIterator, error) {
	panic("not implemented")
}

// ExecuteQueryWithPagination executes the given query and
func (db *MockStateDB) ExecuteQueryWithPagination(namespace, query, bookmark string, pageSize int32) (statedb.QueryResultsIterator, error) {
	panic("not implemented")
}

// BytesKeySupported returns true if the implementation (underlying db) supports the any bytes to be used as key.
func (db *MockStateDB) BytesKeySupported() bool {
	return false
}

// UpdateCache is not implemented
func (db *MockStateDB) UpdateCache(uint64, []byte) error {
	panic("not implemented")
}
