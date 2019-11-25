/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package state

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	ccapi "github.com/hyperledger/fabric/extensions/chaincode/api"
	"github.com/hyperledger/fabric/extensions/gossip/api"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

var logger = flogging.MustGetLogger("ext_gossip_state")

type blockPublisherProvider interface {
	ForChannel(cid string) api.BlockPublisher
}

type ccEventMgrProvider interface {
	GetMgr() ccapi.EventMgr
}

// UpdateHandler handles updates to LSCC state
type UpdateHandler struct {
	bpProvider  blockPublisherProvider
	mgrProvider ccEventMgrProvider
}

// NewUpdateHandler returns a new state update handler
func NewUpdateHandler(bpProvider blockPublisherProvider, mgrProvider ccEventMgrProvider) *UpdateHandler {
	return &UpdateHandler{
		bpProvider:  bpProvider,
		mgrProvider: mgrProvider,
	}
}

// ChannelJoined is called when a peer joins a channel
func (h *UpdateHandler) ChannelJoined(channelID string) {
	if roles.IsCommitter() {
		logger.Debugf("[%s] Not adding LSCC write handler since I'm a committer", channelID)
		return
	}

	logger.Debugf("[%s] Adding LSCC write handler", channelID)
	h.bpProvider.ForChannel(channelID).AddLSCCWriteHandler(
		func(txMetadata api.TxMetadata, chaincodeName string, ccData *ccprovider.ChaincodeData, _ *common.CollectionConfigPackage) error {
			logger.Debugf("[%s] Got LSCC write event for [%s].", channelID, chaincodeName)
			return h.handleStateUpdate(txMetadata.ChannelID, chaincodeName, ccData)
		},
	)
}

func (h *UpdateHandler) handleStateUpdate(channelID string, chaincodeName string, ccData *ccprovider.ChaincodeData) error {
	// Chaincode instantiate/upgrade is not logged on committing peer anywhere else.  This is a good place to log it.
	logger.Debugf("[%s] Handling LSCC state update for chaincode [%s]", channelID, chaincodeName)

	chaincodeDefs := []*ccapi.Definition{
		{Name: ccData.CCName(), Version: ccData.CCVersion(), Hash: ccData.Hash()},
	}

	mgr := h.mgrProvider.GetMgr()

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
