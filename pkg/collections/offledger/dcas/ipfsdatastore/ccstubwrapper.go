/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ipfsdatastore

import (
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/ipfs/go-datastore"
)

// ChaincodeStubWrapper implements the github.com/ipfs/go-datastore.Batching interface.
// This implementation uses a chaincode stub to store and retrieve data.
type ChaincodeStubWrapper struct {
	*dataStore
	stub       shim.ChaincodeStubInterface
	collection string
}

// NewStubWrapper returns a stub-wrapping IPFS data store
func NewStubWrapper(coll string, stub shim.ChaincodeStubInterface) *ChaincodeStubWrapper {
	return &ChaincodeStubWrapper{
		stub:       stub,
		collection: coll,
	}
}

// Get retrieves the object `value` named by `key`.
// Get will return ErrNotFound if the key is not mapped to a value.
func (s *ChaincodeStubWrapper) Get(key datastore.Key) ([]byte, error) {
	logger.Debugf("Getting key %s", key)

	v, err := s.stub.GetPrivateData(s.collection, key.String())
	if err != nil {
		return nil, err
	}

	if len(v) == 0 {
		return nil, datastore.ErrNotFound
	}

	return v, nil
}

// Put stores the object `value` named by `key`.
func (s *ChaincodeStubWrapper) Put(key datastore.Key, value []byte) error {
	logger.Infof("Putting key %s, Value: %s", key, value)

	return s.stub.PutPrivateData(s.collection, key.String(), value)
}
