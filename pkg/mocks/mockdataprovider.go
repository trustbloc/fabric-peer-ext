/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"context"

	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
)

// DataProvider is a mock transient data provider
type DataProvider struct {
	data         map[storeapi.Key]*storeapi.ExpiringValue
	queryResults map[storeapi.QueryKey][]*storeapi.QueryResult
	err          error
}

// NewDataProvider returns a new Data Provider
func NewDataProvider() *DataProvider {
	return &DataProvider{
		data:         make(map[storeapi.Key]*storeapi.ExpiringValue),
		queryResults: make(map[storeapi.QueryKey][]*storeapi.QueryResult),
	}
}

// WithData sets the data to be returned by the retriever
func (p *DataProvider) WithData(key *storeapi.Key, value *storeapi.ExpiringValue) *DataProvider {
	p.data[*key] = value
	return p
}

// WithQueryResults sets the mock query results for the given query string
func (p *DataProvider) WithQueryResults(key *storeapi.QueryKey, results []*storeapi.QueryResult) *DataProvider {
	p.queryResults[*key] = results
	return p
}

// WithError sets the error to be returned by the retriever
func (p *DataProvider) WithError(err error) *DataProvider {
	p.err = err
	return p
}

// RetrieverForChannel returns the retriever for the given channel
func (p *DataProvider) RetrieverForChannel(channel string) storeapi.Retriever {
	return &dataRetriever{
		err:          p.err,
		data:         p.data,
		queryResults: p.queryResults,
	}
}

type dataRetriever struct {
	err          error
	data         map[storeapi.Key]*storeapi.ExpiringValue
	queryResults map[storeapi.QueryKey][]*storeapi.QueryResult
}

// GetTransientData returns the transient data for the given context and key
func (m *dataRetriever) GetTransientData(ctxt context.Context, key *storeapi.Key) (*storeapi.ExpiringValue, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.data[*key], nil
}

// GetTransientDataMultipleKeys returns the transient data with multiple keys for the given context and key
func (m *dataRetriever) GetTransientDataMultipleKeys(ctxt context.Context, key *storeapi.MultiKey) (storeapi.ExpiringValues, error) {
	if m.err != nil {
		return nil, m.err
	}
	values := make(storeapi.ExpiringValues, len(key.Keys))
	for i, k := range key.Keys {
		values[i] = m.data[*storeapi.NewKey(key.EndorsedAtTxID, key.Namespace, key.Collection, k)]
	}
	return values, nil
}

// GetData returns the data for the given context and key
func (m *dataRetriever) GetData(ctxt context.Context, key *storeapi.Key) (*storeapi.ExpiringValue, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.data[*key], nil
}

// GetDataMultipleKeys returns the  data with multiple keys for the given context and key
func (m *dataRetriever) GetDataMultipleKeys(ctxt context.Context, key *storeapi.MultiKey) (storeapi.ExpiringValues, error) {
	if m.err != nil {
		return nil, m.err
	}
	values := make(storeapi.ExpiringValues, len(key.Keys))
	for i, k := range key.Keys {
		values[i] = m.data[*storeapi.NewKey(key.EndorsedAtTxID, key.Namespace, key.Collection, k)]
	}
	return values, nil
}

// Query executes the given rich query
func (m *dataRetriever) Query(ctxt context.Context, key *storeapi.QueryKey) (storeapi.ResultsIterator, error) {
	if m.err != nil {
		return nil, m.err
	}
	return newResultsIterator(m.queryResults[*key]), nil
}

type resultsIterator struct {
	results []*storeapi.QueryResult
	nextIdx int
}

func newResultsIterator(results []*storeapi.QueryResult) *resultsIterator {
	return &resultsIterator{
		results: results,
	}
}

// Next returns the next item in the result set. The `QueryResult` is expected to be nil when
// the iterator gets exhausted
func (it *resultsIterator) Next() (*storeapi.QueryResult, error) {
	if it.nextIdx >= len(it.results) {
		return nil, nil
	}
	qr := it.results[it.nextIdx]
	it.nextIdx++
	return qr, nil
}

// Close releases resources occupied by the iterator
func (it *resultsIterator) Close() {
}
