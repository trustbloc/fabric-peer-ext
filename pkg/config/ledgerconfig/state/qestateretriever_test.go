/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package state

import (
	"errors"
	"testing"

	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/common/compositekey"
	cmocks "github.com/trustbloc/fabric-peer-ext/pkg/common/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

const (
	index   = "index1"
	prefix1 = "prefix1"
)

func TestQERetriever_GetState(t *testing.T) {
	v1 := []byte("v1")

	t.Run("Success", func(t *testing.T) {
		db := &mocks.StateDB{}
		db.GetStateReturns(v1, nil)

		rp := NewQERetrieverProvider(db)

		r, err := rp.GetStateRetriever()
		require.NoError(t, err)
		require.NotNil(t, r)

		v, err := r.GetState(ns1, key1)
		require.NoError(t, err)
		require.Equal(t, v1, v)
	})

	t.Run("Iterator error", func(t *testing.T) {
		errExpected := errors.New("iterator error")
		db := &mocks.StateDB{}
		db.GetStateRangeScanIteratorReturns(nil, errExpected)

		rp := NewQERetrieverProvider(db)

		r, err := rp.GetStateRetriever()
		require.NoError(t, err)
		require.NotNil(t, r)

		it, err := r.GetStateByPartialCompositeKey(ns1, key1, nil)
		require.EqualError(t, err, errExpected.Error())
		require.Nil(t, it)
	})
}

func TestQERetriever_GetStateByPartialCompositeKey(t *testing.T) {
	db := &mocks.StateDB{}
	mit := &cmocks.ResultsIterator{}
	db.GetStateRangeScanIteratorReturns(mit, nil)

	rp := NewQERetrieverProvider(db)

	ck1 := compositekey.Create(index, []string{prefix1, key1})
	mit.NextReturnsOnCall(0, newVersionedKV(ns1, ck1, []byte("{}")), nil)
	ck2 := compositekey.Create(index, []string{prefix1, key2})
	mit.NextReturnsOnCall(1, newVersionedKV(ns1, ck2, []byte("{}")), nil)

	r, err := rp.GetStateRetriever()
	require.NoError(t, err)
	require.NotNil(t, r)
	defer r.Done()

	it, err := r.GetStateByPartialCompositeKey(ns1, index, []string{prefix1})
	require.NoError(t, err)
	require.NotNil(t, it)

	require.True(t, it.HasNext())
	kv, err := it.Next()
	require.NoError(t, err)
	require.NotNil(t, kv)

	indexName, keys := compositekey.Split(kv.Key)
	require.Equal(t, index, indexName)
	require.Equal(t, 2, len(keys))
	require.Equal(t, key1, keys[1])

	require.True(t, it.HasNext())
	kv, err = it.Next()
	require.NoError(t, err)
	require.NotNil(t, kv)

	indexName, keys = compositekey.Split(kv.Key)
	require.Equal(t, index, indexName)
	require.Equal(t, 2, len(keys))
	require.Equal(t, key2, keys[1])

	require.False(t, it.HasNext())

	kv, err = it.Next()
	require.EqualError(t, err, "Next() called when there is no next")
	require.Nil(t, kv)
	require.NoError(t, it.Close())
}

func TestQERetriever_GetStateByPartialCompositeKey_Error(t *testing.T) {
	errExpected := errors.New("iterator error")
	db := &mocks.StateDB{}
	mit := &cmocks.ResultsIterator{}
	mit.NextReturns(nil, errExpected)
	db.GetStateRangeScanIteratorReturns(mit, nil)

	rp := NewQERetrieverProvider(db)

	r, err := rp.GetStateRetriever()
	require.NoError(t, err)
	require.NotNil(t, r)
	defer r.Done()

	it, err := r.GetStateByPartialCompositeKey(ns1, index, []string{prefix1})
	require.NoError(t, err)
	require.NotNil(t, it)

	require.True(t, it.HasNext())
	kv, err := it.Next()
	require.EqualError(t, err, errExpected.Error())
	require.Nil(t, kv)
}

func newVersionedKV(ns, key string, value []byte) *statedb.VersionedKV {
	return &statedb.VersionedKV{
		CompositeKey: statedb.CompositeKey{
			Namespace: ns,
			Key:       key,
		},
		VersionedValue: statedb.VersionedValue{
			Value: value,
		},
	}
}
