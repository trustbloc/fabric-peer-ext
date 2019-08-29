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
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

const (
	channelID = "testchannel"

	index   = "index1"
	prefix1 = "prefix1"
	prefix2 = "prefix2"
)

func TestQERetriever_GetState(t *testing.T) {
	v1 := []byte("v1")

	t.Run("Success", func(t *testing.T) {
		rp := NewQERetrieverProvider(channelID,
			mocks.NewQueryExecutorProvider().
				WithMockQueryExecutor(mocks.NewQueryExecutor().WithState(ns1, key1, v1)),
		)

		r, err := rp.GetStateRetriever()
		require.NoError(t, err)
		require.NotNil(t, r)

		v, err := r.GetState(ns1, key1)
		require.NoError(t, err)
		require.Equal(t, v1, v)

		require.NotPanics(t, func() { r.Done() })
	})
	t.Run("QueryExecutorProvider error", func(t *testing.T) {
		errExpected := errors.New("query executor error")
		rp := NewQERetrieverProvider(channelID,
			mocks.NewQueryExecutorProvider().WithError(errExpected),
		)

		r, err := rp.GetStateRetriever()
		require.EqualError(t, err, errExpected.Error())
		require.Nil(t, r)
	})
	t.Run("Iterator error", func(t *testing.T) {
		errExpected := errors.New("iterator error")
		rp := NewQERetrieverProvider(channelID,
			mocks.NewQueryExecutorProvider().
				WithMockQueryExecutor(mocks.NewQueryExecutor().WithQueryError(errExpected)),
		)

		r, err := rp.GetStateRetriever()
		require.NoError(t, err)
		require.NotNil(t, r)

		it, err := r.GetStateByPartialCompositeKey(ns1, key1, nil)
		require.EqualError(t, err, errExpected.Error())
		require.Nil(t, it)
	})
}

func TestQERetriever_GetStateByPartialCompositeKey(t *testing.T) {
	qe := mocks.NewQueryExecutor()
	rp := NewQERetrieverProvider(
		channelID, mocks.NewQueryExecutorProvider().WithMockQueryExecutor(qe),
	)

	ck1 := compositekey.Create(index, []string{prefix1, key1})
	qe.WithState(ns1, ck1, []byte("{}"))

	ck2 := compositekey.Create(index, []string{prefix1, key2})
	qe.WithState(ns1, ck2, []byte("{}"))

	ck3 := compositekey.Create(index, []string{prefix2, key3})
	qe.WithState(ns1, ck3, []byte("{}"))

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

	it, err = r.GetStateByPartialCompositeKey(ns1, index, []string{prefix2})
	require.NoError(t, err)
	require.NotNil(t, it)

	require.True(t, it.HasNext())
	kv, err = it.Next()
	require.NoError(t, err)
	require.NotNil(t, kv)

	indexName, keys = compositekey.Split(kv.Key)
	require.Equal(t, index, indexName)
	require.Equal(t, 2, len(keys))
	require.Equal(t, key3, keys[1])

	require.False(t, it.HasNext())
	require.NoError(t, it.Close())
}

func TestQERetriever_GetStateByPartialCompositeKey_Error(t *testing.T) {
	errExpected := errors.New("iterator error")
	rp := NewQERetrieverProvider(
		channelID, mocks.NewQueryExecutorProvider().WithMockQueryExecutor(
			mocks.NewQueryExecutor().WithKVIteratorProvider(
				func(kvs []*statedb.VersionedKV) *mocks.KVIterator {
					return mocks.NewKVIterator(kvs).WithError(errExpected)
				},
			),
		),
	)

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
