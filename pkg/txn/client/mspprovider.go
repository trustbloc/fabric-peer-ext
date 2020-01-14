/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	coreApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"
	mspApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/factory/defmsp"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/provider/msppvdr"
)

type mspPkg struct {
	defmsp.ProviderFactory
	CryptoPath string
}

func newMSPPkg(cryptoPath string) *mspPkg {
	return &mspPkg{
		CryptoPath: cryptoPath,
	}
}

// CreateIdentityManagerProvider returns a new custom implementation of msp provider
func (m *mspPkg) CreateIdentityManagerProvider(config fabApi.EndpointConfig, cryptoProvider coreApi.CryptoSuite, _ mspApi.UserStore) (msp.IdentityManagerProvider, error) {
	return &mspProvider{
		config:         config,
		cryptoProvider: cryptoProvider,
		cryptoPath:     m.CryptoPath,
	}, nil
}

type mspProvider struct {
	msppvdr.MSPProvider
	config         fabApi.EndpointConfig
	cryptoProvider coreApi.CryptoSuite
	cryptoPath     string
}

// IdentityManager returns the organization's identity manager
func (p *mspProvider) IdentityManager(orgName string) (mspApi.IdentityManager, bool) {
	identityMgr, err := newIdentityManager(orgName, p.cryptoProvider, p.config, p.cryptoPath)
	if err != nil {
		return nil, false
	}
	return identityMgr, true
}
