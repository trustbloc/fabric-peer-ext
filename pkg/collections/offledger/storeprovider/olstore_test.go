/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package storeprovider

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	pb "github.com/hyperledger/fabric-protos-go/peer"
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	viper "github.com/spf13/viper2015"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	collcommon "github.com/trustbloc/fabric-peer-ext/pkg/collections/common"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/dcas"
	olstoreapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/storeprovider/store/api"
	olmocks "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/storeprovider/store/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

const (
	channelID = "testchannel"

	txID1 = "txid1"
	txID2 = "txid2"
	txID3 = "txid3"
	txID4 = "txid4"

	ns1 = "ns1"
	ns2 = "ns2"

	coll0 = "coll0"
	coll1 = "coll1"
	coll2 = "coll2"
	coll3 = "coll3"

	org1MSP = "Org1MSP"
	org2MSP = "Org2MSP"

	key1 = "key1"
	key2 = "key2"
	key3 = "key3"
)

var (
	value1_1 = []byte("value1_1")
	value1_2 = []byte("value1_2")
	value2_1 = []byte("value2_1")
	value3_1 = []byte("value3_1")
	value4_1 = []byte("value4_1")

	typeConfig = map[pb.CollectionType]*collTypeConfig{
		pb.CollectionType_COL_OFFLEDGER: {},
		pb.CollectionType_COL_DCAS:      {decorator: dcas.Decorator},
	}

	// Ensure roles are initialized
	_ = roles.GetRoles()
)

func TestStore_Close(t *testing.T) {
	s := newStore(channelID, olmocks.NewDBProvider(), &mocks.IdentifierProvider{}, &mocks.IdentityDeserializer{}, typeConfig)
	require.NotNil(t, s)

	s.Close()

	// Multiple calls on Close are allowed
	assert.NotPanics(t, func() {
		s.Close()
	})
}

