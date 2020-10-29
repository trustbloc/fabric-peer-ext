/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ipfsdatastore

import (
	"github.com/ipfs/go-datastore"

	"github.com/trustbloc/fabric-peer-ext/pkg/collections/client"
)

// OffLedgerClientWrapper implements the github.com/ipfs/go-datastore.Batching interface.
// This implementation uses an off-ledger client to store and retrieve data.
type OffLedgerClientWrapper struct {
	*dataStore
	olclient   client.OffLedger
	namespace  string
	collection string
}

// NewOLClientWrapper returns an off-ledger client-wrapping IPFS data store
func NewOLClientWrapper(ns, coll string, olclient client.OffLedger) *OffLedgerClientWrapper {
	return &OffLedgerClientWrapper{
		olclient:   olclient,
		namespace:  ns,
		collection: coll,
	}
}

// Get retrieves the object `value` named by `key`.
// Get will return ErrNotFound if the key is not mapped to a value.
func (s *OffLedgerClientWrapper) Get(key datastore.Key) ([]byte, error) {
	logger.Debugf("Getting key %s", key)

	v, err := s.olclient.Get(s.namespace, s.collection, key.String())
	if err != nil {
		return nil, err
	}

	if len(v) == 0 {
		return nil, datastore.ErrNotFound
	}

	return v, nil
}

// Put stores the object `value` named by `key`.
func (s *OffLedgerClientWrapper) Put(key datastore.Key, value []byte) error {
	logger.Debugf("Putting key %s, Size: %d", key, len(value))

	return s.olclient.Put(s.namespace, s.collection, key.String(), value)
}
