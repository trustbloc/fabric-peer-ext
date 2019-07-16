/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dcasclient

import (
	olclient "github.com/trustbloc/fabric-peer-ext/pkg/collections/client"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/dcas"
)

// DCASClient allows you to put and get DCASClient from outside of a chaincode
type DCASClient struct {
	*olclient.Client
}

// New returns a new DCAS olclient
func New(channelID string) *DCASClient {
	return &DCASClient{
		Client: olclient.New(channelID),
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
			Key:   key,
			Value: bytes,
		}
	}
	if err := d.Client.PutMultipleValues(ns, coll, kvs); err != nil {
		return nil, err
	}
	return keys, nil
}
