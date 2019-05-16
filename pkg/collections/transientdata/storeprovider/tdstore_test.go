/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package storeprovider

import (
	"fmt"
	"testing"
	"time"

	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/storeprovider/store/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

const (
	channelID = "testchannel"

	txID1 = "txid1"
	txID2 = "txid2"
	txID3 = "txid3"

	ns1 = "ns1"

	coll0 = "coll0"
	coll1 = "coll1"
	coll2 = "coll2"
	coll3 = "coll3"

	key1 = "key1"
	key2 = "key2"
	key3 = "key3"
)

func TestStore(t *testing.T) {
	s := newStore(channelID, 100, newMockDB())
	require.NotNil(t, s)
	s.Close()

	// Multiple calls on Close are allowed
	assert.NotPanics(t, func() {
		s.Close()
	})

	// Calls on closed store should panic
	assert.Panics(t, func() {
		s.GetTransientData(storeapi.NewKey("txid", "ns", "coll", "key"))
	})
}

func TestStorePutAndGet(t *testing.T) {
	value1_1 := []byte("value1_1")
	value1_2 := []byte("value1_2")

	value2_1 := []byte("value2_1")
	value3_1 := []byte("value3_1")

	s := newStore(channelID, 1, newMockDB())
	require.NotNil(t, s)
	defer s.Close()

	b := mocks.NewPvtReadWriteSetBuilder()
	ns1Builder := b.Namespace(ns1)
	coll1Builder := ns1Builder.Collection(coll1)
	coll1Builder.
		TransientConfig("OR('Org1MSP.member')", 1, 2, "1m").
		Write(key1, value1_1).
		Write(key2, value1_2)
	coll2Builder := ns1Builder.Collection(coll2)
	coll2Builder.
		TransientConfig("OR('Org1MSP.member')", 1, 2, "1m").
		Write(key1, value2_1).
		Delete(key2)
	coll3Builder := ns1Builder.Collection(coll3)
	coll3Builder.
		StaticConfig("OR('Org1MSP.member')", 1, 2, 100).
		Write(key1, value3_1)

	err := s.Persist(txID1, b.Build())
	assert.NoError(t, err)

	t.Run("GetTransientData invalid collection -> nil", func(t *testing.T) {
		value, err := s.GetTransientData(storeapi.NewKey(txID1, ns1, coll0, key1))
		assert.NoError(t, err)
		assert.Nil(t, value)
	})

	t.Run("GetTransientData in same transaction -> nil", func(t *testing.T) {
		value, err := s.GetTransientData(storeapi.NewKey(txID1, ns1, coll1, key1))
		assert.NoError(t, err)
		assert.Nil(t, value)
	})

	t.Run("GetTransientData in new transaction -> valid", func(t *testing.T) {
		value, err := s.GetTransientData(storeapi.NewKey(txID2, ns1, coll1, key1))
		assert.NoError(t, err)
		assert.Equal(t, value1_1, value.Value)

		value, err = s.GetTransientData(storeapi.NewKey(txID2, ns1, coll1, key2))
		assert.NoError(t, err)
		assert.Equal(t, value1_2, value.Value)
	})

	t.Run("GetTransientData collection2 -> valid", func(t *testing.T) {
		// Collection2
		value, err := s.GetTransientData(storeapi.NewKey(txID2, ns1, coll2, key1))
		assert.NoError(t, err)
		assert.Equal(t, value2_1, value.Value)
	})

	t.Run("GetTransientData on non-transient collection -> nil", func(t *testing.T) {
		value, err := s.GetTransientData(storeapi.NewKey(txID2, ns1, coll3, key1))
		assert.NoError(t, err)
		assert.Nil(t, value)
	})

	t.Run("GetTransientDataMultipleKeys -> valid", func(t *testing.T) {
		values, err := s.GetTransientDataMultipleKeys(storeapi.NewMultiKey(txID2, ns1, coll1, key1, key2))
		assert.NoError(t, err)
		require.Equal(t, 2, len(values))
		assert.Equal(t, value1_1, values[0].Value)
		assert.Equal(t, value1_2, values[1].Value)
	})

	t.Run("Disallow update transient data", func(t *testing.T) {
		b := mocks.NewPvtReadWriteSetBuilder()
		ns1Builder := b.Namespace(ns1)
		coll1Builder := ns1Builder.Collection(coll1)
		coll1Builder.
			TransientConfig("OR('Org1MSP.member')", 1, 2, "1m").
			Write(key1, value2_1)

		err = s.Persist(txID2, b.Build())
		assert.NoError(t, err)

		value, err := s.GetTransientData(storeapi.NewKey(txID3, ns1, coll1, key1))
		assert.NoError(t, err)
		assert.Equalf(t, value1_1, value.Value, "expecting transient data to not have been updated")
	})

	t.Run("Expire transient data", func(t *testing.T) {
		b := mocks.NewPvtReadWriteSetBuilder()
		ns1Builder := b.Namespace(ns1)
		coll1Builder := ns1Builder.Collection(coll1)
		coll1Builder.
			TransientConfig("OR('Org1MSP.member')", 1, 2, "10ms").
			Write(key3, value1_1)

		err = s.Persist(txID2, b.Build())
		assert.NoError(t, err)

		value, err := s.GetTransientData(storeapi.NewKey(txID3, ns1, coll1, key3))
		assert.NoError(t, err)
		assert.Equal(t, value1_1, value.Value)

		time.Sleep(15 * time.Millisecond)

		value, err = s.GetTransientData(storeapi.NewKey(txID3, ns1, coll1, key3))
		assert.NoError(t, err)
		assert.Nilf(t, value, "expecting key to have expired")
	})
}

func TestStoreInvalidRWSet(t *testing.T) {
	s := newStore(channelID, 100, newMockDB())
	require.NotNil(t, s)
	defer s.Close()

	b := mocks.NewPvtReadWriteSetBuilder()
	b.Namespace(ns1).
		Collection(coll1).
		TransientConfig("OR('Org1MSP.member')", 1, 2, "1m").
		Write(key1, []byte("value")).
		WithMarshalError()

	err := s.Persist(txID1, b.Build())
	assert.Errorf(t, err, "expecting marshal error")
}

type mockDB struct {
	err  error
	data map[api.Key]*api.Value
}

func newMockDB() *mockDB {
	return &mockDB{
		data: make(map[api.Key]*api.Value),
	}
}

func (db *mockDB) AddKey(key api.Key, value *api.Value) error {
	fmt.Printf("DB Store - Adding key [%s]\n", key)
	db.data[key] = value
	return db.err
}

func (db *mockDB) DeleteExpiredKeys() error {
	return db.err
}

func (db *mockDB) GetKey(key api.Key) (*api.Value, error) {
	fmt.Printf("DB Store - Getting key [%s]\n", key)
	return db.data[key], db.err
}
