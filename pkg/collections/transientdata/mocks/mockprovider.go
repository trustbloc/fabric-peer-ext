/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"context"

	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	tdataapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/api"
)

// TransientDataProvider is a mock transient data provider
type TransientDataProvider struct {
}

// RetrieverForChannel returns a provider for the given channel
func (p *TransientDataProvider) RetrieverForChannel(channel string) tdataapi.Retriever {
	return &transientDataRetriever{}
}

type transientDataRetriever struct {
}

// GetTransientData returns the transientData
func (m *transientDataRetriever) GetTransientData(ctxt context.Context, key *storeapi.Key) (*storeapi.ExpiringValue, error) {
	return &storeapi.ExpiringValue{Value: []byte(key.Key)}, nil
}

// GetTransientDataMultipleKeys returns the data with multiple keys
func (m *transientDataRetriever) GetTransientDataMultipleKeys(ctxt context.Context, key *storeapi.MultiKey) (storeapi.ExpiringValues, error) {
	values := make(storeapi.ExpiringValues, len(key.Keys))
	for i, k := range key.Keys {
		values[i] = &storeapi.ExpiringValue{Value: []byte(k)}
	}
	return values, nil
}
