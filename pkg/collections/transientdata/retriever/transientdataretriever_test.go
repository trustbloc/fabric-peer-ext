/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package retriever

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	gproto "github.com/hyperledger/fabric-protos-go/gossip"
	gcommon "github.com/hyperledger/fabric/gossip/common"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"

	collcommon "github.com/trustbloc/fabric-peer-ext/pkg/collections/common"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/requestmgr"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

const (
	channelID = "testchannel"
	ns1       = "chaincode1"
	coll1     = "collection1"
	key1      = "key1"
	key2      = "key2"
	key3      = "key3"
	txID      = "tx1"
)

var (
	org1MSPID      = "Org1MSP"
	p1Org1Endpoint = "p1.org1.com"
	p1Org1PKIID    = gcommon.PKIidType("pkiid_P1O1")
	p2Org1Endpoint = "p2.org1.com"
	p2Org1PKIID    = gcommon.PKIidType("pkiid_P2O1")
	p3Org1Endpoint = "p3.org1.com"
	p3Org1PKIID    = gcommon.PKIidType("pkiid_P3O1")

	org2MSPID      = "Org2MSP"
	p1Org2Endpoint = "p1.org2.com"
	p1Org2PKIID    = gcommon.PKIidType("pkiid_P1O2")
	p2Org2Endpoint = "p2.org2.com"
	p2Org2PKIID    = gcommon.PKIidType("pkiid_P2O2")
	p3Org2Endpoint = "p3.org2.com"
	p3Org2PKIID    = gcommon.PKIidType("pkiid_P3O2")

	org3MSPID      = "Org3MSP"
	p1Org3Endpoint = "p1.org3.com"
	p1Org3PKIID    = gcommon.PKIidType("pkiid_P1O3")
	p2Org3Endpoint = "p2.org3.com"
	p2Org3PKIID    = gcommon.PKIidType("pkiid_P2O3")
	p3Org3Endpoint = "p3.org3.com"
	p3Org3PKIID    = gcommon.PKIidType("pkiid_P3O3")

	validatorRole = string(roles.ValidatorRole)
	endorserRole  = string(roles.EndorserRole)

	respTimeout = 100 * time.Millisecond
)

