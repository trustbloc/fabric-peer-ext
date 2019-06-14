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
	state        map[string]map[string][]byte
	queryResults map[string][][]byte
	error        error
}

// NewQueryExecutor returns a new mock query executor
func NewQueryExecutor() *QueryExecutor {
	return &QueryExecutor{
		state:        make(map[string]map[string][]byte),
		queryResults: make(map[string][][]byte),
	}
}

// WithState sets the state
func (m *QueryExecutor) WithState(ns, key string, value []byte) *QueryExecutor {
	nsState, ok := m.state[ns]
	if !ok {
		nsState = make(map[string][]byte)
		m.state[ns] = nsState
	}
	nsState[key] = value
	return m
}

// WithPrivateState sets the private state
func (m *QueryExecutor) WithPrivateState(ns, collection, key string, value []byte) *QueryExecutor {
	nskey := privateNamespace(ns, collection)
	nsState, ok := m.state[nskey]
	if !ok {
		nsState = make(map[string][]byte)
		m.state[nskey] = nsState
	}
	nsState[key] = value
	return m
}

// WithQueryResults sets the query results for a given query on a namespace
func (m *QueryExecutor) WithQueryResults(ns, query string, results [][]byte) *QueryExecutor {
	m.queryResults[queryResultsKey(ns, query)] = results
	return m
}

// WithPrivateQueryResults sets the query results for a given query on a private collection
func (m *QueryExecutor) WithPrivateQueryResults(ns, coll, query string, results [][]byte) *QueryExecutor {
	m.queryResults[privateQueryResultsKey(ns, coll, query)] = results
	return m
}

// WithError injects an error to the mock executor
func (m *QueryExecutor) WithError(err error) *QueryExecutor {
	m.error = err
	return m
}

// GetState returns the mock state for the given namespace and key
func (m *QueryExecutor) GetState(namespace string, key string) ([]byte, error) {
	if m.error != nil {
		return nil, m.error
	}

	ns := m.state[namespace]
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
	return m.GetState(privateNamespace(namespace, collection), key)
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
	return m.GetStateMultipleKeys(privateNamespace(namespace, collection), keys)
}

// GetPrivateDataRangeScanIterator is not currently implemented and will panic if called
func (m *QueryExecutor) GetPrivateDataRangeScanIterator(namespace, collection, startKey, endKey string) (commonledger.ResultsIterator, error) {
	panic("not implemented")
}

// ExecuteQueryOnPrivateData is not currently implemented and will panic if called
func (m *QueryExecutor) ExecuteQueryOnPrivateData(namespace, collection, query string) (commonledger.ResultsIterator, error) {
	return NewResultsIterator().WithResults(m.queryResults[privateQueryResultsKey(namespace, collection, query)]), m.error
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

func privateNamespace(namespace, collection string) string {
	return namespace + "$" + collection
}

func queryResultsKey(namespace, query string) string {
	return namespace + "~" + query
}

func privateQueryResultsKey(namespace, coll, query string) string {
	return privateNamespace(namespace, coll) + "~" + query
}
