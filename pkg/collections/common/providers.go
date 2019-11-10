/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger/fabric/msp"
)

//go:generate counterfeiter -o ../../mocks/identitydeserializerprovider.gen.go --fake-name IdentityDeserializerProvider . IdentityDeserializerProvider
//go:generate counterfeiter -o ../../mocks/identitydeserializer.gen.go --fake-name IdentityDeserializer github.com/hyperledger/fabric/msp.IdentityDeserializer

// IdentityDeserializerProvider returns the identity deserializer for the given channel
type IdentityDeserializerProvider interface {
	GetIdentityDeserializer(channelID string) msp.IdentityDeserializer
}