func TestTransientDataProvider(t *testing.T) {
	value1 := &storeapi.ExpiringValue{Value: []byte("value1")}
	value2 := &storeapi.ExpiringValue{Value: []byte("value2")}
	value3 := &storeapi.ExpiringValue{Value: []byte("value3")}

	ccRetriever := mocks.NewCollectionConfigRetriever().
		WithCollectionPolicy(&mocks.MockAccessPolicy{
			MaxPeerCount: 2,
			Orgs:         []string{org1MSPID, org2MSPID, org3MSPID},
		})
	ccProvider := &mocks.CollectionConfigProvider{}
	ccProvider.ForChannelReturns(ccRetriever)

	identifierProvider := &mocks.IdentifierProvider{}
	identifierProvider.GetIdentifierReturns(org1MSPID, nil)

	gossip := mocks.NewMockGossipAdapter()
	gossip.Self(org1MSPID, mocks.NewMember(p1Org1Endpoint, p1Org1PKIID)).
		Member(org1MSPID, mocks.NewMember(p2Org1Endpoint, p2Org1PKIID, validatorRole)).
		Member(org1MSPID, mocks.NewMember(p3Org1Endpoint, p3Org1PKIID, validatorRole)).
		Member(org2MSPID, mocks.NewMember(p1Org2Endpoint, p1Org2PKIID, endorserRole)).
		Member(org2MSPID, mocks.NewMember(p2Org2Endpoint, p2Org2PKIID, validatorRole)).
		Member(org2MSPID, mocks.NewMember(p3Org2Endpoint, p3Org2PKIID, endorserRole)).
		Member(org3MSPID, mocks.NewMember(p1Org3Endpoint, p1Org3PKIID, endorserRole)).
		Member(org3MSPID, mocks.NewMember(p2Org3Endpoint, p2Org3PKIID, validatorRole)).
		Member(org3MSPID, mocks.NewMember(p3Org3Endpoint, p3Org3PKIID, endorserRole)).IdentityInfo()

	localStore := mocks.NewDataStore().
		TransientData(storeapi.NewKey(txID, ns1, coll1, key1), value1)

	storeProvider := &mocks.StoreProvider{}
	storeProvider.StoreForChannelReturns(localStore)

	gossipProvider := &mocks.GossipProvider{}
	gossipProvider.GetGossipServiceReturns(gossip)

	providers := &collcommon.Providers{
		BlockPublisherProvider: mocks.NewBlockPublisherProvider(),
		StoreProvider:          storeProvider,
		GossipProvider:         gossipProvider,
		CCProvider:             ccProvider,
		IdentifierProvider:     identifierProvider,
	}
	p := NewProvider(providers)

	retriever := p.RetrieverForChannel(channelID)
	require.NotNil(t, retriever)

	t.Run("GetTransientData - From local peer", func(t *testing.T) {
		ctx, _ := context.WithTimeout(context.Background(), respTimeout)
		value, err := retriever.GetTransientData(ctx, storeapi.NewKey(txID, ns1, coll1, key1))
		require.NoError(t, err)
		require.NotNil(t, value)
		require.Equal(t, value1, value)
	})

	t.Run("GetTransientData - From local peer", func(t *testing.T) {
		errExpected := errors.New("identifier error")
		identifierProvider.GetIdentifierReturns("", errExpected)
		defer func() { identifierProvider.GetIdentifierReturns(org1MSPID, nil) }()

		ctx, _ := context.WithTimeout(context.Background(), respTimeout)
		value, err := retriever.GetTransientData(ctx, storeapi.NewKey(txID, ns1, coll1, key1))
		require.EqualError(t, errors.Cause(err), errExpected.Error())
		require.Nil(t, value)
	})

	t.Run("GetTransientData - From remote peer", func(t *testing.T) {
		gossip.MessageHandler(
			newMockGossipMsgHandler(channelID).
				Value(key2, value2).
				Handle)

		ctx, _ := context.WithTimeout(context.Background(), respTimeout)
		value, err := retriever.GetTransientData(ctx, storeapi.NewKey(txID, ns1, coll1, key2))
		require.NoError(t, err)
		require.NotNil(t, value)
		require.Equal(t, value2, value)
	})

	t.Run("GetTransientData - No response from remote peer", func(t *testing.T) {
		gossip.MessageHandler(func(msg *gproto.GossipMessage) {})

		ctx, _ := context.WithTimeout(context.Background(), respTimeout)
		value, err := retriever.GetTransientData(ctx, storeapi.NewKey(txID, ns1, coll1, key2))
		require.NoError(t, err)
		assert.Nil(t, value)
	})

	t.Run("GetTransientData - Cancel request from remote peer", func(t *testing.T) {
		gossip.MessageHandler(func(msg *gproto.GossipMessage) {})

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		value, err := retriever.GetTransientData(ctx, storeapi.NewKey(txID, ns1, coll1, key2))
		require.NoError(t, err)
		assert.Nil(t, value)
	})

	t.Run("GetTransientDataMultipleKeys - 1 key -> success", func(t *testing.T) {
		gossip.MessageHandler(
			newMockGossipMsgHandler(channelID).
				Value(key1, value2).
				Handle)

		ctx, _ := context.WithTimeout(context.Background(), respTimeout)
		values, err := retriever.GetTransientDataMultipleKeys(ctx, storeapi.NewMultiKey(txID, ns1, coll1, key1))
		require.NoError(t, err)
		require.Equal(t, 1, len(values))
		assert.Equal(t, value1, values[0])
	})

	t.Run("GetTransientDataMultipleKeys - 3 keys -> success", func(t *testing.T) {
		gossip.MessageHandler(
			newMockGossipMsgHandler(channelID).
				Value(key1, value1).
				Value(key2, value2).
				Value(key3, value3).
				Handle)

		ctx, _ := context.WithTimeout(context.Background(), respTimeout)
		values, err := retriever.GetTransientDataMultipleKeys(ctx, storeapi.NewMultiKey(txID, ns1, coll1, key1, key2, key3))
		require.NoError(t, err)
		require.Equal(t, 3, len(values))
		assert.Equal(t, value1, values[0])
		assert.Equal(t, value2, values[1])
		assert.Equal(t, value3, values[2])
	})

	t.Run("GetTransientDataMultipleKeys - Key not found -> success", func(t *testing.T) {
		gossip.MessageHandler(
			newMockGossipMsgHandler(channelID).
				Value(key1, value1).
				Value(key2, value2).
				Value(key3, value3).
				Handle)

		ctx, _ := context.WithTimeout(context.Background(), respTimeout)
		values, err := retriever.GetTransientDataMultipleKeys(ctx, storeapi.NewMultiKey(txID, ns1, coll1, "xxx", key2, key3))
		require.NoError(t, err)
		require.Equal(t, 3, len(values))
		assert.Nil(t, values[0])
		assert.Equal(t, value2, values[1])
		assert.Equal(t, value3, values[2])
	})

	t.Run("GetTransientDataMultipleKeys - Timeout -> fail", func(t *testing.T) {
		handler := newMockGossipMsgHandler(channelID).
			Value(key1, value1).
			Value(key2, value2).
			Value(key3, value3)
		gossip.MessageHandler(
			func(msg *gproto.GossipMessage) {
				time.Sleep(10 * time.Millisecond)
				handler.Handle(msg)
			},
		)

		ctx, _ := context.WithTimeout(context.Background(), time.Microsecond)
		values, err := retriever.GetTransientDataMultipleKeys(ctx, storeapi.NewMultiKey(txID, ns1, coll1, key2, key3))
		require.NoError(t, err)
		assert.True(t, values.Values().IsEmpty())
	})
}

