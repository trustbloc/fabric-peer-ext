/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package state

import (
	"context"
	"encoding/json"

	"github.com/bluele/gcache"
	gproto "github.com/hyperledger/fabric-protos-go/gossip"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	ccapi "github.com/hyperledger/fabric/extensions/chaincode/api"
	"github.com/hyperledger/fabric/extensions/gossip/api"
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"

	"github.com/trustbloc/fabric-peer-ext/pkg/collections/common"
	extcommon "github.com/trustbloc/fabric-peer-ext/pkg/common"
	"github.com/trustbloc/fabric-peer-ext/pkg/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/gossip/appdata"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
	extstatedb "github.com/trustbloc/fabric-peer-ext/pkg/statedb"
)

var logger = flogging.MustGetLogger("ext_gossip_state")

var updateHandler *UpdateHandler

type blockPublisherProvider interface {
	ForChannel(cid string) api.BlockPublisher
}

type ccEventMgrProvider interface {
	GetMgr() ccapi.EventMgr
}

type appDataHandlerRegistry interface {
	Register(dataType string, handler appdata.Handler) error
}

type stateRetriever interface {
	Retrieve(ctxt context.Context, request *appdata.Request, responseHandler appdata.ResponseHandler, allSet appdata.AllSet, opts ...appdata.Option) (extcommon.Values, error)
}

type stateDBProvider interface {
	StateDBForChannel(channelID string) extstatedb.StateDB
}

type mspProvider interface {
	MSPID() string
}

type providers struct {
	BPProvider      blockPublisherProvider
	MgrProvider     ccEventMgrProvider
	MspProvider     mspProvider
	GossipProvider  common.GossipProvider
	StateDBProvider stateDBProvider
	HandlerRegistry appDataHandlerRegistry
}

// UpdateHandler handles updates to LSCC state and the state cache.
// As blocks are committed, the state cache is updated on the committing peer and the cache-updates are temporarily stored.
// When an endorsing peer receives a validated block from the committing peer, a request is made to the committing peer for
// the cache-updates. When the cache-updates are received then the cache on the endorsing peer is updated (pre-populated).
type UpdateHandler struct {
	*providers
	cacheUpdaters gcache.Cache
}

// NewUpdateHandler returns a new state update handler.
func NewUpdateHandler(providers *providers) *UpdateHandler {
	logger.Info("Creating new cache update updateHandler")

	h := &UpdateHandler{
		providers: providers,
	}

	h.cacheUpdaters = gcache.New(0).LoaderFunc(func(channelID interface{}) (interface{}, error) {
		return h.createCacheUpdater(channelID.(string)), nil
	}).Build()

	if roles.IsCommitter() {
		logger.Info("Registering cache updates request handler")

		if err := h.HandlerRegistry.Register(cacheUpdatesDataType, h.handleCacheUpdatesRequest); err != nil {
			// Should never happen
			panic(err)
		}
	}

	updateHandler = h

	return h
}

// ChannelJoined is called when a peer joins a channel
func (h *UpdateHandler) ChannelJoined(channelID string) {
	if roles.IsCommitter() {
		logger.Debugf("[%s] Not adding LSCC write handler since I'm a committer", channelID)
		return
	}

	logger.Debugf("[%s] Adding LSCC write handler", channelID)
	h.BPProvider.ForChannel(channelID).AddLSCCWriteHandler(
		func(txMetadata api.TxMetadata, chaincodeName string, ccData *ccprovider.ChaincodeData, _ *pb.CollectionConfigPackage) error {
			logger.Debugf("[%s] Got LSCC write event for [%s].", channelID, chaincodeName)
			return h.handleStateUpdate(txMetadata.ChannelID, chaincodeName, ccData)
		},
	)
}

func (h *UpdateHandler) handleStateUpdate(channelID string, chaincodeName string, ccData *ccprovider.ChaincodeData) error {
	// Chaincode instantiate/upgrade is not logged on committing peer anywhere else.  This is a good place to log it.
	logger.Debugf("[%s] Handling LSCC state update for chaincode [%s]", channelID, chaincodeName)

	chaincodeDefs := []*ccapi.Definition{
		{Name: ccData.Name, Version: ccData.Version, Hash: ccData.Id},
	}

	mgr := h.MgrProvider.GetMgr()

	err := mgr.HandleChaincodeDeploy(channelID, chaincodeDefs)
	// HandleChaincodeDeploy acquires a lock and does not unlock it, even if an error occurs.
	// We need to unlock it, regardless of an error.
	defer mgr.ChaincodeDeployDone(channelID)

	if err != nil {
		logger.Errorf("[%s] Error handling LSCC state update for chaincode [%s]: %s", channelID, chaincodeName, err)
		return err
	}

	return nil
}

func (h *UpdateHandler) getCacheUpdater(channelID string) cacheUpdater {
	cu, err := h.cacheUpdaters.Get(channelID)
	if err != nil {
		// Should never happen
		panic(err)
	}

	return cu.(cacheUpdater)
}

func (h *UpdateHandler) createCacheUpdater(channelID string) cacheUpdater {
	if !isPrePopulateStateCache() {
		logger.Infof("[%s] Pre-populating the state cache is disabled", channelID)

		return noCacheUpdates
	}

	logger.Infof("[%s] Creating cache cacheUpdater with retention size [%d] and request timeout [%s]",
		channelID, config.GetStateCacheRetentionSize(), config.GetStateCacheGossipTimeout())

	return &gossipCacheUpdater{
		channelID:      channelID,
		mspID:          h.MspProvider.MSPID(),
		stateDB:        h.StateDBProvider.StateDBForChannel(channelID),
		stateRetriever: createStateRetriever(channelID, h.GossipProvider.GetGossipService()),
		updates:        gcache.New(config.GetStateCacheRetentionSize()).LRU().Build(),
		timeout:        config.GetStateCacheGossipTimeout(),
	}
}

func (h *UpdateHandler) handleCacheUpdatesRequest(channelID string, request *gproto.AppDataRequest) ([]byte, error) {
	req := &cacheUpdatesRequest{}
	err := jsonUnmarshal(request.Request, req)
	if err != nil {
		return nil, err
	}

	logger.Debugf("[%s] Handling cache updates request for block [%d]", channelID, req.BlockNum)

	updates, err := h.getCacheUpdater(channelID).GetCacheUpdatesForBlock(req.BlockNum)
	if err != nil {
		if err == errNotFound {
			// This isn't an error; it just means that there are no cache updates for the requested block.
			// (i.e. it could be that there are no writes for the block)
			logger.Debugf("[%s] Cache updates not found for block [%d]", channelID, req.BlockNum)

			return nil, nil
		}

		// Should never happen
		return nil, err
	}

	return updates, nil
}

var createStateRetriever = func(channelID string, gossipService gossipapi.GossipService) stateRetriever {
	return appdata.NewRetriever(channelID, gossipService, 1, 1)
}

var jsonUnmarshal = func(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
