/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	commonledger "github.com/hyperledger/fabric/common/ledger"
)

// KVResultsIter is a mock key-value results iterator
type KVResultsIter struct {
	it       commonledger.ResultsIterator
	next     commonledger.QueryResult
	nextErr  error
	closeErr error
}

// NewKVResultsIter returns a new KVResultsIter
func NewKVResultsIter(it commonledger.ResultsIterator) *KVResultsIter {
	return &KVResultsIter{it: it}
}

// WithCloseError injects an error into the mock iterator
func (m *KVResultsIter) WithCloseError(err error) *KVResultsIter {
	m.closeErr = err
	return m
}

// HasNext returns true if there are more items
func (m *KVResultsIter) HasNext() bool {
	queryResult, err := m.it.Next()
	if err != nil {
		// Save the error and return true. The caller will get the error when Next is called.
		m.nextErr = err
		return true
	}
	if queryResult == nil {
		return false
	}
	m.next = queryResult
	return true
}

// Next returns the next item
func (m *KVResultsIter) Next() (*queryresult.KV, error) {
	if m.nextErr != nil {
		return nil, m.nextErr
	}

	queryResult := m.next
	if queryResult == nil {
		qr, err := m.it.Next()
		if err != nil {
			return nil, err
		}
		queryResult = qr
	} else {
		m.next = nil
	}

	if queryResult == nil {
		return nil, nil
	}

	versionedKV := queryResult.(*queryresult.KV)
	return &queryresult.KV{
		Namespace: versionedKV.Namespace,
		Key:       versionedKV.Key,
		Value:     versionedKV.Value,
	}, nil
}

// Close closes the iterator
func (m *KVResultsIter) Close() error {
	if m.closeErr != nil {
		return m.closeErr
	}
	m.it.Close()
	return nil
}
