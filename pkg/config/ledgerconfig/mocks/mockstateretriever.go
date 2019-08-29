/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	commonledger "github.com/hyperledger/fabric/common/ledger"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/compositekey"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/state/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

// StateRetriever is a mock implementation of StateRetriever
type StateRetriever struct {
	*mocks.QueryExecutor
	kvResultsItProvider func(it commonledger.ResultsIterator) *KVResultsIter
}

// NewStateRetriever returns a mock StateRetriever
func NewStateRetriever() *StateRetriever {
	return &StateRetriever{
		QueryExecutor:       mocks.NewQueryExecutor(),
		kvResultsItProvider: NewKVResultsIter,
	}
}

// WithKVResultsIteratorProvider sets the KV results iterator provider
func (m *StateRetriever) WithKVResultsIteratorProvider(p func(it commonledger.ResultsIterator) *KVResultsIter) *StateRetriever {
	m.kvResultsItProvider = p
	return m
}

// GetStateByPartialCompositeKey returns an iterator for the range of keys/values for the given partial keys
func (m *StateRetriever) GetStateByPartialCompositeKey(namespace, objectType string, attributes []string) (api.ResultsIterator, error) {
	startKey, endKey := compositekey.CreateRangeKeysForPartialCompositeKey(objectType, attributes)
	it, err := m.GetStateRangeScanIterator(namespace, startKey, endKey)
	if err != nil {
		return nil, err
	}
	return m.kvResultsItProvider(it), nil
}
