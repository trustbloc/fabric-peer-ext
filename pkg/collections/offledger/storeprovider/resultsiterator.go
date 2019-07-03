/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package storeprovider

import (
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
)

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
