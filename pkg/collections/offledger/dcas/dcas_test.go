/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dcas

import (
	"testing"

	"github.com/btcsuite/btcutil/base58"
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	ns1   = "chaincode1"
	coll1 = "coll1"
	txID1 = "txid1"
)

func TestValidator(t *testing.T) {
	value := []byte("value1")

	t.Run("Valid key/value -> success", func(t *testing.T) {
		err := Validator("", "", "", base58.Encode(getCASKey(value)), value)
		assert.NoError(t, err)
	})

	t.Run("Invalid key -> error", func(t *testing.T) {
		err := Validator("", "", "", "key1", value)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "the key should be the hash of the value")
	})

	t.Run("Nil value -> error", func(t *testing.T) {
		err := Validator("", "", "", "key1", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil value for key")
	})
}

func TestDecorator_BeforeSave(t *testing.T) {
	value1_1 := []byte("value1_1")
	value := &storeapi.ExpiringValue{
		Value: value1_1,
	}

	t.Run("CAS key -> success", func(t *testing.T) {
		key := storeapi.NewKey(txID1, ns1, coll1, base58.Encode(getCASKey(value1_1)))
		k, v, err := Decorator.BeforeSave(key, value)
		require.NoError(t, err)
		assert.Equal(t, key.Key, k.Key)
		assert.Equal(t, value, v)
	})

	t.Run("Empty key -> fail", func(t *testing.T) {
		key := storeapi.NewKey(txID1, ns1, coll1, "")
		_, _, err := Decorator.BeforeSave(key, value)
		require.Error(t, err)
	})

	t.Run("Invalid key -> error", func(t *testing.T) {
		key := storeapi.NewKey(txID1, ns1, coll1, "key1")
		k, v, err := Decorator.BeforeSave(key, value)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "the key should be the hash of the value")
		assert.Nil(t, k)
		assert.Nil(t, v)
	})

	t.Run("Nil value -> error", func(t *testing.T) {
		k, v, err := Decorator.BeforeSave(storeapi.NewKey(txID1, ns1, coll1, "key1"), &storeapi.ExpiringValue{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil value for key")
		assert.Nil(t, k)
		assert.Nil(t, v)
	})
}

func TestDecorator_BeforeLoad(t *testing.T) {
	value1_1 := []byte("value1_1")

	t.Run("CAS key -> success", func(t *testing.T) {
		key := storeapi.NewKey(txID1, ns1, coll1, base58.Encode(getCASKey(value1_1)))
		k, err := Decorator.BeforeLoad(key)
		require.NoError(t, err)
		assert.Equal(t, key.Key, k.Key)
	})
}

func TestDecorator_AfterQuery(t *testing.T) {
	value1_1 := []byte("value1_1")
	value := &storeapi.ExpiringValue{
		Value: value1_1,
	}

	t.Run("CAS key -> success", func(t *testing.T) {
		key := storeapi.NewKey(txID1, ns1, coll1, base58.Encode(getCASKey(value1_1)))
		k, v, err := Decorator.AfterQuery(key, value)
		require.NoError(t, err)
		assert.Equal(t, key.Key, k.Key)
		assert.Equal(t, value, v)
	})

	t.Run("Invalid CAS key -> success", func(t *testing.T) {
		key := storeapi.NewKey(txID1, ns1, coll1, "key1")
		k, v, err := Decorator.AfterQuery(key, value)
		require.NoError(t, err)
		assert.Equal(t, key, k)
		assert.Equal(t, value, v)
	})
}
