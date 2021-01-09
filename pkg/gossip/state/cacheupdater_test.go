/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package state

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	gproto "github.com/hyperledger/fabric-protos-go/gossip"
	gcommon "github.com/hyperledger/fabric/gossip/common"
	viper "github.com/spf13/viper2015"
	"github.com/stretchr/testify/require"

	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	cmocks "github.com/trustbloc/fabric-peer-ext/pkg/common/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/requestmgr"
	"github.com/trustbloc/fabric-peer-ext/pkg/gossip/appdata"
	statemocks "github.com/trustbloc/fabric-peer-ext/pkg/gossip/state/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

const (
	org1MSPID      = "Org1MSP"
	p0Org1Endpoint = "p0.org1.com"
	p1Org1Endpoint = "p1.org1.com"
)

var (
	p0Org1PKIID = gcommon.PKIidType("pkiid_P0O1")
	p1Org1PKIID = gcommon.PKIidType("pkiid_P1O1")
)

// Ensure that roles are initialized
var _ = roles.GetRoles()

func TestSaveCacheUpdates(t *testing.T) {
	reset := initRoles(roles.CommitterRole)
	defer reset()

	stateDBProvider := &cmocks.StateDBProvider{}
	stateDB := &mocks.StateDB{}
	stateDBProvider.StateDBForChannelReturns(stateDB)

	peerConfig := &mocks.PeerConfig{}
	peerConfig.MSPIDReturns("org1")

	gossipProvider := &mocks.GossipProvider{}
	gossipService := mocks.NewMockGossipAdapter()
	gossipProvider.GetGossipServiceReturns(gossipService)

	providers := &Providers{
		HandlerRegistry: &statemocks.AppDataHandlerRegistry{},
		StateDBProvider: stateDBProvider,
		MspProvider:     peerConfig,
		GossipProvider:  gossipProvider,
	}

	t.Run("Pre-populate enabled", func(t *testing.T) {
		viper.Set("ledger.state.dbConfig.cache.prePopulate", true)
		updateHandler = NewUpdateHandler(providers)
		require.NotPanics(t, func() { SaveCacheUpdates(channel1, 1001, []byte("cache-updates")) })
	})

	t.Run("Pre-populate disabled", func(t *testing.T) {
		viper.Set("ledger.state.dbConfig.cache.prePopulate", false)
		updateHandler = NewUpdateHandler(providers)
		require.NotPanics(t, func() { SaveCacheUpdates(channel1, 1001, []byte("cache-updates")) })
	})
}

func TestUpdateCache(t *testing.T) {
	reset := initRoles(roles.EndorserRole)
	defer reset()

	stateDBProvider := &cmocks.StateDBProvider{}
	stateDB := &mocks.StateDB{}
	stateDBProvider.StateDBForChannelReturns(stateDB)

	peerConfig := &mocks.PeerConfig{}
	peerConfig.MSPIDReturns(org1MSPID)

	gossipProvider := &mocks.GossipProvider{}
	gossipService := mocks.NewMockGossipAdapter()
	gossipService.Self(org1MSPID, mocks.NewMember(p1Org1Endpoint, p1Org1PKIID)).
		Member(org1MSPID, mocks.NewMember(p0Org1Endpoint, p0Org1PKIID, string(roles.CommitterRole)))
	gossipProvider.GetGossipServiceReturns(gossipService)

	providers := &Providers{
		HandlerRegistry: &statemocks.AppDataHandlerRegistry{},
		StateDBProvider: stateDBProvider,
		MspProvider:     peerConfig,
		GossipProvider:  gossipProvider,
	}

	t.Run("Pre-populate disabled", func(t *testing.T) {
		viper.Set("ledger.state.dbConfig.cache.prePopulate", false)

		updateHandler = NewUpdateHandler(providers)

		require.NoError(t, UpdateCache(channel1, 1001))
	})

	t.Run("With no response", func(t *testing.T) {
		viper.Set("ledger.state.dbConfig.cache.prePopulate", true)

		updateHandler = NewUpdateHandler(providers)

		require.NoError(t, UpdateCache(channel1, 1001))
	})

	t.Run("With success response", func(t *testing.T) {
		viper.Set("ledger.state.dbConfig.cache.prePopulate", true)

		var req requestmgr.Request
		var mutex sync.RWMutex

		reqMgr := requestmgr.Get(channel1)
		req = reqMgr.NewRequest()

		restoreStateRetriever := createStateRetriever
		defer func() { createStateRetriever = restoreStateRetriever }()

		createStateRetriever = func(channelID string, gossipService gossipapi.GossipService) stateRetriever {
			return appdata.NewRetrieverForTest(channel1, gossipService, 1, 1,
				func() requestmgr.Request {
					mutex.Lock()
					defer mutex.Unlock()

					req = reqMgr.NewRequest()
					return req
				},
			)
		}

		response := &requestmgr.Response{
			Endpoint: "peer0",
			MSPID:    org1MSPID,
			Data:     []byte("cache-updates"),
		}

		go func() {
			time.Sleep(250 * time.Millisecond)

			mutex.RLock()
			defer mutex.RUnlock()

			requestmgr.Get(channel1).Respond(req.ID(), response)
		}()

		updateHandler = NewUpdateHandler(providers)

		require.NoError(t, UpdateCache(channel1, 1001))
	})

	t.Run("With marshal error", func(t *testing.T) {
		viper.Set("ledger.state.dbConfig.cache.prePopulate", true)

		errExpected := fmt.Errorf("injected marshal error")
		jsonMarshal = func(v interface{}) ([]byte, error) { return nil, errExpected }
		defer func() { jsonMarshal = json.Marshal }()

		updateHandler = NewUpdateHandler(providers)

		require.EqualError(t, UpdateCache(channel1, 1001), errExpected.Error())
	})
}

