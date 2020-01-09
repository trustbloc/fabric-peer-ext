/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package state

import (
	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	commonledger "github.com/hyperledger/fabric/common/ledger"
	"github.com/pkg/errors"
)

// KVResultsIter is a key-value results iterator
type KVResultsIter struct {
	it      commonledger.ResultsIterator
	next    commonledger.QueryResult
	nextErr error
}

// NewKVResultsIter returns a new KVResultsIter
func NewKVResultsIter(it commonledger.ResultsIterator) *KVResultsIter {
	return &KVResultsIter{it: it}
}

// HasNext returns true if there are more items
func (it *KVResultsIter) HasNext() bool {
	queryResult, err := it.it.Next()
	if err != nil {
		// Save the error and return true. The caller will get the error when Next is called.
		it.nextErr = err
		return true
	}
	if queryResult == nil {
		return false
	}
	it.next = queryResult
	return true
}

// Next returns the next item
func (it *KVResultsIter) Next() (*queryresult.KV, error) {
	if it.nextErr != nil {
		return nil, it.nextErr
	}

	queryResult := it.next
	if queryResult == nil {
		qr, err := it.it.Next()
		if err != nil {
			return nil, err
		}
		queryResult = qr
	} else {
		it.next = nil
	}

	if queryResult == nil {
		return nil, errors.New("Next() called when there is no next")
	}

	versionedKV := queryResult.(*queryresult.KV)
	return &queryresult.KV{
		Namespace: versionedKV.Namespace,
		Key:       versionedKV.Key,
		Value:     versionedKV.Value,
	}, nil
}

// Close closes the iterator
func (it *KVResultsIter) Close() error {
	it.it.Close()
	return nil
}
