/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package state

import (
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/state/api"
)

// ShimStoreProvider is a RetrieverProvider which uses the chaincode stub to store and retrieve data
type ShimStoreProvider struct {
	store *shimStore
}

// NewShimStoreProvider returns a new ShimStoreProvider
func NewShimStoreProvider(stub shim.ChaincodeStubInterface) *ShimStoreProvider {
	return &ShimStoreProvider{
		store: &shimStore{
			stub: stub,
		},
	}
}

// GetStore returns the state store
func (p *ShimStoreProvider) GetStore() (api.StateStore, error) {
	return p.store, nil
}

// GetStateRetriever returns the state store
func (p *ShimStoreProvider) GetStateRetriever() (api.StateRetriever, error) {
	return p.store, nil
}

type shimStore struct {
	stub shim.ChaincodeStubInterface
}

// PutState saves the value for the given key
func (s *shimStore) PutState(_, key string, value []byte) error {
	return s.stub.PutState(key, value)
}

// GetState returns the value for the given key
func (s *shimStore) GetState(_, key string) ([]byte, error) {
	return s.stub.GetState(key)
}

// DelState deletes the given key
func (s *shimStore) DelState(_, key string) error {
	return s.stub.DelState(key)
}

// GetStateByPartialCompositeKey returns an iterator for the given index and attributes
func (s *shimStore) GetStateByPartialCompositeKey(_, objectType string, attributes []string) (api.ResultsIterator, error) {
	return s.stub.GetStateByPartialCompositeKey(objectType, attributes)
}

// Done does nothing
func (s *shimStore) Done() {
	// Nothing to do
}
