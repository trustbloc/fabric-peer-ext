/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package storeprovider

import (
	"errors"
	"fmt"
	"testing"
	"time"

	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/core/common/privdata"
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	gcommon "github.com/hyperledger/fabric/gossip/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/storeprovider/store/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/discovery"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

const (
	channelID = "testchannel"

	txID1 = "txid1"
	txID2 = "txid2"
	txID3 = "txid3"
	txID4 = "txid4"

	ns1 = "ns1"

	coll0 = "coll0"
	coll1 = "coll1"
	coll2 = "coll2"
	coll3 = "coll3"

	key1 = "key1"
	key2 = "key2"
	key4 = "key4"
	key6 = "key6"

	collPolicy = "OR('Org1MSP.member','Org2MSP.member')"
)

var (
	org1MSPID      = "Org1MSP"
	p1Org1Endpoint = "p1.org1.com"
	p1Org1PKIID    = gcommon.PKIidType("pkiid_P1O1")
	p2Org1Endpoint = "p2.org1.com"
	p2Org1PKIID    = gcommon.PKIidType("pkiid_P2O1")

	org2MSPID      = "Org2MSP"
	p1Org2Endpoint = "p1.org2.com"
	p1Org2PKIID    = gcommon.PKIidType("pkiid_P1O2")
	p2Org2Endpoint = "p2.org2.com"
	p2Org2PKIID    = gcommon.PKIidType("pkiid_P2O2")
)

func TestStore(t *testing.T) {
	s := newStore(channelID, 100, false, newMockDB(), mocks.NewMockGossipAdapter(), &mocks.IdentityDeserializer{})
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
	value1_4 := []byte("value1_4")

	value2_1 := []byte("value2_1")
	value3_1 := []byte("value3_1")
	value4_1 := []byte("value4_1")

	b := mocks.NewPvtReadWriteSetBuilder()
	ns1Builder := b.Namespace(ns1)
	coll1Builder := ns1Builder.Collection(coll1)
	coll1Builder.
		TransientConfig(collPolicy, 1, 2, "1m").
		Write(key1, value1_1).
		Write(key2, value1_2).
		Write(key4, value1_4)
	coll2Builder := ns1Builder.Collection(coll2)
	coll2Builder.
		TransientConfig(collPolicy, 1, 2, "1m").
		Write(key1, value2_1).
		Write(key4, value4_1).
		Delete(key2)
	coll3Builder := ns1Builder.Collection(coll3)
	coll3Builder.
		StaticConfig("OR('Org1MSP.member')", 1, 2, 100).
		Write(key1, value3_1)

	p1Org1 := mocks.NewMember(p1Org1Endpoint, p1Org1PKIID)
	p2Org1 := mocks.NewMember(p2Org1Endpoint, p2Org1PKIID)
	p1Org2 := mocks.NewMember(p1Org2Endpoint, p1Org2PKIID)
	p2Org2 := mocks.NewMember(p2Org2Endpoint, p2Org2PKIID)

	gossip := mocks.NewMockGossipAdapter().
		Self(org1MSPID, p1Org1).
		Member(org1MSPID, p2Org1).
		Member(org2MSPID, p1Org2).
		Member(org2MSPID, p2Org2)

	s := newStore(channelID, 1, false, newMockDB(), gossip, &mocks.IdentityDeserializer{})
	require.NotNil(t, s)
	defer s.Close()

	// Expected endorsers:
	// - coll1-key1: p2.org2.com, p2.org1.com
	// - coll1-key2: p1.org1.com, p1.org2.com
	// - coll2-key1: p2.org2.com, p2.org1.com
	// - coll2-key2: p1.org1.com, p1.org2.com

	err := s.Persist(txID1, b.Build())
	assert.NoError(t, err)

	t.Run("GetTransientData invalid collection -> nil", func(t *testing.T) {
		value, err := s.GetTransientData(storeapi.NewKey(txID1, ns1, coll0, key1))
		assert.NoError(t, err)
		assert.Nil(t, value)
	})

	t.Run("GetTransientData in same transaction -> nil", func(t *testing.T) {
		value, err := s.GetTransientData(storeapi.NewKey(txID1, ns1, coll1, key2))
		require.NoError(t, err)
		require.Nil(t, value)

		value, err = s.GetTransientData(storeapi.NewKey(txID2, ns1, coll1, key2))
		require.NoError(t, err)
		require.NotNil(t, value)
		require.Equal(t, value1_2, value.Value)
	})

	t.Run("GetTransientData in new transaction -> valid", func(t *testing.T) {
		value, err := s.GetTransientData(storeapi.NewKey(txID2, ns1, coll1, key1))
		assert.NoError(t, err)
		assert.Nil(t, value) // key1 should not be persisted locally since p1.org1.com is not part of the peer group for key1

		value, err = s.GetTransientData(storeapi.NewKey(txID2, ns1, coll1, key2))
		assert.NoError(t, err)
		assert.Equal(t, value1_2, value.Value)
	})

	t.Run("GetTransientData collection2 -> valid", func(t *testing.T) {
		// coll2-key1
		value, err := s.GetTransientData(storeapi.NewKey(txID2, ns1, coll2, key1)) // key1 should not be persisted locally since p1.org1.com is not part of the peer group for key1
		assert.NoError(t, err)
		assert.Nil(t, value)

		// coll2-key4
		value, err = s.GetTransientData(storeapi.NewKey(txID2, ns1, coll2, key4))
		assert.NoError(t, err)
		assert.Equal(t, value4_1, value.Value)
	})

	t.Run("GetTransientData on non-transient collection -> nil", func(t *testing.T) {
		value, err := s.GetTransientData(storeapi.NewKey(txID2, ns1, coll3, key1))
		assert.NoError(t, err)
		assert.Nil(t, value)
	})

	t.Run("GetTransientDataMultipleKeys -> valid", func(t *testing.T) {
		values, err := s.GetTransientDataMultipleKeys(storeapi.NewMultiKey(txID2, ns1, coll1, key1, key2, key4))
		assert.NoError(t, err)
		require.Equal(t, 3, len(values))
		assert.Nil(t, values[0]) // key1 should not be persisted locally since p1.org1.com is not part of the peer group for key1
		assert.Equal(t, value1_2, values[1].Value)
		assert.Equal(t, value1_4, values[2].Value)
	})

	t.Run("Disallow update transient data", func(t *testing.T) {
		b := mocks.NewPvtReadWriteSetBuilder()
		nsBuilder1 := b.Namespace(ns1)
		collBuilder1 := nsBuilder1.Collection(coll2)
		collBuilder1.
			TransientConfig(collPolicy, 1, 2, "1m").
			Write(key6, value1_1)
		err = s.Persist(txID1, b.Build())
		require.NoError(t, err)

		value, err := s.GetTransientData(storeapi.NewKey(txID2, ns1, coll2, key6))
		require.NoError(t, err)
		require.Equal(t, value1_1, value.Value)

		b2 := mocks.NewPvtReadWriteSetBuilder()
		nsBuilder2 := b2.Namespace(ns1)
		collBuilder2 := nsBuilder2.Collection(coll2)
		collBuilder2.
			TransientConfig(collPolicy, 1, 2, "1m").
			Write(key6, value2_1)
		err = s.Persist(txID3, b2.Build())
		assert.NoError(t, err)

		value, err = s.GetTransientData(storeapi.NewKey(txID4, ns1, coll2, key6))
		assert.NoError(t, err)
		assert.Equalf(t, value1_1, value.Value, "expecting transient data to not have been updated")
	})

	t.Run("Expire transient data", func(t *testing.T) {
		b := mocks.NewPvtReadWriteSetBuilder()
		ns1Builder := b.Namespace(ns1)
		coll1Builder := ns1Builder.Collection(coll1)
		coll1Builder.
			TransientConfig(collPolicy, 1, 2, "10ms").
			Write(key6, value1_1)

		err = s.Persist(txID2, b.Build())
		assert.NoError(t, err)

		value, err := s.GetTransientData(storeapi.NewKey(txID3, ns1, coll1, key6))
		assert.NoError(t, err)
		assert.Equal(t, value1_1, value.Value)

		time.Sleep(15 * time.Millisecond)

		value, err = s.GetTransientData(storeapi.NewKey(txID3, ns1, coll1, key6))
		assert.NoError(t, err)
		assert.Nilf(t, value, "expecting key to have expired")
	})
}

func TestStoreInvalidData(t *testing.T) {
	s := newStore(channelID, 100, false, newMockDB(), mocks.NewMockGossipAdapter(), &mocks.IdentityDeserializer{})
	require.NotNil(t, s)
	defer s.Close()

	t.Run("Invalid RW set", func(t *testing.T) {
		b := mocks.NewPvtReadWriteSetBuilder()
		b.Namespace(ns1).
			Collection(coll1).
			TransientConfig(collPolicy, 1, 2, "1m").
			Write(key1, []byte("value")).
			WithMarshalError()

		err := s.Persist(txID1, b.Build())
		require.Errorf(t, err, "expecting marshal error")
	})

	t.Run("Invalid time-to-live", func(t *testing.T) {
		b := mocks.NewPvtReadWriteSetBuilder()
		b.Namespace(ns1).
			Collection(coll1).
			TransientConfig(collPolicy, 1, 2, "1xxx").
			Write(key1, []byte("value"))

		err := s.Persist(txID1, b.Build())
		require.Error(t, err)
		require.Contains(t, err.Error(), "error parsing time-to-live for collection")
	})

	t.Run("Invalid policy", func(t *testing.T) {
		p, err := s.loadPolicy(ns1, &pb.StaticCollectionConfig{})
		require.Error(t, err)
		require.Nil(t, p)
		require.Contains(t, err.Error(), "error setting up collection policy")
	})

	t.Run("Resolver error", func(t *testing.T) {
		restore := getResolver
		errExpected := errors.New("resolver error")
		getResolver = func(channelID, ns, coll string, policy privdata.CollectionAccessPolicy, gossip gossipAdapter) endorserResolver {
			return newMockResolver().WithError(errExpected)
		}
		defer func() { getResolver = restore }()

		b := mocks.NewPvtReadWriteSetBuilder()
		b.Namespace(ns1).
			Collection(coll1).
			TransientConfig(collPolicy, 1, 2, "1m").
			Write(key1, []byte("value"))

		err := s.Persist(txID1, b.Build())
		require.EqualError(t, err, errExpected.Error())
	})
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

type mockResolver struct {
	err error
}

func newMockResolver() *mockResolver {
	return &mockResolver{}
}

func (m *mockResolver) WithError(err error) *mockResolver {
	m.err = err
	return m
}

func (m *mockResolver) ResolveEndorsers(key string) (discovery.PeerGroup, error) {
	if m.err != nil {
		return nil, m.err
	}
	panic("not implemented")
}