func TestStore_PutAndGet(t *testing.T) {
	s := newStore(channelID, olmocks.NewDBProvider(), &mocks.IdentifierProvider{}, &mocks.IdentityDeserializer{}, typeConfig)
	require.NotNil(t, s)
	defer s.Close()

	getLocalMSPID = func(collcommon.IdentifierProvider) (string, error) { return org1MSP, nil }

	b := mocks.NewPvtReadWriteSetBuilder()
	ns1Builder := b.Namespace(ns1)
	coll1Builder := ns1Builder.Collection(coll1)
	coll1Builder.
		OffLedgerConfig("OR('Org1MSP.member')", 1, 2, "1m").
		Write(key1, value1_1).
		Write(key2, value1_2)
	coll2Builder := ns1Builder.Collection(coll2)
	coll2Builder.
		OffLedgerConfig("OR('Org1MSP.member')", 1, 2, "").
		Write(key2, value2_1)
	coll3Builder := ns1Builder.Collection(coll3)
	coll3Builder.
		StaticConfig("OR('Org1MSP.member')", 1, 2, 100).
		Write(key1, value3_1)

	err := s.Persist(txID1, b.Build())
	assert.NoError(t, err)

	t.Run("GetData invalid collection -> nil", func(t *testing.T) {
		value, err := s.GetData(storeapi.NewKey(txID1, ns1, coll0, key1))
		assert.NoError(t, err)
		assert.Nil(t, value)
	})

	t.Run("GetData in same transaction -> nil", func(t *testing.T) {
		value, err := s.GetData(storeapi.NewKey(txID1, ns1, coll1, key1))
		assert.NoError(t, err)
		assert.Nil(t, value)
	})

	t.Run("GetData in new transaction -> valid", func(t *testing.T) {
		value, err := s.GetData(storeapi.NewKey(txID2, ns1, coll1, key1))
		assert.NoError(t, err)
		require.NotNil(t, value)
		assert.Equal(t, value1_1, value.Value)

		value, err = s.GetData(storeapi.NewKey(txID2, ns1, coll1, key2))
		assert.NoError(t, err)
		require.NotNil(t, value)
		assert.Equal(t, value1_2, value.Value)
	})

	t.Run("GetData collection2 -> valid", func(t *testing.T) {
		// Collection2
		value, err := s.GetData(storeapi.NewKey(txID2, ns1, coll2, key2))
		assert.NoError(t, err)
		require.NotNil(t, value)
		assert.Equal(t, value2_1, value.Value)
	})

	t.Run("GetData on non-off-ledger collection -> nil", func(t *testing.T) {
		value, err := s.GetData(storeapi.NewKey(txID2, ns1, coll3, key1))
		assert.NoError(t, err)
		assert.Nil(t, value)
	})

	t.Run("GetDataMultipleKeys in same transaction -> nil", func(t *testing.T) {
		values, err := s.GetDataMultipleKeys(storeapi.NewMultiKey(txID1, ns1, coll1, key1, key2))
		assert.NoError(t, err)
		require.Equal(t, 2, len(values))
		assert.Nil(t, values[0])
		assert.Nil(t, values[1])
	})

	t.Run("GetDataMultipleKeys in new transaction -> valid", func(t *testing.T) {
		values, err := s.GetDataMultipleKeys(storeapi.NewMultiKey(txID2, ns1, coll1, key1, key2))
		assert.NoError(t, err)
		require.Equal(t, 2, len(values))
		require.NotNil(t, values[0])
		require.NotNil(t, values[1])
		assert.Equal(t, value1_1, values[0].Value)
		assert.Equal(t, value1_2, values[1].Value)
	})

	t.Run("Delete data", func(t *testing.T) {
		value, err := s.GetData(storeapi.NewKey(txID4, ns1, coll1, key1))
		assert.NoError(t, err)
		assert.NotNil(t, value)

		b := mocks.NewPvtReadWriteSetBuilder()
		ns1Builder := b.Namespace(ns1)
		coll1Builder := ns1Builder.Collection(coll1)
		coll1Builder.
			OffLedgerConfig("OR('Org1MSP.member')", 1, 2, "1m").
			Delete(key1)
		err = s.Persist(txID3, b.Build())

		value, err = s.GetData(storeapi.NewKey(txID4, ns1, coll1, key1))
		assert.NoError(t, err)
		assert.Nil(t, value)
	})

	t.Run("Expire data", func(t *testing.T) {
		b := mocks.NewPvtReadWriteSetBuilder()
		ns1Builder := b.Namespace(ns1)
		coll1Builder := ns1Builder.Collection(coll1)
		coll1Builder.
			OffLedgerConfig("OR('Org1MSP.member')", 1, 2, "10ms").
			Write(key3, value3_1)

		err = s.Persist(txID2, b.Build())
		assert.NoError(t, err)

		value, err := s.GetData(storeapi.NewKey(txID3, ns1, coll1, key3))
		assert.NoError(t, err)
		require.NotNil(t, value)
		assert.Equal(t, value3_1, value.Value)

		time.Sleep(200 * time.Millisecond)

		value, err = s.GetData(storeapi.NewKey(txID3, ns1, coll1, key3))
		assert.NoError(t, err)
		assert.Nilf(t, value, "expecting key to have expired")
	})
}

