/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ipfsdatastore

import (
	"testing"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/stretchr/testify/require"
)

func TestDataStore_Batch(t *testing.T) {
	ds := &dataStore{}

	b, err := ds.Batch()
	require.EqualError(t, err, datastore.ErrBatchUnsupported.Error())
	require.Nil(t, b)
}

func TestDataStore_Close(t *testing.T) {
	ds := &dataStore{}
	require.NoError(t, ds.Close())
}

func TestDataStore_Sync(t *testing.T) {
	ds := &dataStore{}
	require.NoError(t, ds.Sync(datastore.NewKey(key1)))
}

func TestDataStore_Has(t *testing.T) {
	ds := &dataStore{}
	has, err := ds.Has(datastore.NewKey(key1))
	require.NoError(t, err)
	require.False(t, has)
}

func TestDataStore_Query(t *testing.T) {
	ds := &dataStore{}
	require.Panics(t, func() {
		_, err := ds.Query(query.Query{})
		require.NoError(t, err)
	})
}

func TestDataStore_Delete(t *testing.T) {
	ds := &dataStore{}
	require.Panics(t, func() {
		require.NoError(t, ds.Delete(datastore.NewKey(key1)))
	})
}

func TestDataStore_GetSize(t *testing.T) {
	ds := &dataStore{}
	require.Panics(t, func() {
		_, err := ds.GetSize(datastore.NewKey(key1))
		require.NoError(t, err)
	})
}
