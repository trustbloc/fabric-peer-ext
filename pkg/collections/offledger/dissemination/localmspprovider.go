/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dissemination

import (
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/common"
)

// LocalMSPProvider provides the local peer's MSP ID
var LocalMSPProvider = &localMSPProvider{}

type localMSPProvider struct {
	identifierProvider common.IdentifierProvider
}

// Initialize initialized the local MSP provider with the identifier provider
func (p *localMSPProvider) Initialize(provider common.IdentifierProvider) *localMSPProvider {
	logger.Infof("Initializing local MSP provider")

	p.identifierProvider = provider

	return p
}

// LocalMSP returns the MSP of the local peer
func (p *localMSPProvider) LocalMSP() (string, error) {
	return p.identifierProvider.GetIdentifier()
}
