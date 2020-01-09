/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"sort"
	"strings"

	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	commonledger "github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
)

// QueryExecutor is a mock query executor
type QueryExecutor struct {
	state        map[string]map[string][]byte
	queryResults map[string][]*queryresult.KV
	error        error
	queryError   error
	itProvider   func() *ResultsIterator
}

// NewQueryExecutor returns a new mock query executor
func NewQueryExecutor() *QueryExecutor {
	return &QueryExecutor{
		state:        make(map[string]map[string][]byte),
		queryResults: make(map[string][]*queryresult.KV),
		itProvider:   NewResultsIterator,
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

// WithDeletedState deletes the state
func (m *QueryExecutor) WithDeletedState(ns, key string) *QueryExecutor {
	nsState, ok := m.state[ns]
	if ok {
		delete(nsState, key)
	}
	return m
}

// WithDeletedPrivateState deletes the state
func (m *QueryExecutor) WithDeletedPrivateState(ns, collection, key string) *QueryExecutor {
	nsState, ok := m.state[privateNamespace(ns, collection)]
	if ok {
		delete(nsState, key)
	}
	return m
}

// WithQueryResults sets the query results for a given query on a namespace
func (m *QueryExecutor) WithQueryResults(ns, query string, results []*queryresult.KV) *QueryExecutor {
	m.queryResults[queryResultsKey(ns, query)] = results
	return m
}

// WithPrivateQueryResults sets the query results for a given query on a private collection
func (m *QueryExecutor) WithPrivateQueryResults(ns, coll, query string, results []*queryresult.KV) *QueryExecutor {
	m.queryResults[privateQueryResultsKey(ns, coll, query)] = results
	return m
}

// WithIteratorProvider sets the results iterator provider
func (m *QueryExecutor) WithIteratorProvider(p func() *ResultsIterator) *QueryExecutor {
	m.itProvider = p
	return m
}

// WithError injects an error to the mock executor
func (m *QueryExecutor) WithError(err error) *QueryExecutor {
	m.error = err
	return m
}

// WithQueryError injects an error to the mock executor for queries
func (m *QueryExecutor) WithQueryError(err error) *QueryExecutor {
	m.queryError = err
	return m
}

// GetState returns the mock state for the given namespace and key
func (m *QueryExecutor) GetState(namespace string, key string) ([]byte, error) {
	if m.error != nil {
		return nil, m.error
	}
	return m.state[namespace][key], nil
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

// GetStateRangeScanIterator returns an iterator for mock results ranging from startKey (inclusive) to endKey (exclusive)
func (m *QueryExecutor) GetStateRangeScanIterator(namespace string, startKey string, endKey string) (commonledger.ResultsIterator, error) {
	if m.queryError != nil {
		return nil, m.queryError
	}

	var kvs []*queryresult.KV
	for key, value := range m.state[namespace] {
		if strings.Compare(key, startKey) < 0 || strings.Compare(key, endKey) >= 0 {
			continue
		}
		kvs = append(kvs, &queryresult.KV{
			Namespace: namespace,
			Key:       key,
			Value:     value,
		})
	}
	sort.Slice(kvs, func(i, j int) bool {
		return strings.Compare(kvs[i].Key, kvs[j].Key) < 0
	})

	return m.itProvider().WithResults(kvs), nil
}

// GetStateRangeScanIteratorWithMetadata is not currently implemented and will panic if called
func (m *QueryExecutor) GetStateRangeScanIteratorWithMetadata(namespace string, startKey, endKey string, metadata map[string]interface{}) (ledger.QueryResultsIterator, error) {
	panic("not implemented")
}

// ExecuteQuery returns mock results for the given query
func (m *QueryExecutor) ExecuteQuery(namespace, query string) (commonledger.ResultsIterator, error) {
	if m.queryError != nil {
		return nil, m.queryError
	}
	return m.itProvider().WithResults(m.queryResults[queryResultsKey(namespace, query)]), m.error
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

// ExecuteQueryOnPrivateData  returns mock results for the given query
func (m *QueryExecutor) ExecuteQueryOnPrivateData(namespace, collection, query string) (commonledger.ResultsIterator, error) {
	if m.error != nil {
		return nil, m.error
	}
	return m.itProvider().WithResults(m.queryResults[privateQueryResultsKey(namespace, collection, query)]), m.error
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

// QueryExecutorProvider is a mock query executor provider
type QueryExecutorProvider struct {
	qe  *QueryExecutor
	err error
}

// NewQueryExecutorProvider returns a mock query executor provider
func NewQueryExecutorProvider() *QueryExecutorProvider {
	return &QueryExecutorProvider{
		qe: NewQueryExecutor(),
	}
}

// WithMockQueryExecutor sets the mock query executor
func (m *QueryExecutorProvider) WithMockQueryExecutor(qe *QueryExecutor) *QueryExecutorProvider {
	m.qe = qe
	return m
}

// WithError injects an error into the mock provider
func (m *QueryExecutorProvider) WithError(err error) *QueryExecutorProvider {
	m.err = err
	return m
}

// GetQueryExecutorForLedger returns the query executor for the given channel ID
func (m *QueryExecutorProvider) GetQueryExecutorForLedger(channelID string) (ledger.QueryExecutor, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.qe, nil
}

// KVIterator is a mock key-value iterator
type KVIterator struct {
	kvs     []*statedb.VersionedKV
	nextIdx int
	err     error
}

// NewKVIterator returns a mock key-value iterator
func NewKVIterator(kvs []*statedb.VersionedKV) *KVIterator {
	return &KVIterator{
		kvs: kvs,
	}
}

// WithError injects an error
func (it *KVIterator) WithError(err error) *KVIterator {
	it.err = err
	return it
}

// Next returns the next item in the result set. The `QueryResult` is expected to be nil when
// the iterator gets exhausted
func (it *KVIterator) Next() (commonledger.QueryResult, error) {
	if it.err != nil {
		return nil, it.err
	}

	if it.nextIdx >= len(it.kvs) {
		return nil, nil
	}
	qr := it.kvs[it.nextIdx]
	it.nextIdx++
	return qr, nil
}

// Close releases resources occupied by the iterator
func (it *KVIterator) Close() {
}