func TestStore_PutAndGet_DCAS(t *testing.T) {
	s := newStore(channelID, olmocks.NewDBProvider(), &mocks.IdentifierProvider{}, &mocks.IdentityDeserializer{}, typeConfig)
	require.NotNil(t, s)
	defer s.Close()

	getLocalMSPID = func(collcommon.IdentifierProvider) (string, error) { return org1MSP, nil }

	t.Run("Persist invalid CAS key -> fail", func(t *testing.T) {
		b := mocks.NewPvtReadWriteSetBuilder()
		ns1Builder := b.Namespace(ns1)
		coll1Builder := ns1Builder.Collection(coll1)
		coll1Builder.
			DCASConfig("OR('Org1MSP.member')", 1, 2, "1m").
			Write(key1, value1_1)
		err := s.Persist(txID1, b.Build())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "the key should be the hash of the value")
	})

	t.Run("GetData -> success", func(t *testing.T) {
		casKey1, value1, err := dcas.GetCASKeyAndValueBase58(value1_1)
		require.NoError(t, err)

		casKey2, value2, err := dcas.GetCASKeyAndValueBase58(value1_2)
		require.NoError(t, err)

		b := mocks.NewPvtReadWriteSetBuilder()
		ns1Builder := b.Namespace(ns1)
		coll1Builder := ns1Builder.Collection(coll1)
		coll1Builder.
			DCASConfig("OR('Org1MSP.member')", 1, 2, "1m").
			Write(casKey1, value1).
			Write(casKey2, value2)

		require.NoError(t, s.Persist(txID1, b.Build()))

		value, err := s.GetData(storeapi.NewKey(txID2, ns1, coll1, casKey1))
		assert.NoError(t, err)
		require.NotNil(t, value)
		assert.Equal(t, value1_1, value.Value)

		value, err = s.GetData(storeapi.NewKey(txID2, ns1, coll1, casKey2))
		assert.NoError(t, err)
		require.NotNil(t, value)
		assert.Equal(t, value1_2, value.Value)
	})

	t.Run("Delete data", func(t *testing.T) {
		casKey1, value1, err := dcas.GetCASKeyAndValueBase58(value1_1)
		require.NoError(t, err)

		b := mocks.NewPvtReadWriteSetBuilder()
		ns1Builder := b.Namespace(ns2)
		coll1Builder := ns1Builder.Collection(coll1)
		coll1Builder.
			DCASConfig("OR('Org1MSP.member')", 1, 2, "1m").
			Write(casKey1, value1)

		require.NoError(t, s.Persist(txID1, b.Build()))

		value, err := s.GetData(storeapi.NewKey(txID2, ns2, coll1, casKey1))
		assert.NoError(t, err)
		assert.NotNil(t, value)

		b = mocks.NewPvtReadWriteSetBuilder()
		ns1Builder = b.Namespace(ns2)
		coll1Builder = ns1Builder.Collection(coll1)
		coll1Builder.
			DCASConfig("OR('Org1MSP.member')", 1, 2, "1m").
			Delete(casKey1)
		err = s.Persist(txID3, b.Build())

		value, err = s.GetData(storeapi.NewKey(txID4, ns2, coll1, casKey1))
		assert.NoError(t, err)
		assert.Nil(t, value)
	})
}

func TestStore_LoadFromDB(t *testing.T) {
	getLocalMSPID = func(collcommon.IdentifierProvider) (string, error) { return org1MSP, nil }

	dbProvider := olmocks.NewDBProvider().
		WithValue(ns1, coll1, key1, &olstoreapi.Value{Value: value1_1})

	s := newStore(channelID, dbProvider, &mocks.IdentifierProvider{}, &mocks.IdentityDeserializer{}, typeConfig)
	require.NotNil(t, s)
	defer s.Close()

	t.Run("GetData -> success", func(t *testing.T) {
		value, err := s.GetData(storeapi.NewKey(txID2, ns1, coll1, key1))
		assert.NoError(t, err)
		assert.Equal(t, value1_1, value.Value)
	})
}

func TestStore_PersistError(t *testing.T) {
	getLocalMSPID = func(collcommon.IdentifierProvider) (string, error) { return org1MSP, nil }

	s := newStore(channelID, olmocks.NewDBProvider(), &mocks.IdentifierProvider{}, &mocks.IdentityDeserializer{}, typeConfig)
	require.NotNil(t, s)

	defer s.Close()

	t.Run("Persist marshal error -> error", func(t *testing.T) {
		b := mocks.NewPvtReadWriteSetBuilder()
		b.Namespace(ns1).
			Collection(coll1).
			OffLedgerConfig("OR('Org1MSP.member')", 1, 2, "1m").
			Write(key1, value1_1).
			WithMarshalError()

		err := s.Persist(txID1, b.Build())
		assert.Errorf(t, err, "expecting marshal error")
	})

	t.Run("Persist invalid duration -> error", func(t *testing.T) {
		b := mocks.NewPvtReadWriteSetBuilder()
		ns1Builder := b.Namespace(ns1)
		coll1Builder := ns1Builder.Collection(coll1)
		coll1Builder.
			OffLedgerConfig("OR('Org1MSP.member')", 1, 2, "xxxxx")

		err := s.Persist(txID1, b.Build())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid duration")
	})
}

