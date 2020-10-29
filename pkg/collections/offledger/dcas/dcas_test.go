/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dcas

import (
	"testing"

	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
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
		key, err := GetCASKey(value, CIDV1, cid.Raw, mh.SHA2_256)
		require.NoError(t, err)

		require.NoError(t, Validator("", "", "", dsPrefix+key, value))
	})

	t.Run("Invalid key -> error", func(t *testing.T) {
		err := Validator("", "", "", "key1", value)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid CAS key")
	})

	t.Run("Nil value -> error", func(t *testing.T) {
		err := Validator("", "", "", "key1", nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "nil value for key")
	})
}

func TestDecorator_BeforeSave(t *testing.T) {
	value1_1 := []byte("value1_1")
	value := &storeapi.ExpiringValue{
		Value: value1_1,
	}

	t.Run("CAS key -> success", func(t *testing.T) {
		dsk, err := GetCASKey(value1_1, CIDV1, cid.Raw, mh.SHA2_256)
		require.NoError(t, err)

		key := storeapi.NewKey(txID1, ns1, coll1, dsk)
		k, v, err := Decorator.BeforeSave(key, value)
		require.NoError(t, err)
		require.Equal(t, key.Key, k.Key)
		require.Equal(t, value, v)
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
		require.Contains(t, err.Error(), "invalid CAS key")
		require.Nil(t, k)
		require.Nil(t, v)
	})

	t.Run("Nil value -> error", func(t *testing.T) {
		k, v, err := Decorator.BeforeSave(storeapi.NewKey(txID1, ns1, coll1, "key1"), &storeapi.ExpiringValue{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "nil value for key")
		require.Nil(t, k)
		require.Nil(t, v)
	})
}

func TestDecorator_BeforeLoad(t *testing.T) {
	value1_1 := []byte("value1_1")

	t.Run("CAS key -> success", func(t *testing.T) {
		dsk, err := GetCASKey(value1_1, CIDV1, cid.Raw, mh.SHA2_256)
		require.NoError(t, err)

		key := storeapi.NewKey(txID1, ns1, coll1, dsk)
		k, err := Decorator.BeforeLoad(key)
		require.NoError(t, err)
		require.Equal(t, dsk, k.Key)
	})
}

func TestDecorator_AfterQuery(t *testing.T) {
	value1_1 := []byte("value1_1")
	value := &storeapi.ExpiringValue{
		Value: value1_1,
	}

	t.Run("CAS key -> success", func(t *testing.T) {
		dsk, err := GetCASKey(value1_1, CIDV1, cid.Raw, mh.SHA2_256)
		require.NoError(t, err)

		key := storeapi.NewKey(txID1, ns1, coll1, dsk)
		k, v, err := Decorator.AfterQuery(key, value)
		require.NoError(t, err)
		require.Equal(t, key.Key, k.Key)
		require.Equal(t, value, v)
	})

	t.Run("Invalid CAS key -> success", func(t *testing.T) {
		key := storeapi.NewKey(txID1, ns1, coll1, "key1")
		k, v, err := Decorator.AfterQuery(key, value)
		require.NoError(t, err)
		require.Equal(t, key, k)
		require.Equal(t, value, v)
	})
}

func TestValidateDatastoreKey(t *testing.T) {
	value := []byte("value1")

	t.Run("success", func(t *testing.T) {
		key, err := GetCASKey(value, CIDV1, cid.DagCBOR, mh.SHA2_256)
		require.NoError(t, err)
		require.NoError(t, ValidateDatastoreKey(key, value))
	})

	t.Run("Nil value -> error", func(t *testing.T) {
		err := ValidateDatastoreKey("", nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "attempt to put nil value for key")
	})

	t.Run("Invalid key -> error", func(t *testing.T) {
		err := ValidateDatastoreKey("xxx", value)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid CAS key")
	})

	t.Run("Key/value mismatch -> error", func(t *testing.T) {
		err := ValidateDatastoreKey("/blocks/AFKREIHD6UP6LUJCG5X2ZVYBZUMDFPE5FXO5LTZVRDUWZ4E5UBNGMPUHKA", value)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid data store key")
	})
}
