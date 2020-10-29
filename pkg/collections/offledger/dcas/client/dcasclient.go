/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"io"
	"io/ioutil"

	"github.com/btcsuite/btcutil/base58"
	olclient "github.com/trustbloc/fabric-peer-ext/pkg/collections/client"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/dcas"
)

// DCASClient allows you to put and get DCASClient from outside of a chaincode
type DCASClient struct {
	client   *olclient.Client
	ns, coll string
}

// New returns a new DCAS client
func New(channelID, ns, coll string, providers *olclient.ChannelProviders) *DCASClient {
	return &DCASClient{
		ns:     ns,
		coll:   coll,
		client: olclient.New(channelID, providers),
	}
}

// Put puts the data and returns the content ID (CID) for the value
func (d *DCASClient) Put(data io.Reader) (string, error) {
	bytes, err := ioutil.ReadAll(data)
	if err != nil {
		return "", err
	}

	key, v, err := dcas.GetCASKeyAndValue(bytes)
	if err != nil {
		return "", err
	}

	k58 := base58.Encode([]byte(key)) // Encode the key in base58 before saving to Fabric

	err = d.client.Put(d.ns, d.coll, k58, v)
	if err != nil {
		return "", err
	}

	return key, nil
}

// Delete deletes the values for the given content IDs
func (d *DCASClient) Delete(cids ...string) error {
	base58Keys := make([]string, len(cids))
	for i, key := range cids {
		base58Keys[i] = base58.Encode([]byte(key))
	}
	return d.client.Delete(d.ns, d.coll, base58Keys...)
}

// Get retrieves the value for the given content ID (CID).
func (d *DCASClient) Get(cid string, w io.Writer) error {
	v, err := d.client.Get(d.ns, d.coll, base58.Encode([]byte(cid)))
	if err != nil {
		return err
	}

	_, err = w.Write(v)
	return err
}