func TestStore_PutData(t *testing.T) {
	getLocalMSPID = func(collcommon.IdentifierProvider) (string, error) { return org1MSP, nil }

	s := newStore(channelID, olmocks.NewDBProvider(), &mocks.IdentifierProvider{}, &mocks.IdentityDeserializer{}, typeConfig)
	require.NotNil(t, s)

	defer s.Close()

	collConfig := &pb.StaticCollectionConfig{
		Type: pb.CollectionType_COL_OFFLEDGER,
		Name: coll1,
	}

	t.Run("Valid key -> success", func(t *testing.T) {
		err := s.PutData(
			collConfig,
			&storeapi.Key{
				EndorsedAtTxID: txID1,
				Namespace:      ns1,
				Collection:     coll1,
				Key:            key1,
			},
			&storeapi.ExpiringValue{
				Value: value1_1,
			},
		)
		assert.NoError(t, err)

		v, err := s.GetData(&storeapi.Key{EndorsedAtTxID: txID2, Namespace: ns1, Collection: coll1, Key: key1})
		assert.NoError(t, err)
		require.NotNil(t, v)
		assert.Equal(t, value1_1, v.Value)
	})

	t.Run("Nil value -> error", func(t *testing.T) {
		err := s.PutData(
			collConfig,
			&storeapi.Key{
				EndorsedAtTxID: txID1,
				Namespace:      ns1,
				Collection:     coll1,
			},
			&storeapi.ExpiringValue{},
		)
		assert.Error(t, err)
	})

	t.Run("Already expired -> not added", func(t *testing.T) {
		err := s.PutData(
			collConfig,
			&storeapi.Key{
				EndorsedAtTxID: txID1,
				Namespace:      ns1,
				Collection:     coll1,
				Key:            key3,
			},
			&storeapi.ExpiringValue{
				Value:  value4_1,
				Expiry: time.Now().Add(-1 * time.Second),
			},
		)
		assert.NoError(t, err)

		v, err := s.GetData(&storeapi.Key{EndorsedAtTxID: txID2, Namespace: ns1, Collection: coll1, Key: key3})
		assert.NoError(t, err)
		require.Nil(t, v)
	})

	t.Run("Invalid CAS key -> fail", func(t *testing.T) {
		dcasCollConfig := &pb.StaticCollectionConfig{
			Type: pb.CollectionType_COL_DCAS,
			Name: coll1,
		}
		err := s.PutData(
			dcasCollConfig,
			&storeapi.Key{
				EndorsedAtTxID: txID1,
				Namespace:      ns1,
				Collection:     coll1,
				Key:            key1,
			},
			&storeapi.ExpiringValue{
				Value: value1_1,
			},
		)
		assert.Error(t, err)

		v, err := s.GetData(&storeapi.Key{EndorsedAtTxID: txID2, Namespace: ns1, Collection: coll1, Key: key1})
		assert.NoError(t, err)
		require.NotNil(t, v)
		assert.Equal(t, value1_1, v.Value)
	})
}

