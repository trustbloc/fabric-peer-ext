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
	extretriever "github.com/trustbloc/fabric-peer-ext/pkg/collections/retriever"
)

// NewProvider returns a new private data Retriever provider
func NewProvider(
	storeProvider func(channelID string) storeapi.Store,
	ledgerProvider func(channelID string) ledger.PeerLedger,
	gossipProvider func() supportapi.GossipAdapter,
	blockPublisherProvider func(channelID string) gossipapi.BlockPublisher) storeapi.Provider {

	return extretriever.NewProvider(storeProvider, ledgerProvider, gossipProvider, blockPublisherProvider)
}