func TestTransientDataProvider_AccessDenied(t *testing.T) {
	value1 := &storeapi.ExpiringValue{Value: []byte("value1")}
	value2 := &storeapi.ExpiringValue{Value: []byte("value2")}

	identifierProvider := &mocks.IdentifierProvider{}
	identifierProvider.GetIdentifierReturns(org3MSPID, nil)

	gossip := mocks.NewMockGossipAdapter()
	gossip.Self(org3MSPID, mocks.NewMember(p1Org3Endpoint, p1Org3PKIID)).
		Member(org1MSPID, mocks.NewMember(p2Org1Endpoint, p2Org1PKIID, validatorRole)).
		Member(org1MSPID, mocks.NewMember(p3Org1Endpoint, p3Org1PKIID, validatorRole)).
		Member(org2MSPID, mocks.NewMember(p1Org2Endpoint, p1Org2PKIID, endorserRole)).
		Member(org2MSPID, mocks.NewMember(p2Org2Endpoint, p2Org2PKIID, validatorRole)).
		Member(org2MSPID, mocks.NewMember(p3Org2Endpoint, p3Org2PKIID, endorserRole))

	gossip.MessageHandler(
		newMockGossipMsgHandler(channelID).
			Value(key2, value2).
			Handle)

	localStore := mocks.NewDataStore().
		TransientData(storeapi.NewKey(txID, ns1, coll1, key1), value1)

	ccRetriever := mocks.NewCollectionConfigRetriever().
		WithCollectionPolicy(&mocks.MockAccessPolicy{
			MaxPeerCount: 2,
			Orgs:         []string{org1MSPID, org2MSPID},
		})
	ccProvider := &mocks.CollectionConfigProvider{}
	ccProvider.ForChannelReturns(ccRetriever)

	blockPublisher := mocks.NewBlockPublisher()

	storeProvider := &mocks.StoreProvider{}
	storeProvider.StoreForChannelReturns(localStore)

	gossipProvider := &mocks.GossipProvider{}
	gossipProvider.GetGossipServiceReturns(gossip)

	providers := &collcommon.Providers{
		BlockPublisherProvider: mocks.NewBlockPublisherProvider().WithBlockPublisher(blockPublisher),
		StoreProvider:          storeProvider,
		GossipProvider:         gossipProvider,
		CCProvider:             ccProvider,
		IdentifierProvider:     identifierProvider,
	}
	p := NewProvider(providers)

	retriever := p.RetrieverForChannel(channelID)
	require.NotNil(t, retriever)

	t.Run("GetTransientData - From remote peer -> nil (not authorized)", func(t *testing.T) {
		ctx, _ := context.WithTimeout(context.Background(), respTimeout)
		value, err := retriever.GetTransientData(ctx, storeapi.NewKey(txID, ns1, coll1, key2))
		assert.NoError(t, err)
		assert.Nil(t, value)
	})

	t.Run("GetTransientDataMultipleKeys - From remote peer -> nil (not authorized)", func(t *testing.T) {
		ctx, _ := context.WithTimeout(context.Background(), respTimeout)
		values, err := retriever.GetTransientDataMultipleKeys(ctx, storeapi.NewMultiKey(txID, ns1, coll1, key2))
		assert.NoError(t, err)
		require.Equal(t, 1, len(values))
		assert.Nil(t, values[0])
	})

	t.Run("GetTransientData - From remote peer after CC upgrade -> success", func(t *testing.T) {
		ccRetriever.WithCollectionPolicy(&mocks.MockAccessPolicy{
			MaxPeerCount: 2,
			Orgs:         []string{org1MSPID, org2MSPID, org3MSPID},
		})

		require.NoError(t, blockPublisher.HandleUpgrade(gossipapi.TxMetadata{BlockNum: 1001, TxID: txID}, ns1))
		ctx, _ := context.WithTimeout(context.Background(), respTimeout)
		value, err := retriever.GetTransientData(ctx, storeapi.NewKey(txID, ns1, coll1, key2))
		assert.NoError(t, err)
		assert.NotNil(t, value)
	})
}