func TestStore_DBError(t *testing.T) {
	getLocalMSPID = func(collcommon.IdentifierProvider) (string, error) { return org1MSP, nil }

	dbProvider := olmocks.NewDBProvider()
	s := newStore(channelID, dbProvider, &mocks.IdentifierProvider{}, &mocks.IdentityDeserializer{}, typeConfig)
	require.NotNil(t, s)

	t.Run("GetData -> error", func(t *testing.T) {
		key := storeapi.NewKey(txID1, ns1, coll1, key1)

		expectedErr := fmt.Errorf("error getting DB")
		dbProvider.WithError(expectedErr)
		v, err := s.GetData(key)
		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
		assert.Nil(t, v)

		expectedErr = fmt.Errorf("error getting value")
		dbProvider.WithError(nil).MockDB(ns1, coll1).WithError(expectedErr)
		v, err = s.GetData(key)
		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
		assert.Nil(t, v)
	})

	t.Run("GetDataMultipleKeys -> error", func(t *testing.T) {
		key := storeapi.NewMultiKey(txID1, ns1, coll1, key1)

		expectedErr := fmt.Errorf("error getting DB")
		dbProvider.WithError(expectedErr)
		v, err := s.GetDataMultipleKeys(key)
		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
		assert.Nil(t, v)

		expectedErr = fmt.Errorf("error getting value")
		dbProvider.WithError(nil).MockDB(ns1, coll1).WithError(expectedErr)
		v, err = s.GetDataMultipleKeys(key)
		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
		assert.Nil(t, v)
	})

	t.Run("Persist -> error", func(t *testing.T) {
		expectedErr := fmt.Errorf("error getting DB")
		dbProvider.WithError(expectedErr)

		b := mocks.NewPvtReadWriteSetBuilder()
		b.Namespace(ns1).Collection(coll1).OffLedgerConfig("OR('Org1MSP.member')", 1, 2, "1m")

		err := s.Persist(txID1, b.Build())
		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())

		expectedErr = fmt.Errorf("error putting value")
		dbProvider.WithError(nil).MockDB(ns1, coll1).WithError(expectedErr)

		err = s.Persist(txID1, b.Build())
		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
	})

	t.Run("Persist -> error", func(t *testing.T) {
		expectedErr := fmt.Errorf("error getting DB")
		dbProvider.WithError(expectedErr)

		collConfig := &pb.StaticCollectionConfig{
			Type: pb.CollectionType_COL_OFFLEDGER,
			Name: coll1,
		}

		key := &storeapi.Key{EndorsedAtTxID: txID1, Namespace: ns1, Collection: coll1}
		value := &storeapi.ExpiringValue{Value: value1_1}

		err := s.PutData(collConfig, key, value)
		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())

		expectedErr = fmt.Errorf("error putting value")
		dbProvider.WithError(nil).MockDB(ns1, coll1).WithError(expectedErr)

		err = s.PutData(collConfig, key, value)
		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
	})
}

func TestStore_PersistNotAuthorized(t *testing.T) {
	getLocalMSPID = func(collcommon.IdentifierProvider) (string, error) { return org2MSP, nil }

	s := newStore(channelID, olmocks.NewDBProvider(), &mocks.IdentifierProvider{}, &mocks.IdentityDeserializer{}, typeConfig)
	require.NotNil(t, s)
	defer s.Close()

	b := mocks.NewPvtReadWriteSetBuilder()
	ns1Builder := b.Namespace(ns1)
	coll1Builder := ns1Builder.Collection(coll1)
	coll1Builder.
		OffLedgerConfig("OR('Org1MSP.member')", 1, 2, "1m").
		Write(key1, value1_1)
	coll2Builder := ns1Builder.Collection(coll2)
	coll2Builder.
		OffLedgerConfig("OR('Org1MSP.member','Org2MSP.member')", 1, 2, "").
		Write(key2, value2_1)

	err := s.Persist(txID1, b.Build())
	assert.NoError(t, err)

	// Key shouldn't have been persisted since org2 doesn't have access to collection1
	value, err := s.GetData(storeapi.NewKey(txID2, ns1, coll1, key1))
	assert.NoError(t, err)
	assert.Nil(t, value)

	// Key should have been persisted to collection2
	value, err = s.GetData(storeapi.NewKey(txID2, ns1, coll2, key2))
	assert.NoError(t, err)
	require.NotNil(t, value)
	assert.Equal(t, value2_1, value.Value)
}

