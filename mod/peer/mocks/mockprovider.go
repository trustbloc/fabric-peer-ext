/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"context"

	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
)

// DataProvider is a mock data provider
type DataProvider struct {
}

// RetrieverForChannel retrieve data for channel
func (p *DataProvider) RetrieverForChannel(channel string) storeapi.Retriever {
	return &dataRetriever{}
}

type dataRetriever struct {
}

// GetTransientData returns the transient data for the given context and key
func (m *dataRetriever) GetTransientData(ctxt context.Context, key *storeapi.Key) (*storeapi.ExpiringValue, error) {
	return &storeapi.ExpiringValue{Value: []byte(key.Key)}, nil
}

// GetTransientDataMultipleKeys returns the transient data with multiple keys for the given context and key
func (m *dataRetriever) GetTransientDataMultipleKeys(ctxt context.Context, key *storeapi.MultiKey) (storeapi.ExpiringValues, error) {
	values := make(storeapi.ExpiringValues, len(key.Keys))
	for i, k := range key.Keys {
		values[i] = &storeapi.ExpiringValue{Value: []byte(k)}
	}
	return values, nil
}

// GetData returns the data for the given context and key
func (m *dataRetriever) GetData(ctxt context.Context, key *storeapi.Key) (*storeapi.ExpiringValue, error) {
	return &storeapi.ExpiringValue{Value: []byte(key.Key)}, nil
}

// GetDataMultipleKeys returns the  data with multiple keys for the given context and key
func (m *dataRetriever) GetDataMultipleKeys(ctxt context.Context, key *storeapi.MultiKey) (storeapi.ExpiringValues, error) {
	values := make(storeapi.ExpiringValues, len(key.Keys))
	for i, k := range key.Keys {
		values[i] = &storeapi.ExpiringValue{Value: []byte(k)}
	}
	return values, nil
}
