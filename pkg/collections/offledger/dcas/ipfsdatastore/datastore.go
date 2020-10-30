/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ipfsdatastore

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
)

var logger = flogging.MustGetLogger("ext_offledger")

type dataStore struct {
}

// Has returns whether the `key` is mapped to a `value`.
// This function is only called just before a Put is called.
// It is cheaper to always return false and let the 'Put' go through
// than to ask for the data from other peers before each Put.
func (s *dataStore) Has(key datastore.Key) (bool, error) {
	return false, nil
}

// GetSize returns the size of the `value` named by `key`.
// Note: This function is never called for the DCAS so we'll leave
// it unimplemented. If it is ever called in the future then
// an implementation will need to be provided.
func (s *dataStore) GetSize(key datastore.Key) (int, error) {
	panic("not implemented")
}

// Query searches the datastore and returns a query result.
// Note: This function is never called for the DCAS so we'll leave
// it unimplemented. If it is ever called in the future then
// an implementation will need to be provided.
func (s *dataStore) Query(q query.Query) (query.Results, error) {
	panic("not implemented")
}

// Sync does nothing.
func (s *dataStore) Sync(prefix datastore.Key) error {
	// No-op
	return nil
}

// Close does nothing
func (s *dataStore) Close() error {
	// No-op
	return nil
}

// Batch is not supported
func (s *dataStore) Batch() (datastore.Batch, error) {
	// Not supported
	return nil, datastore.ErrBatchUnsupported
}

// Delete removes the value for given `key`.
// Note: This function is never called for the DCAS so we'll leave
// it unimplemented. If it is ever called in the future then
// an implementation will need to be provided.
func (s *dataStore) Delete(key datastore.Key) error {
	panic("not implemented")
}