func TestStore_PersistNonCommitter(t *testing.T) {
	roles.SetRoles(map[roles.Role]struct{}{roles.EndorserRole: {}})
	defer func() { roles.SetRoles(nil) }()

	getLocalMSPID = func(collcommon.IdentifierProvider) (string, error) { return org1MSP, nil }

	s := newStore(channelID, olmocks.NewDBProvider(), &mocks.IdentifierProvider{}, &mocks.IdentityDeserializer{}, typeConfig)
	require.NotNil(t, s)
	defer s.Close()

	b := mocks.NewPvtReadWriteSetBuilder()

	t.Run("MaxPeerCount = 0 -> persisted", func(t *testing.T) {
		ns1Builder := b.Namespace(ns1)
		coll1Builder := ns1Builder.Collection(coll1)
		coll1Builder.
			OffLedgerConfig("OR('Org1MSP.member')", 0, 0, "1m").
			Write(key1, value1_1)

		err := s.Persist(txID1, b.Build())
		require.NoError(t, err)

		value, err := s.GetData(storeapi.NewKey(txID2, ns1, coll1, key1))
		require.NoError(t, err)
		require.NotNil(t, value)
		require.Equal(t, value1_1, value.Value)
	})

	t.Run("MaxPeerCount = 1 -> not persisted", func(t *testing.T) {
		ns1Builder := b.Namespace(ns1)
		coll1Builder := ns1Builder.Collection(coll1)
		coll1Builder.
			OffLedgerConfig("OR('Org1MSP.member')", 0, 1, "1m").
			Write(key2, value1_1)

		err := s.Persist(txID1, b.Build())
		require.NoError(t, err)

		value, err := s.GetData(storeapi.NewKey(txID2, ns1, coll1, key2))
		require.NoError(t, err)
		require.Nil(t, value)
	})
}

type testValue struct {
	Field1 string
	Field2 int
}

func TestStore_ExecuteQuery(t *testing.T) {
	const query = "some query"
	const field2Val = 12345

	v1 := &testValue{
		Field1: "value1",
		Field2: field2Val,
	}
	v2 := &testValue{
		Field1: "value2",
		Field2: field2Val,
	}

	value1_1, err := json.Marshal(v1)
	require.NoError(t, err)
	value1_2, err := json.Marshal(v2)
	require.NoError(t, err)

	results := []*olstoreapi.KeyValue{
		{
			Key: key1,
			Value: &olstoreapi.Value{
				Value: value1_1,
				TxID:  txID1,
			},
		},
		{
			Key: key2,
			Value: &olstoreapi.Value{
				Value: value1_2,
				TxID:  txID1,
			},
		},
	}

	dbProvider := olmocks.NewDBProvider().WithQueryResults(ns1, coll1, query, results)
	s := newStore(channelID, dbProvider, &mocks.IdentifierProvider{}, &mocks.IdentityDeserializer{}, typeConfig)
	require.NotNil(t, s)
	defer s.Close()

	getLocalMSPID = func(collcommon.IdentifierProvider) (string, error) { return org1MSP, nil }

	t.Run("Query in same transaction -> empty", func(t *testing.T) {
		it, err := s.Query(storeapi.NewQueryKey(txID1, ns1, coll1, query))
		require.NoError(t, err)
		require.NotNil(t, it)

		next, err := it.Next()
		require.NoError(t, err)
		require.Nil(t, next)

		it.Close()
	})

	t.Run("Query in new transaction -> valid", func(t *testing.T) {
		it, err := s.Query(storeapi.NewQueryKey(txID2, ns1, coll1, query))
		require.NoError(t, err)
		require.NotNil(t, it)

		next, err := it.Next()
		require.NoError(t, err)
		require.NotNil(t, next)
		require.Equal(t, key1, next.Key.Key)

		next, err = it.Next()
		require.NoError(t, err)
		require.NotNil(t, next)
		require.Equal(t, key2, next.Key.Key)

		it.Close()
	})

	t.Run("DB provider error -> fail", func(t *testing.T) {
		errExpected := errors.New("injected error")
		dbProvider.WithError(errExpected)
		defer dbProvider.WithError(nil)

		it, err := s.Query(storeapi.NewQueryKey(txID1, ns1, coll1, query))
		require.EqualError(t, err, errExpected.Error())
		require.Nil(t, it)
	})

	t.Run("DB error -> fail", func(t *testing.T) {
		errExpected := errors.New("injected error")
		dbProvider.MockDB(ns1, coll1).WithError(errExpected)
		defer dbProvider.MockDB(ns1, coll1).WithError(nil)

		it, err := s.Query(storeapi.NewQueryKey(txID1, ns1, coll1, query))
		require.EqualError(t, err, errExpected.Error())
		require.Nil(t, it)
	})
}