func TestHandleCacheUpdatesRequest(t *testing.T) {
	reset := initRoles(roles.EndorserRole)
	defer reset()

	viper.Set("ledger.state.dbConfig.cache.prePopulate", true)

	stateDBProvider := &cmocks.StateDBProvider{}
	stateDB := &mocks.StateDB{}
	stateDBProvider.StateDBForChannelReturns(stateDB)

	peerConfig := &mocks.PeerConfig{}
	peerConfig.MSPIDReturns(org1MSPID)

	gossipProvider := &mocks.GossipProvider{}
	gossipProvider.GetGossipServiceReturns(mocks.NewMockGossipAdapter())

	providers := &Providers{
		HandlerRegistry: &statemocks.AppDataHandlerRegistry{},
		StateDBProvider: stateDBProvider,
		MspProvider:     peerConfig,
		GossipProvider:  gossipProvider,
	}

	reqBytes, err := json.Marshal(&cacheUpdatesRequest{
		ChannelID: channel1,
		BlockNum:  1001,
	})
	require.NoError(t, err)

	req := &gproto.AppDataRequest{
		Nonce:    0,
		DataType: cacheUpdatesDataType,
		Request:  reqBytes,
	}

	t.Run("Pre-populate disabled", func(t *testing.T) {
		viper.Set("ledger.state.dbConfig.cache.prePopulate", false)

		updateHandler = NewUpdateHandler(providers)

		resp := &mockResponder{}

		updateHandler.handleCacheUpdatesRequest(channel1, req, resp)
		require.Nil(t, resp.data)
	})

	t.Run("With success response", func(t *testing.T) {
		viper.Set("ledger.state.dbConfig.cache.prePopulate", true)

		updateHandler = NewUpdateHandler(providers)

		require.NotPanics(t, func() { SaveCacheUpdates(channel1, 1001, []byte("cache-updates")) })

		resp := &mockResponder{}

		updateHandler.handleCacheUpdatesRequest(channel1, req, resp)
		require.NoError(t, err)
		require.NotNil(t, resp.data)
	})

	t.Run("Cache updates not found", func(t *testing.T) {
		viper.Set("ledger.state.dbConfig.cache.prePopulate", true)

		updateHandler = NewUpdateHandler(providers)

		resp := &mockResponder{}

		updateHandler.handleCacheUpdatesRequest(channel1, req, resp)
		require.NoError(t, err)
		require.Nil(t, resp.data)
	})

	t.Run("With unmarshal error", func(t *testing.T) {
		viper.Set("ledger.state.dbConfig.cache.prePopulate", true)

		errExpected := fmt.Errorf("injected unmarshal error")
		jsonUnmarshal = func(data []byte, v interface{}) error { return errExpected }
		defer func() { jsonUnmarshal = json.Unmarshal }()

		updateHandler = NewUpdateHandler(providers)

		require.NotPanics(t, func() { SaveCacheUpdates(channel1, 1001, []byte("cache-updates")) })

		resp := &mockResponder{}

		updateHandler.handleCacheUpdatesRequest(channel1, req, resp)
		require.Nil(t, resp.data)
	})
}

type mockResponder struct {
	data []byte
}

func (m *mockResponder) Respond(data []byte) {
	m.data = data
}
