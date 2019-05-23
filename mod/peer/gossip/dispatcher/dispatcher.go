/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dispatcher

import (
	"github.com/hyperledger/fabric/core/ledger"
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	gossip "github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	extdispatcher "github.com/trustbloc/fabric-peer-ext/pkg/gossip/dispatcher"
)

type gossipAdapter interface {
	PeersOfChannel(common.ChainID) []discovery.NetworkMember
	SelfMembershipInfo() discovery.NetworkMember
	IdentityInfo() gossip.PeerIdentitySet
}

type blockPublisher interface {
	AddCCUpgradeHandler(handler gossipapi.ChaincodeUpgradeHandler)
}

// New returns a new Gossip message dispatcher
func New(
	channelID string,
	dataStore storeapi.Store,
	gossipAdapter gossipAdapter,
	ledger ledger.PeerLedger,
	blockPublisher blockPublisher) *extdispatcher.Dispatcher {
	return extdispatcher.New(channelID, dataStore, gossipAdapter, ledger, blockPublisher)
}
