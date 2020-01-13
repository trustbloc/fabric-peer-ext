/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"testing"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/client/mocks"
)

func TestMspPkg_CreateIdentityManagerProvider(t *testing.T) {
	epCfg := &mocks.EndpointConfig{}
	netCfg := &fab.NetworkConfig{
		Organizations: map[string]fab.OrganizationConfig{
			"org1msp": {
				Users: make(map[string]fab.CertKeyPair),
			},
		},
	}
	epCfg.NetworkConfigReturns(netCfg)

	cp := &mocks.CryptoSuite{}

	pkg := newMSPPkg("path")
	p, err := pkg.CreateIdentityManagerProvider(epCfg, cp, nil)
	require.NoError(t, err)
	require.NotNil(t, p)

	m, ok := p.IdentityManager("Org1MSP")
	require.NotNil(t, m)
	require.True(t, ok)

	m, ok = p.IdentityManager("")
	require.Nil(t, m)
	require.False(t, ok)
}
