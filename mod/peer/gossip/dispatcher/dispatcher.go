/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dispatcher

import (
	"github.com/hyperledger/fabric/core/ledger"
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	supportapi "github.com/hyperledger/fabric/extensions/collections/api/support"
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	gossip "github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/support"
	extdispatcher "github.com/trustbloc/fabric-peer-ext/pkg/gossip/dispatcher"
)

type gossipAdapter interface {
	PeersOfChannel(common.ChainID) []discovery.NetworkMember
	SelfMembershipInfo() discovery.NetworkMember
	IdentityInfo() gossip.PeerIdentitySet
}

type blockPublisher interface {
	AddLSCCWriteHandler(handler gossipapi.LSCCWriteHandler)
}

// New returns a new Gossip message dispatcher
func New(
	channelID string,
	dataStore storeapi.Store,
	gossipAdapter gossipAdapter,
	ledger ledger.PeerLedger,
	blockPublisher blockPublisher) *extdispatcher.Dispatcher {

	p := extdispatcher.NewProvider()

	// For now, explicitly call Initialize. After the dispatcher provider is registered as a resource,
	// this code should be removed since Initialize will be called by the resource manager.
	p.Initialize(gossipAdapter, &ccProvider{})

	return p.ForChannel(channelID, dataStore)
}

// The code below is temporarily added to facilitate the migration to dependency injected resources.
// This code should be removed once all resources have been converted to use dependency injection.

type ccProvider struct {
}

func (p *ccProvider) ForChannel(channelID string) supportapi.CollectionConfigRetriever {
	return support.CollectionConfigRetrieverForChannel(channelID)
}