func TestTransientDataFailover(t *testing.T) {
	value := &storeapi.ExpiringValue{Value: []byte("value")}

	ccRetriever := mocks.NewCollectionConfigRetriever().
		WithCollectionPolicy(&mocks.MockAccessPolicy{
			MaxPeerCount: 2,
			Orgs:         []string{org1MSPID, org2MSPID, org3MSPID},
		})
	ccProvider := &mocks.CollectionConfigProvider{}
	ccProvider.ForChannelReturns(ccRetriever)

	identifierProvider := &mocks.IdentifierProvider{}
	identifierProvider.GetIdentifierReturns(org1MSPID, nil)

	gossip := mocks.NewMockGossipAdapter()
	gossip.Self(org1MSPID, mocks.NewMember(p1Org1Endpoint, p1Org1PKIID)).
		Member(org1MSPID, mocks.NewMember(p2Org1Endpoint, p2Org1PKIID, validatorRole)).
		Member(org1MSPID, mocks.NewMember(p3Org1Endpoint, p3Org1PKIID, endorserRole)).
		Member(org2MSPID, mocks.NewMember(p1Org2Endpoint, p1Org2PKIID, endorserRole)).
		Member(org2MSPID, mocks.NewMember(p2Org2Endpoint, p2Org2PKIID, validatorRole)).
		Member(org2MSPID, mocks.NewMember(p3Org2Endpoint, p3Org2PKIID, endorserRole)).
		Member(org3MSPID, mocks.NewMember(p1Org3Endpoint, p1Org3PKIID, endorserRole)).
		Member(org3MSPID, mocks.NewMember(p2Org3Endpoint, p2Org3PKIID, validatorRole)).
		Member(org3MSPID, mocks.NewMember(p3Org3Endpoint, p3Org3PKIID, endorserRole)).IdentityInfo()

	localStore := mocks.NewDataStore()

	storeProvider := &mocks.StoreProvider{}
	storeProvider.StoreForChannelReturns(localStore)

	gossipProvider := &mocks.GossipProvider{}
	gossipProvider.GetGossipServiceReturns(gossip)

	providers := &collcommon.Providers{
		BlockPublisherProvider: mocks.NewBlockPublisherProvider(),
		StoreProvider:          storeProvider,
		GossipProvider:         gossipProvider,
		CCProvider:             ccProvider,
		IdentifierProvider:     identifierProvider,
	}
	p := NewProvider(providers)

	retriever := p.RetrieverForChannel(channelID)
	require.NotNil(t, retriever)

	t.Run("GetTransientData", func(t *testing.T) {
		gossip.MessageHandler(
			newMockGossipMsgHandler(channelID).
				ValueOnCall(0, key2, nil).
				ValueOnCall(1, key2, nil).
				ValueOnCall(2, key2, nil).
				ValueOnCall(3, key2, nil).
				ValueOnCall(4, key2, value).
				Handle)

		ctx, _ := context.WithTimeout(context.Background(), respTimeout)
		value, err := retriever.GetTransientData(ctx, storeapi.NewKey(txID, ns1, coll1, key2))
		require.NoError(t, err)
		require.NotNil(t, value)
		require.Equal(t, value, value)
	})
}

type mockGossipMsgHandler struct {
	channelID    string
	values       map[string]*storeapi.ExpiringValue
	valuesOnCall []map[string]*storeapi.ExpiringValue
	call         int32
}

func newMockGossipMsgHandler(channelID string) *mockGossipMsgHandler {
	return &mockGossipMsgHandler{
		channelID: channelID,
		values:    make(map[string]*storeapi.ExpiringValue),
	}
}

func (m *mockGossipMsgHandler) Value(key string, value *storeapi.ExpiringValue) *mockGossipMsgHandler {
	m.values[key] = value
	return m
}

func (m *mockGossipMsgHandler) ValueOnCall(i int, key string, value *storeapi.ExpiringValue) *mockGossipMsgHandler {
	if len(m.valuesOnCall) <= i {
		m.valuesOnCall = append(m.valuesOnCall, make(map[string]*storeapi.ExpiringValue))
	}

	m.valuesOnCall[i][key] = value
	return m
}

func (m *mockGossipMsgHandler) Handle(msg *gproto.GossipMessage) {
	req := msg.GetCollDataReq()

	res := &requestmgr.Response{
		Endpoint:  "p1.org1",
		MSPID:     "org1",
		Identity:  []byte("p1.org1"),
		Signature: []byte("sig"),
	}

	var values map[string]*storeapi.ExpiringValue

	call := atomic.LoadInt32(&m.call)

	if len(m.valuesOnCall) > int(call) {
		values = m.valuesOnCall[call]
	} else {
		values = m.values
	}

	for _, d := range req.Digests {
		e := &requestmgr.Element{
			Namespace:  d.Namespace,
			Collection: d.Collection,
			Key:        d.Key,
		}

		v := values[d.Key]
		if v != nil {
			e.Value = v.Value
			e.Expiry = v.Expiry
		}

		res.Data = append(requestmgr.AsElements(res.Data), e)
	}

	atomic.AddInt32(&m.call, 1)

	requestmgr.Get(m.channelID).Respond(req.Nonce, res)
}
