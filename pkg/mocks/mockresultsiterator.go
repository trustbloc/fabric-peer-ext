/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	commonledger "github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
)

// ResultsIterator is a mock key-value iterator
type ResultsIterator struct {
	results []*statedb.VersionedKV
	nextIdx int
	err     error
}

// NewResultsIterator returns a mock key-value iterator
func NewResultsIterator() *ResultsIterator {
	return &ResultsIterator{}
}

// WithResults sets the mock results
func (m *ResultsIterator) WithResults(results []*statedb.VersionedKV) *ResultsIterator {
	m.results = results
	return m
}

// WithError injects an error
func (m *ResultsIterator) WithError(err error) *ResultsIterator {
	m.err = err
	return m
}

// Next returns the next item in the result set. The `QueryResult` is expected to be nil when
// the iterator gets exhausted
func (m *ResultsIterator) Next() (commonledger.QueryResult, error) {
	if m.err != nil {
		return nil, m.err
	}

	if m.nextIdx >= len(m.results) {
		return nil, nil
	}
	qr := m.results[m.nextIdx]
	m.nextIdx++
	return qr, nil
}

// Close releases resources occupied by the iterator
func (m *ResultsIterator) Close() {
}
