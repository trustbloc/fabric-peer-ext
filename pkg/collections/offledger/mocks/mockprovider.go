/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"context"

	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	olapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/api"
)

// Provider is a mock off-ledger data data provider
type Provider struct {
}

// RetrieverForChannel returns the retriever for the given channel
func (p *Provider) RetrieverForChannel(channel string) olapi.Retriever {
	return &retriever{}
}

type retriever struct {
}

// GetData gets data for the given key
func (m *retriever) GetData(ctxt context.Context, key *storeapi.Key) (*storeapi.ExpiringValue, error) {
	return &storeapi.ExpiringValue{Value: []byte(key.Key)}, nil
}

// GetDataMultipleKeys gets data for multiple keys
func (m *retriever) GetDataMultipleKeys(ctxt context.Context, key *storeapi.MultiKey) (storeapi.ExpiringValues, error) {
	values := make(storeapi.ExpiringValues, len(key.Keys))
	for i, k := range key.Keys {
		values[i] = &storeapi.ExpiringValue{Value: []byte(k)}
	}
	return values, nil
}

func (m *retriever) Query(ctxt context.Context, key *storeapi.QueryKey) (storeapi.ResultsIterator, error) {
	return newResultsIterator(), nil
}

type resultsIterator struct {
}

func newResultsIterator() *resultsIterator {
	return &resultsIterator{}
}

func (it *resultsIterator) Next() (*storeapi.QueryResult, error) {
	return nil, nil
}

func (it *resultsIterator) Close() {
}
