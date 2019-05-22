/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"fmt"

	commonledger "github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/core/ledger"
)

// QueryExecutor is a mock query executor
type QueryExecutor struct {
	State map[string]map[string][]byte
	Error error
}

// NewQueryExecutor returns a new mock query executor
func NewQueryExecutor(state map[string]map[string][]byte) *QueryExecutor {
	return &QueryExecutor{
		State: state,
	}
}

// WithError injects an error to the mock executor
func (m *QueryExecutor) WithError(err error) *QueryExecutor {
	m.Error = err
	return m
}

// GetState returns the mock state for the given namespace and key
func (m *QueryExecutor) GetState(namespace string, key string) ([]byte, error) {
	if m.Error != nil {
		return nil, m.Error
	}

	ns := m.State[namespace]
	if ns == nil {
		return nil, fmt.Errorf("Could not retrieve namespace %s", namespace)
	}

	return ns[key], nil
}

// GetStateMultipleKeys returns the mock state for the given namespace and keys
func (m *QueryExecutor) GetStateMultipleKeys(namespace string, keys []string) ([][]byte, error) {
	values := make([][]byte, len(keys))
	for i, k := range keys {
		v, err := m.GetState(namespace, k)
		if err != nil {
			return nil, err
		}
		values[i] = v
	}
	return values, nil
}

// GetStateRangeScanIterator is not currently implemented and will panic if called
func (m *QueryExecutor) GetStateRangeScanIterator(namespace string, startKey string, endKey string) (commonledger.ResultsIterator, error) {
	panic("not implemented")
}

// GetStateRangeScanIteratorWithMetadata is not currently implemented and will panic if called
func (m *QueryExecutor) GetStateRangeScanIteratorWithMetadata(namespace string, startKey, endKey string, metadata map[string]interface{}) (ledger.QueryResultsIterator, error) {
	panic("not implemented")
}

// ExecuteQuery is not currently implemented and will panic if called
func (m *QueryExecutor) ExecuteQuery(namespace, query string) (commonledger.ResultsIterator, error) {
	panic("not implemented")
}

// ExecuteQueryWithMetadata is not currently implemented and will panic if called
func (m *QueryExecutor) ExecuteQueryWithMetadata(namespace, query string, metadata map[string]interface{}) (ledger.QueryResultsIterator, error) {
	panic("not implemented")
}

// GetPrivateData returns the private data for the given namespace, collection, and key
func (m *QueryExecutor) GetPrivateData(namespace, collection, key string) ([]byte, error) {
	return m.GetState(namespace+"$"+collection, key)
}

// GetPrivateDataHash is not currently implemented and will panic if called
func (m *QueryExecutor) GetPrivateDataHash(namespace, collection, key string) ([]byte, error) {
	panic("not implemented")
}

// GetPrivateDataMetadataByHash is not currently implemented and will panic if called
func (m *QueryExecutor) GetPrivateDataMetadataByHash(namespace, collection string, keyhash []byte) (map[string][]byte, error) {
	panic("not implemented")
}

// GetPrivateDataMultipleKeys returns the private data for the given namespace, collection, and keys
func (m *QueryExecutor) GetPrivateDataMultipleKeys(namespace, collection string, keys []string) ([][]byte, error) {
	return m.GetStateMultipleKeys(namespace+"$"+collection, keys)
}

// GetPrivateDataRangeScanIterator is not currently implemented and will panic if called
func (m *QueryExecutor) GetPrivateDataRangeScanIterator(namespace, collection, startKey, endKey string) (commonledger.ResultsIterator, error) {
	panic("not implemented")
}

// ExecuteQueryOnPrivateData is not currently implemented and will panic if called
func (m *QueryExecutor) ExecuteQueryOnPrivateData(namespace, collection, query string) (commonledger.ResultsIterator, error) {
	panic("not implemented")
}

// Done does nothing
func (m *QueryExecutor) Done() {
}

// GetStateMetadata is not currently implemented and will panic if called
func (m *QueryExecutor) GetStateMetadata(namespace, key string) (map[string][]byte, error) {
	panic("not implemented")
}

// GetPrivateDataMetadata is not currently implemented and will panic if called
func (m *QueryExecutor) GetPrivateDataMetadata(namespace, collection, key string) (map[string][]byte, error) {
	panic("not implemented")
}
