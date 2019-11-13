/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package retriever

import (
	"github.com/hyperledger/fabric/core/ledger"
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	supportapi "github.com/hyperledger/fabric/extensions/collections/api/support"
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	gapi "github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/comm"
	gcommon "github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/msp/mgmt"
	gproto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/common"
	extretriever "github.com/trustbloc/fabric-peer-ext/pkg/collections/retriever"
	tretriever "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/retriever"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/support"
)

// NewProvider returns a new private data Retriever provider
func NewProvider(
	storeProvider func(channelID string) storeapi.Store,
	ledgerProvider func(channelID string) ledger.PeerLedger,
	gossipProvider func() supportapi.GossipAdapter,
	blockPublisherProvider func(channelID string) gossipapi.BlockPublisher) storeapi.Provider {

	p := extretriever.NewProvider()

	// For now, explicitly call Initialize. After the retriever is registered as a resource,
	// this code should be removed since Initialize will be called by the resource manager.
	providers := &common.Providers{
		BlockPublisherProvider: &bpProvider{getBlockPublisher: blockPublisherProvider},
		StoreProvider:          &strProvider{getStoreProvider: storeProvider},
		GossipAdapter:          &gsspProvider{getGossip: gossipProvider},
		CCProvider:             &ccProvider{},
		IdentifierProvider:     mgmt.GetLocalMSP(),
	}
	p.Initialize(tretriever.NewProvider(providers), extretriever.NewOffLedgerProvider(providers))

	return p
}

// The code below is temporarily added to facilitate the migration to dependency injected resources.
// This code should be removed once all resources have been converted to use dependency injection.

type bpProvider struct {
	getBlockPublisher func(channelID string) gossipapi.BlockPublisher
}

func (p *bpProvider) ForChannel(channelID string) gossipapi.BlockPublisher {
	return p.getBlockPublisher(channelID)
}

type strProvider struct {
	getStoreProvider func(channelID string) storeapi.Store
}

func (p *strProvider) StoreForChannel(channelID string) storeapi.Store {
	return p.getStoreProvider(channelID)
}

type gsspProvider struct {
	getGossip func() supportapi.GossipAdapter
}

func (p *gsspProvider) PeersOfChannel(channelID gcommon.ChainID) []discovery.NetworkMember {
	return p.getGossip().PeersOfChannel(channelID)
}
func (p *gsspProvider) SelfMembershipInfo() discovery.NetworkMember {
	return p.getGossip().SelfMembershipInfo()
}

func (p *gsspProvider) IdentityInfo() gapi.PeerIdentitySet {
	return p.getGossip().IdentityInfo()
}

func (p *gsspProvider) Send(msg *gproto.GossipMessage, peers ...*comm.RemotePeer) {
	p.getGossip().Send(msg, peers...)
}

type ccProvider struct {
}

func (p *ccProvider) ForChannel(channelID string) supportapi.CollectionConfigRetriever {
	return support.CollectionConfigRetrieverForChannel(channelID)
}
