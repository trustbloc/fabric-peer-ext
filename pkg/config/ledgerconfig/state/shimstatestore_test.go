/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package state

import (
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/stretchr/testify/require"
)

const (
	ns1  = "ns1"
	key1 = "key1"
	key2 = "key2"
	key3 = "key3"
)

func TestShimStore_GetPutDel(t *testing.T) {
	stub := shim.NewMockStub(ns1, nil)
	stub.MockTransactionStart("tx1")
	sp := NewShimStoreProvider(stub)

	s, err := sp.GetStore()
	require.NoError(t, err)
	require.NotNil(t, s)

	r, err := sp.GetStateRetriever()
	require.NoError(t, err)
	require.Equal(t, s, r)

	defer s.Done()

	v1 := []byte("v1")

	require.NoError(t, s.PutState(ns1, key1, v1))

	v, err := s.GetState(ns1, key1)
	require.NoError(t, err)
	require.Equal(t, v1, v)

	require.NoError(t, s.DelState(ns1, key1))

	v, err = s.GetState(ns1, key1)
	require.NoError(t, err)
	require.Nil(t, v)
}

func TestShimStore_GetStateByPartialCompositeKey(t *testing.T) {
	const (
		index   = "index1"
		prefix1 = "prefix1"
		prefix2 = "prefix2"
	)

	stub := shim.NewMockStub(ns1, nil)
	stub.MockTransactionStart("tx1")
	sp := NewShimStoreProvider(stub)

	s, err := sp.GetStore()
	require.NoError(t, err)
	require.NotNil(t, s)
	defer s.Done()

	ck1, err := stub.CreateCompositeKey(index, []string{prefix1, key1})
	require.NoError(t, err)
	require.NoError(t, s.PutState(ns1, ck1, []byte("{}")))

	ck2, err := stub.CreateCompositeKey(index, []string{prefix1, key2})
	require.NoError(t, err)
	require.NoError(t, s.PutState(ns1, ck2, []byte("{}")))

	ck3, err := stub.CreateCompositeKey(index, []string{prefix2, key3})
	require.NoError(t, err)
	require.NoError(t, s.PutState(ns1, ck3, []byte("{}")))

	it, err := s.GetStateByPartialCompositeKey(ns1, index, []string{prefix1})
	require.NoError(t, err)
	require.NotNil(t, it)

	require.True(t, it.HasNext())
	kv, err := it.Next()
	require.NoError(t, err)
	require.NotNil(t, kv)

	indexName, keys, err := stub.SplitCompositeKey(kv.Key)
	require.NoError(t, err)
	require.Equal(t, index, indexName)
	require.Equal(t, 2, len(keys))
	require.Equal(t, key1, keys[1])

	require.True(t, it.HasNext())
	kv, err = it.Next()
	require.NoError(t, err)
	require.NotNil(t, kv)

	indexName, keys, err = stub.SplitCompositeKey(kv.Key)
	require.NoError(t, err)
	require.Equal(t, index, indexName)
	require.Equal(t, 2, len(keys))
	require.Equal(t, key2, keys[1])

	it, err = s.GetStateByPartialCompositeKey(ns1, index, []string{prefix2})
	require.NoError(t, err)
	require.NotNil(t, it)

	require.True(t, it.HasNext())
	kv, err = it.Next()
	require.NoError(t, err)
	require.NotNil(t, kv)

	indexName, keys, err = stub.SplitCompositeKey(kv.Key)
	require.NoError(t, err)
	require.Equal(t, index, indexName)
	require.Equal(t, 2, len(keys))
	require.Equal(t, key3, keys[1])

	require.False(t, it.HasNext())
}
