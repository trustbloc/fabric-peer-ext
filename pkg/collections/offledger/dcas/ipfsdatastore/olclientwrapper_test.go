/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ipfsdatastore

import (
	"fmt"
	"testing"

	"github.com/ipfs/go-datastore"
	"github.com/stretchr/testify/require"
	olmocks "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/mocks"
)

//go:generate counterfeiter -o ../../mocks/offledgerclient.gen.go --fake-name OffLedgerClient ../../../client OffLedger

const (
	ns1   = "ns1"
	coll1 = "coll1"

	key1 = "key1"
)

func TestOffLedgerClientWrapper_Put(t *testing.T) {
	value1 := []byte("value1")

	c := NewOLClientWrapper(ns1, coll1, &olmocks.OffLedgerClient{})
	require.NotNil(t, c)
	require.NoError(t, c.Put(datastore.NewKey(key1), value1))
}

func TestOffLedgerClientWrapper_Get(t *testing.T) {
	t.Run("Get - not found -> success", func(t *testing.T) {
		c := NewOLClientWrapper(ns1, coll1, &olmocks.OffLedgerClient{})
		require.NotNil(t, c)
		value, err := c.Get(datastore.NewKey(key1))
		require.EqualError(t, err, datastore.ErrNotFound.Error())
		require.Empty(t, value)
	})

	t.Run("Get -> success", func(t *testing.T) {
		value1 := []byte("value1")

		olClient := &olmocks.OffLedgerClient{}
		olClient.GetReturns(value1, nil)

		c := NewOLClientWrapper(ns1, coll1, olClient)
		require.NotNil(t, c)

		value, err := c.Get(datastore.NewKey(key1))
		require.NoError(t, err)
		require.Equal(t, value1, value)
	})

	t.Run("Get - olclient error -> error", func(t *testing.T) {
		errExpected := fmt.Errorf("injected OL client error")

		olClient := &olmocks.OffLedgerClient{}
		olClient.GetReturns(nil, errExpected)

		c := NewOLClientWrapper(ns1, coll1, olClient)
		require.NotNil(t, c)

		value, err := c.Get(datastore.NewKey(key1))
		require.EqualError(t, err, errExpected.Error())
		require.Empty(t, value)
	})
}
