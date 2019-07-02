/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pvtdatahandler

import (
	"testing"

	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/queryresult"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

const (
	channelID = "testchannel"
	tx1       = "tx1"
	ns1       = "ns1"
	coll1     = "coll1"
	coll2     = "coll2"
	key1      = "key1"
	key2      = "key2"
)

func TestHandler_HandleGetPrivateData(t *testing.T) {
	t.Run("Unhandled collection", func(t *testing.T) {
		config := &common.StaticCollectionConfig{}

		h := New(channelID, mocks.NewDataProvider())
		require.NotNil(t, h)

		value, handled, err := h.HandleGetPrivateData(tx1, ns1, config, key1)
		assert.NoError(t, err)
		assert.False(t, handled)
		assert.Nil(t, value)
	})

	t.Run("Transient Data", func(t *testing.T) {
		testHandleGetPrivateData(t, &common.StaticCollectionConfig{
			Type: common.CollectionType_COL_TRANSIENT,
			Name: coll1,
		})
	})

	t.Run("Off-ledger Data", func(t *testing.T) {
		testHandleGetPrivateData(t, &common.StaticCollectionConfig{
			Type: common.CollectionType_COL_OFFLEDGER,
			Name: coll1,
		})
	})

	t.Run("DCAS Data", func(t *testing.T) {
		testHandleGetPrivateData(t, &common.StaticCollectionConfig{
			Type: common.CollectionType_COL_DCAS,
			Name: coll1,
		})
	})
}

func TestHandler_HandleGetPrivateDataMultipleKeys(t *testing.T) {
	t.Run("Unhandled collection", func(t *testing.T) {
		config := &common.StaticCollectionConfig{}

		dataProvider := mocks.NewDataProvider()
		h := New(channelID, dataProvider)
		require.NotNil(t, h)

		value, handled, err := h.HandleGetPrivateDataMultipleKeys(tx1, ns1, config, []string{key1, key2})
		assert.NoError(t, err)
		assert.False(t, handled)
		assert.Nil(t, value)
	})

	t.Run("Transient Data", func(t *testing.T) {
		testHandleGetPrivateDataMultipleKeys(t, &common.StaticCollectionConfig{
			Type: common.CollectionType_COL_TRANSIENT,
			Name: coll1,
		})
	})

	t.Run("Off-ledger Data", func(t *testing.T) {
		testHandleGetPrivateDataMultipleKeys(t, &common.StaticCollectionConfig{
			Type: common.CollectionType_COL_OFFLEDGER,
			Name: coll1,
		})
	})

	t.Run("DCAS Data", func(t *testing.T) {
		testHandleGetPrivateDataMultipleKeys(t, &common.StaticCollectionConfig{
			Type: common.CollectionType_COL_DCAS,
			Name: coll1,
		})
	})
}

func TestHandler_HandleExecuteQueryOnPrivateData(t *testing.T) {
	const query = "some query"

	v1 := []byte("v1")
	v2 := []byte("v2")

	olResults := []*storeapi.QueryResult{
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
	dcasResults := []*storeapi.QueryResult{
		{
			Key: storeapi.NewKey(tx1, ns1, coll2, key1),
			ExpiringValue: &storeapi.ExpiringValue{
				Value: v1,
			},
		},
		{
			Key: storeapi.NewKey(tx1, ns1, coll2, key2),
			ExpiringValue: &storeapi.ExpiringValue{
				Value: v2,
			},
		},
	}

	dataProvider := mocks.NewDataProvider().
		WithQueryResults(storeapi.NewQueryKey(tx1, ns1, coll1, query), olResults).
		WithQueryResults(storeapi.NewQueryKey(tx1, ns1, coll2, query), dcasResults)

	h := New(channelID, dataProvider)
	require.NotNil(t, h)

	t.Run("Unhandled collection", func(t *testing.T) {
		config := &common.StaticCollectionConfig{}
		value, handled, err := h.HandleExecuteQueryOnPrivateData(tx1, ns1, config, "some query")
		assert.NoError(t, err)
		assert.False(t, handled)
		assert.Nil(t, value)
	})

	t.Run("Transient Data", func(t *testing.T) {
		config := &common.StaticCollectionConfig{
			Type: common.CollectionType_COL_TRANSIENT,
			Name: coll1,
		}

		_, handled, err := h.HandleExecuteQueryOnPrivateData(tx1, ns1, config, query)
		require.Error(t, err)
		require.True(t, handled)
	})

	t.Run("Off-ledger Data", func(t *testing.T) {
		config := &common.StaticCollectionConfig{
			Type: common.CollectionType_COL_OFFLEDGER,
			Name: coll1,
		}

		it, handled, err := h.HandleExecuteQueryOnPrivateData(tx1, ns1, config, "some query")
		require.NoError(t, err)
		require.True(t, handled)
		require.NotNil(t, it)

		next, err := it.Next()
		require.NoError(t, err)
		require.NotNil(t, next)
		kv, ok := next.(*queryresult.KV)
		require.True(t, ok)
		require.Equal(t, asPvtDataNs(ns1, coll1), kv.Namespace)
		require.Equal(t, key1, kv.Key)
		require.Equal(t, v1, kv.Value)

		next, err = it.Next()
		require.NoError(t, err)
		require.NotNil(t, next)
		kv, ok = next.(*queryresult.KV)
		require.True(t, ok)
		require.Equal(t, asPvtDataNs(ns1, coll1), kv.Namespace)
		require.Equal(t, key2, kv.Key)
		require.Equal(t, v2, kv.Value)

		next, err = it.Next()
		require.NoError(t, err)
		require.Nil(t, next)
	})

	t.Run("DCAS Data", func(t *testing.T) {
		config := &common.StaticCollectionConfig{
			Type: common.CollectionType_COL_DCAS,
			Name: coll2,
		}

		it, handled, err := h.HandleExecuteQueryOnPrivateData(tx1, ns1, config, "some query")
		require.NoError(t, err)
		require.True(t, handled)
		require.NotNil(t, it)

		next, err := it.Next()
		require.NoError(t, err)
		require.NotNil(t, next)
		kv, ok := next.(*queryresult.KV)
		require.True(t, ok)
		require.Equal(t, asPvtDataNs(ns1, coll2), kv.Namespace)
		require.Equal(t, key1, kv.Key)
		require.Equal(t, v1, kv.Value)

		next, err = it.Next()
		require.NoError(t, err)
		require.NotNil(t, next)
		kv, ok = next.(*queryresult.KV)
		require.True(t, ok)
		require.Equal(t, asPvtDataNs(ns1, coll2), kv.Namespace)
		require.Equal(t, key2, kv.Key)
		require.Equal(t, v2, kv.Value)

		next, err = it.Next()
		require.NoError(t, err)
		require.Nil(t, next)
	})
}

func testHandleGetPrivateData(t *testing.T, config *common.StaticCollectionConfig) {
	dataProvider := mocks.NewDataProvider()
	h := New(channelID, dataProvider)
	require.NotNil(t, h)

	value, handled, err := h.HandleGetPrivateData(tx1, ns1, config, key1)
	assert.NoError(t, err)
	assert.True(t, handled)
	assert.Nil(t, value)

	key := storeapi.NewKey(tx1, ns1, coll1, key1)
	exValue := &storeapi.ExpiringValue{Value: []byte("value1")}
	dataProvider.WithData(key, exValue)

	value, handled, err = h.HandleGetPrivateData(tx1, ns1, config, key1)
	assert.NoError(t, err)
	assert.True(t, handled)
	assert.Equal(t, exValue.Value, value)

	expectedErr := errors.New("data provider error")
	dataProvider.WithError(expectedErr)

	value, handled, err = h.HandleGetPrivateData(tx1, ns1, config, key1)
	assert.Error(t, err)
	assert.True(t, handled)
	assert.Nil(t, value)
}

func testHandleGetPrivateDataMultipleKeys(t *testing.T, config *common.StaticCollectionConfig) {
	dataProvider := mocks.NewDataProvider()
	h := New(channelID, dataProvider)
	require.NotNil(t, h)

	values, handled, err := h.HandleGetPrivateDataMultipleKeys(tx1, ns1, config, []string{key1, key2})
	assert.NoError(t, err)
	assert.True(t, handled)
	require.Equal(t, 2, len(values))
	assert.Nil(t, values[0])
	assert.Nil(t, values[1])

	k1 := storeapi.NewKey(tx1, ns1, coll1, key1)
	k2 := storeapi.NewKey(tx1, ns1, coll1, key2)
	exValue1 := &storeapi.ExpiringValue{Value: []byte("value1")}
	exValue2 := &storeapi.ExpiringValue{Value: []byte("value2")}
	dataProvider.WithData(k1, exValue1).WithData(k2, exValue2)

	values, handled, err = h.HandleGetPrivateDataMultipleKeys(tx1, ns1, config, []string{key1, key2})
	assert.NoError(t, err)
	assert.True(t, handled)
	require.Equal(t, 2, len(values))
	assert.Equal(t, exValue1.Value, values[0])
	assert.Equal(t, exValue2.Value, values[1])

	expectedErr := errors.New("data provider error")
	dataProvider.WithError(expectedErr)

	values, handled, err = h.HandleGetPrivateDataMultipleKeys(tx1, ns1, config, []string{key1, key2})
	assert.Error(t, err)
	assert.True(t, handled)
	assert.Nil(t, values)
}
