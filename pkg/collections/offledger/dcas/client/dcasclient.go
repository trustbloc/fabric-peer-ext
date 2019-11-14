/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"github.com/btcsuite/btcutil/base58"
	commonledger "github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/protos/ledger/queryresult"
	olclient "github.com/trustbloc/fabric-peer-ext/pkg/collections/client"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/dcas"
)

// DCASClient allows you to put and get DCASClient from outside of a chaincode
type DCASClient struct {
	*olclient.Client
}

// New returns a new DCAS client
func New(channelID string, providers *olclient.ChannelProviders) *DCASClient {
	return &DCASClient{
		Client: olclient.New(channelID, providers),
	}
}

// Put puts the DCAS value and returns the key for the value
func (d *DCASClient) Put(ns, coll string, value []byte) (string, error) {
	keys, err := d.PutMultipleValues(ns, coll, [][]byte{value})
	if err != nil {
		return "", err
	}
	return keys[0], nil
}

// PutMultipleValues puts the DCAS values and returns the keys for the values
func (d *DCASClient) PutMultipleValues(ns, coll string, values [][]byte) ([]string, error) {
	keys := make([]string, len(values))
	kvs := make([]*olclient.KeyValue, len(values))
	for i, v := range values {
		key, bytes, err := dcas.GetCASKeyAndValue(v)
		if err != nil {
			return nil, err
		}
		keys[i] = key
		kvs[i] = &olclient.KeyValue{
			Key:   base58.Encode([]byte(key)), // Encode the key in base58 before saving to Fabric
			Value: bytes,
		}
	}
	if err := d.Client.PutMultipleValues(ns, coll, kvs); err != nil {
		return nil, err
	}
	return keys, nil
}

// Delete deletes the given key(s). The key(s) must be the base64-encoded hash of the value.
func (d *DCASClient) Delete(ns, coll string, keys ...string) error {
	base58Keys := make([]string, len(keys))
	for i, key := range keys {
		base58Keys[i] = base58.Encode([]byte(key))
	}
	return d.Client.Delete(ns, coll, base58Keys...)
}

// Get retrieves the value for the given key. The key must be the base64-encoded hash of the value.
func (d *DCASClient) Get(ns, coll, key string) ([]byte, error) {
	return d.Client.Get(ns, coll, base58.Encode([]byte(key)))
}

// GetMultipleKeys retrieves the values for the given keys. The key(s) must be the base64-encoded hash of the value.
func (d *DCASClient) GetMultipleKeys(ns, coll string, keys ...string) ([][]byte, error) {
	base58Keys := make([]string, len(keys))
	for i, key := range keys {
		base58Keys[i] = base58.Encode([]byte(key))
	}
	return d.Client.GetMultipleKeys(ns, coll, base58Keys...)
}

// Query executes the given query and returns an iterator that contains results.
// Only used for state databases that support query.
// (Note that this function is not supported by transient data collections)
// The returned ResultsIterator contains results of type *KV which is defined in protos/ledger/queryresult.
func (d *DCASClient) Query(ns, coll, query string) (commonledger.ResultsIterator, error) {
	it, err := d.Client.Query(ns, coll, query)
	if err != nil {
		return nil, err
	}
	return &decodingResultsIterator{
		target: it,
	}, nil
}

type decodingResultsIterator struct {
	target commonledger.ResultsIterator
}

// Next returns the next item in the result set. The key is base58-decoded before it is returned.
func (it *decodingResultsIterator) Next() (commonledger.QueryResult, error) {
	qr, err := it.target.Next()
	if err != nil {
		return nil, err
	}
	if qr == nil {
		return nil, nil
	}

	kv := qr.(*queryresult.KV)
	dkv := *kv
	dkv.Key = string(base58.Decode(kv.Key))
	return &dkv, nil
}

// Close releases resources occupied by the iterator
func (it *decodingResultsIterator) Close() {
	it.target.Close()
}