func TestStore_PutAndGet_NoCache(t *testing.T) {

	viper.Set("coll.offledger.cache.enable", "false")
	defer viper.Set("coll.offledger.cache.enable", "true")

	s := newStore(channelID, olmocks.NewDBProvider(), &mocks.IdentifierProvider{}, &mocks.IdentityDeserializer{}, typeConfig)
	require.NotNil(t, s)
	defer s.Close()

	getLocalMSPID = func(collcommon.IdentifierProvider) (string, error) { return org1MSP, nil }

	b := mocks.NewPvtReadWriteSetBuilder()
	ns1Builder := b.Namespace(ns1)
	coll1Builder := ns1Builder.Collection(coll1)
	coll1Builder.
		OffLedgerConfig("OR('Org1MSP.member')", 1, 2, "1m").
		Write(key1, value1_1).
		Write(key2, value1_2)
	coll2Builder := ns1Builder.Collection(coll2)
	coll2Builder.
		OffLedgerConfig("OR('Org1MSP.member')", 1, 2, "").
		Write(key2, value2_1)
	coll3Builder := ns1Builder.Collection(coll3)
	coll3Builder.
		StaticConfig("OR('Org1MSP.member')", 1, 2, 100).
		Write(key1, value3_1)

	err := s.Persist(txID1, b.Build())
	assert.NoError(t, err)

	t.Run("GetData invalid collection -> nil", func(t *testing.T) {
		value, err := s.GetData(storeapi.NewKey(txID1, ns1, coll0, key1))
		assert.NoError(t, err)
		assert.Nil(t, value)
	})

	t.Run("GetData in same transaction -> nil", func(t *testing.T) {
		value, err := s.GetData(storeapi.NewKey(txID1, ns1, coll1, key1))
		assert.NoError(t, err)
		assert.Nil(t, value)
	})

	t.Run("GetData in new transaction -> valid", func(t *testing.T) {
		value, err := s.GetData(storeapi.NewKey(txID2, ns1, coll1, key1))
		assert.NoError(t, err)
		require.NotNil(t, value)
		assert.Equal(t, value1_1, value.Value)

		value, err = s.GetData(storeapi.NewKey(txID2, ns1, coll1, key2))
		assert.NoError(t, err)
		require.NotNil(t, value)
		assert.Equal(t, value1_2, value.Value)
	})

	t.Run("GetData collection2 -> valid", func(t *testing.T) {
		// Collection2
		value, err := s.GetData(storeapi.NewKey(txID2, ns1, coll2, key2))
		assert.NoError(t, err)
		require.NotNil(t, value)
		assert.Equal(t, value2_1, value.Value)
	})

	t.Run("GetData on non-off-ledger collection -> nil", func(t *testing.T) {
		value, err := s.GetData(storeapi.NewKey(txID2, ns1, coll3, key1))
		assert.NoError(t, err)
		assert.Nil(t, value)
	})

	t.Run("GetDataMultipleKeys in same transaction -> nil", func(t *testing.T) {
		values, err := s.GetDataMultipleKeys(storeapi.NewMultiKey(txID1, ns1, coll1, key1, key2))
		assert.NoError(t, err)
		require.Equal(t, 2, len(values))
		assert.Nil(t, values[0])
		assert.Nil(t, values[1])
	})

	t.Run("GetDataMultipleKeys in new transaction -> valid", func(t *testing.T) {
		values, err := s.GetDataMultipleKeys(storeapi.NewMultiKey(txID2, ns1, coll1, key1, key2))
		assert.NoError(t, err)
		require.Equal(t, 2, len(values))
		require.NotNil(t, values[0])
		require.NotNil(t, values[1])
		assert.Equal(t, value1_1, values[0].Value)
		assert.Equal(t, value1_2, values[1].Value)
	})

	t.Run("Delete data", func(t *testing.T) {
		value, err := s.GetData(storeapi.NewKey(txID4, ns1, coll1, key1))
		assert.NoError(t, err)
		assert.NotNil(t, value)

		b := mocks.NewPvtReadWriteSetBuilder()
		ns1Builder := b.Namespace(ns1)
		coll1Builder := ns1Builder.Collection(coll1)
		coll1Builder.
			OffLedgerConfig("OR('Org1MSP.member')", 1, 2, "1m").
			Delete(key1)
		err = s.Persist(txID3, b.Build())

		value, err = s.GetData(storeapi.NewKey(txID4, ns1, coll1, key1))
		assert.NoError(t, err)
		assert.Nil(t, value)
	})
}

