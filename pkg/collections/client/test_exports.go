// +build testing

/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
)

// SetLedgerProvider sets the ledger provider for unit tests
func SetLedgerProvider(provider func(channelID string) PeerLedger) {
	getLedger = provider
}

// SetGossipProvider sets the Gossip provider for unit tests
func SetGossipProvider(provider func() GossipAdapter) {
	getGossipAdapter = provider
}

// SetBlockPublisherProvider sets the block publisher provider for unit tests
func SetBlockPublisherProvider(provider func(channelID string) gossipapi.BlockPublisher) {
	getBlockPublisher = provider
}

// SetCollConfigRetrieverProvider sets the collection config retriever provider for unit tests
func SetCollConfigRetrieverProvider(provider func(_ string, _ PeerLedger, _ gossipapi.BlockPublisher) CollectionConfigRetriever) {
	getCollConfigRetriever = provider
}

// SetCreatorProvider sets the creator provider for unit tests
func SetCreatorProvider(provider func() ([]byte, error)) {
	newCreator = provider
}
