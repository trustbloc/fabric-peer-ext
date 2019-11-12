/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/msp"
)

//go:generate counterfeiter -o ../../mocks/identitydeserializerprovider.gen.go --fake-name IdentityDeserializerProvider . IdentityDeserializerProvider
//go:generate counterfeiter -o ../../mocks/identitydeserializer.gen.go --fake-name IdentityDeserializer github.com/hyperledger/fabric/msp.IdentityDeserializer
//go:generate counterfeiter -o ../../mocks/ledgerprovider.gen.go --fake-name LedgerProvider . LedgerProvider
//go:generate counterfeiter -o ../../mocks/identifierprovider.gen.go --fake-name IdentifierProvider . IdentifierProvider

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