func TestStore_DBError_NoCache(t *testing.T) {

	viper.Set("coll.offledger.cache.enable", "false")
	defer viper.Set("coll.offledger.cache.enable", "true")

	getLocalMSPID = func(collcommon.IdentifierProvider) (string, error) { return org1MSP, nil }

	dbProvider := olmocks.NewDBProvider()
	s := newStore(channelID, dbProvider, &mocks.IdentifierProvider{}, &mocks.IdentityDeserializer{}, typeConfig)
	require.NotNil(t, s)

	t.Run("GetData -> error", func(t *testing.T) {
		key := storeapi.NewKey(txID1, ns1, coll1, key1)

		expectedErr := fmt.Errorf("error getting DB")
		dbProvider.WithError(expectedErr)
		v, err := s.GetData(key)
		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
		assert.Nil(t, v)

		expectedErr = fmt.Errorf("error getting value")
		dbProvider.WithError(nil).MockDB(ns1, coll1).WithError(expectedErr)
		v, err = s.GetData(key)
		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
		assert.Nil(t, v)
	})

	t.Run("GetDataMultipleKeys -> error", func(t *testing.T) {
		key := storeapi.NewMultiKey(txID1, ns1, coll1, key1)

		expectedErr := fmt.Errorf("error getting DB")
		dbProvider.WithError(expectedErr)
		v, err := s.GetDataMultipleKeys(key)
		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
		assert.Nil(t, v)

		expectedErr = fmt.Errorf("error getting value")
		dbProvider.WithError(nil).MockDB(ns1, coll1).WithError(expectedErr)
		v, err = s.GetDataMultipleKeys(key)
		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
		assert.Nil(t, v)
	})

	t.Run("Persist -> error", func(t *testing.T) {
		expectedErr := fmt.Errorf("error getting DB")
		dbProvider.WithError(expectedErr)

		b := mocks.NewPvtReadWriteSetBuilder()
		b.Namespace(ns1).Collection(coll1).OffLedgerConfig("OR('Org1MSP.member')", 1, 2, "1m")

		err := s.Persist(txID1, b.Build())
		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())

		expectedErr = fmt.Errorf("error putting value")
		dbProvider.WithError(nil).MockDB(ns1, coll1).WithError(expectedErr)

		err = s.Persist(txID1, b.Build())
		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
	})

	t.Run("Persist -> error", func(t *testing.T) {
		expectedErr := fmt.Errorf("error getting DB")
		dbProvider.WithError(expectedErr)

		collConfig := &pb.StaticCollectionConfig{
			Type: pb.CollectionType_COL_OFFLEDGER,
			Name: coll1,
		}

		key := &storeapi.Key{EndorsedAtTxID: txID1, Namespace: ns1, Collection: coll1}
		value := &storeapi.ExpiringValue{Value: value1_1}

		err := s.PutData(collConfig, key, value)
		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())

		expectedErr = fmt.Errorf("error putting value")
		dbProvider.WithError(nil).MockDB(ns1, coll1).WithError(expectedErr)

		err = s.PutData(collConfig, key, value)
		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
	})
}
