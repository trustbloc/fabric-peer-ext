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
	data map[storeapi.Key]*storeapi.ExpiringValue
	err  error
}

// NewDataProvider returns a new Data Provider
func NewDataProvider() *DataProvider {
	return &DataProvider{
		data: make(map[storeapi.Key]*storeapi.ExpiringValue),
	}
}

// WithData sets the data to be returned by the retriever
func (p *DataProvider) WithData(key *storeapi.Key, value *storeapi.ExpiringValue) *DataProvider {
	p.data[*key] = value
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
		err:  p.err,
		data: p.data,
	}
}

type dataRetriever struct {
	err  error
	data map[storeapi.Key]*storeapi.ExpiringValue
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
