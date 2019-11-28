/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package storeprovider

import (
	"testing"

	pb "github.com/hyperledger/fabric-protos-go/peer"
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	spmocks "github.com/trustbloc/fabric-peer-ext/pkg/collections/storeprovider/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

func TestStoreProvider(t *testing.T) {
	tdataProvider := spmocks.NewTransientDataStoreProvider()
	olProvider := NewOffLedgerProvider(&mocks.IdentifierProvider{}, &mocks.IdentityDeserializerProvider{})

	t.Run("OpenStore - success", func(t *testing.T) {
		p := New().Initialize(tdataProvider, olProvider)
		require.NotNil(t, p)

		s, err := p.OpenStore("testchannel")
		require.NoError(t, err)
		require.NotNil(t, s)

		s2 := p.StoreForChannel("testchannel")
		require.Equal(t, s, s2)
	})

	t.Run("OpenStore - transient data error", func(t *testing.T) {
		p := New().Initialize(tdataProvider, olProvider)
		require.NotNil(t, p)

		expectedErr := errors.New("transientdata error")
		tdataProvider.Error(expectedErr)
		defer tdataProvider.Error(nil)

		s, err := p.OpenStore("testchannel")
		assert.EqualError(t, err, expectedErr.Error())
		require.Nil(t, s)
	})

	t.Run("Close - success", func(t *testing.T) {
		p := New().Initialize(tdataProvider, olProvider)
		require.NotNil(t, p)

		s, err := p.OpenStore("testchannel")
		require.NoError(t, err)
		require.NotNil(t, s)

		assert.False(t, tdataProvider.IsStoreClosed())

		p.Close()

		assert.True(t, tdataProvider.IsStoreClosed())
	})
}

func TestStore_PutAndGetData(t *testing.T) {
	const (
		tx1   = "tx1"
		ns1   = "ns1"
		coll1 = "coll1"
		coll2 = "coll2"
		key1  = "key1"
		key2  = "key2"
	)

	k1 := storeapi.NewKey(tx1, ns1, coll1, key1)
	k2 := storeapi.NewKey(tx1, ns1, coll1, key2)
	k3 := storeapi.NewKey(tx1, ns1, coll2, key1)

	v1 := &storeapi.ExpiringValue{Value: []byte("value1")}
	v2 := &storeapi.ExpiringValue{Value: []byte("value1")}

	tdataProvider := spmocks.NewTransientDataStoreProvider()
	olProvider := spmocks.NewOffLedgerStoreProvider()

	p := New().Initialize(tdataProvider.Data(k1, v1).Data(k2, v2), olProvider.Data(k1, v2).Data(k2, v1))
	require.NotNil(t, p)

	s, err := p.OpenStore("testchannel")
	require.NoError(t, err)
	require.NotNil(t, s)

	t.Run("GetTransientData", func(t *testing.T) {
		value, err := s.GetTransientData(k1)
		require.NoError(t, err)
		require.NotNil(t, value)

		values, err := s.GetTransientDataMultipleKeys(storeapi.NewMultiKey(tx1, ns1, coll1, key1, key2))
		require.NoError(t, err)
		assert.Equal(t, 2, len(values))
	})

	t.Run("GetData", func(t *testing.T) {
		value, err := s.GetData(k1)
		require.NoError(t, err)
		require.NotNil(t, value)

		values, err := s.GetDataMultipleKeys(storeapi.NewMultiKey(tx1, ns1, coll1, key1, key2))
		require.NoError(t, err)
		assert.Equal(t, 2, len(values))
	})

	t.Run("PutData", func(t *testing.T) {
		collConfig := &pb.StaticCollectionConfig{
			Type: pb.CollectionType_COL_DCAS,
			Name: coll2,
		}
		err := s.PutData(collConfig, k3, v1)
		require.NoError(t, err)
	})

	t.Run("Persist", func(t *testing.T) {
		isCommitter = func() bool { return true }

		err := s.Persist(tx1, mocks.NewPvtReadWriteSetBuilder().Build())
		assert.NoError(t, err)

		expectedErr := errors.New("transient data error")
		tdataProvider.StoreError(expectedErr)
		err = s.Persist(tx1, mocks.NewPvtReadWriteSetBuilder().Build())
		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
		tdataProvider.StoreError(nil)

		expectedErr = errors.New("DCAS error")
		olProvider.StoreError(expectedErr)
		err = s.Persist(tx1, mocks.NewPvtReadWriteSetBuilder().Build())
		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
		olProvider.StoreError(nil)
	})
}

func TestStore_ExecuteQuery(t *testing.T) {
	const (
		tx1   = "tx1"
		ns1   = "ns1"
		coll1 = "coll1"
		key1  = "key1"
		key2  = "key2"
	)

	const query = `some query`

	v1 := []byte("")
	v2 := []byte("")
	results := []*storeapi.QueryResult{
		{
			Key: storeapi.NewKey(tx1, ns1, coll1, key1),
			ExpiringValue: &storeapi.ExpiringValue{
				Value: v1,
			},
		},
		{
			Key: storeapi.NewKey(tx1, ns1, coll1, key2),
			ExpiringValue: &storeapi.ExpiringValue{
				Value: v2,
			},
		},
	}

	p := New().Initialize(spmocks.NewTransientDataStoreProvider(), spmocks.NewOffLedgerStoreProvider().WithQueryResults(storeapi.NewQueryKey(tx1, ns1, coll1, query), results))
	require.NotNil(t, p)

	s, err := p.OpenStore("testchannel")
	require.NoError(t, err)
	require.NotNil(t, s)

	it, err := s.Query(storeapi.NewQueryKey(tx1, ns1, coll1, query))
	require.NoError(t, err)
	require.NotNil(t, it)

	next, err := it.Next()
	require.NoError(t, err)
	require.NotNil(t, next)
	require.Equal(t, key1, next.Key.Key)
	require.Equal(t, v1, next.Value)

	next, err = it.Next()
	require.NoError(t, err)
	require.NotNil(t, next)
	require.Equal(t, key2, next.Key.Key)
	require.Equal(t, v2, next.Value)

	next, err = it.Next()
	require.NoError(t, err)
	require.Nil(t, next)
}
