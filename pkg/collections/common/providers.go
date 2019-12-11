/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger/fabric/core/ledger"
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	"github.com/hyperledger/fabric/extensions/collections/api/support"
	"github.com/hyperledger/fabric/extensions/endorser/api"
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	"github.com/hyperledger/fabric/msp"
)

//go:generate counterfeiter -o ../../mocks/identitydeserializerprovider.gen.go --fake-name IdentityDeserializerProvider . IdentityDeserializerProvider
//go:generate counterfeiter -o ../../mocks/identitydeserializer.gen.go --fake-name IdentityDeserializer github.com/hyperledger/fabric/msp.IdentityDeserializer
//go:generate counterfeiter -o ../../mocks/signingidentity.gen.go --fake-name SigningIdentity github.com/hyperledger/fabric/msp.SigningIdentity
//go:generate counterfeiter -o ../../mocks/identityprovider.gen.go --fake-name IdentityProvider . IdentityProvider
//go:generate counterfeiter -o ../../mocks/ledgerprovider.gen.go --fake-name LedgerProvider . LedgerProvider
//go:generate counterfeiter -o ../../mocks/ledger.gen.go --fake-name Ledger github.com/hyperledger/fabric/core/ledger.PeerLedger
//go:generate counterfeiter -o ../../mocks/identifierprovider.gen.go --fake-name IdentifierProvider . IdentifierProvider
//go:generate counterfeiter -o ../../mocks/storeprovider.gen.go --fake-name StoreProvider . StoreProvider
//go:generate counterfeiter -o ../../mocks/collconfigprovider.gen.go --fake-name CollectionConfigProvider . CollectionConfigProvider
//go:generate counterfeiter -o ../../mocks/gossipprovider.gen.go --fake-name GossipProvider . GossipProvider

// LedgerProvider retrieves ledgers by channel ID
type LedgerProvider interface {
	GetLedger(cid string) ledger.PeerLedger
}

// IdentityDeserializerProvider returns the identity deserializer for the given channel
type IdentityDeserializerProvider interface {
	GetIdentityDeserializer(channelID string) msp.IdentityDeserializer
}

// IdentifierProvider is the signing identity provider
type IdentifierProvider interface {
	GetIdentifier() (string, error)
}

// IdentityProvider is the signing identity provider
type IdentityProvider interface {
	GetDefaultSigningIdentity() (msp.SigningIdentity, error)
}

// StoreProvider provides stores by channel
type StoreProvider interface {
	StoreForChannel(channelID string) storeapi.Store
}

// CollectionConfigProvider provides collection config retrievers by channel
type CollectionConfigProvider interface {
	ForChannel(channelID string) support.CollectionConfigRetriever
}

// GossipProvider is a Gossip service provider
type GossipProvider interface {
	GetGossipService() gossipapi.GossipService
}

// Providers holds all of the dependencies required by the retriever provider
type Providers struct {
	BlockPublisherProvider api.BlockPublisherProvider
	StoreProvider          StoreProvider
	GossipProvider         GossipProvider
	CCProvider             CollectionConfigProvider
	IdentifierProvider     IdentifierProvider
}
