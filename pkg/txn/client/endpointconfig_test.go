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
	txnmocks "github.com/trustbloc/fabric-peer-ext/pkg/txn/mocks"
)

//go:generate counterfeiter -o ./mocks/endpointconfig.gen.go --fake-name EndpointConfig github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab.EndpointConfig

func TestNewEndpointConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		epCfg := &mocks.EndpointConfig{}
		epCfg.PeerConfigReturns(&fab.PeerConfig{}, true)

		peerCfg := &txnmocks.PeerConfig{}
		peerCfg.TLSCertPathReturns("./testdata/tls.crt")
		peerCfg.MSPIDReturns("org1MSP")
		peerCfg.PeerAddressReturns("peer0.org1.com:7051")

		cfg, err := newEndpointConfig(epCfg, peerCfg)
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})

	t.Run("No TLS cert -> error", func(t *testing.T) {
		cfg, err := newEndpointConfig(&mocks.EndpointConfig{}, &txnmocks.PeerConfig{})
		require.EqualError(t, err, "no TLS cert path specified")
		require.Nil(t, cfg)
	})

	t.Run("Invalid TLS cert file -> error", func(t *testing.T) {
		peerCfg := &txnmocks.PeerConfig{}
		peerCfg.TLSCertPathReturns("./invalid.crt")

		cfg, err := newEndpointConfig(&mocks.EndpointConfig{}, peerCfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cert fixture missing at path")
		require.Nil(t, cfg)
	})

	t.Run("No MSP -> error", func(t *testing.T) {
		epCfg := &mocks.EndpointConfig{}

		peerCfg := &txnmocks.PeerConfig{}
		peerCfg.TLSCertPathReturns("./testdata/tls.crt")

		cfg, err := newEndpointConfig(epCfg, peerCfg)
		require.EqualError(t, err, "MSP ID not defined")
		require.Nil(t, cfg)
	})

	t.Run("Peer address not defined -> error", func(t *testing.T) {
		epCfg := &mocks.EndpointConfig{}
		peerCfg := &txnmocks.PeerConfig{}
		peerCfg.TLSCertPathReturns("./testdata/tls.crt")
		peerCfg.MSPIDReturns("org1MSP")

		cfg, err := newEndpointConfig(epCfg, peerCfg)
		require.EqualError(t, err, "peer address not defined")
		require.Nil(t, cfg)
	})

	t.Run("Invalid peer address -> error", func(t *testing.T) {
		epCfg := &mocks.EndpointConfig{}
		peerCfg := &txnmocks.PeerConfig{}
		peerCfg.TLSCertPathReturns("./testdata/tls.crt")
		peerCfg.MSPIDReturns("org1MSP")
		peerCfg.PeerAddressReturns("peer0.org1.com:xxxx")

		cfg, err := newEndpointConfig(epCfg, peerCfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid port in peer address")
		require.Nil(t, cfg)
	})

	t.Run("Peer not found -> error", func(t *testing.T) {
		epCfg := &mocks.EndpointConfig{}
		epCfg.PeerConfigReturns(nil, false)

		peerCfg := &txnmocks.PeerConfig{}
		peerCfg.TLSCertPathReturns("./testdata/tls.crt")
		peerCfg.MSPIDReturns("org1MSP")
		peerCfg.PeerAddressReturns("peer0.org1.com:7051")

		cfg, err := newEndpointConfig(epCfg, peerCfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "could not find channel peer for")
		require.Nil(t, cfg)
	})
}

func TestEndpointConfig_ChannelPeers(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		epCfg := &mocks.EndpointConfig{}
		epCfg.PeerConfigReturns(&fab.PeerConfig{}, true)

		peerCfg := &txnmocks.PeerConfig{}
		peerCfg.TLSCertPathReturns("./testdata/tls.crt")
		peerCfg.MSPIDReturns("org1MSP")
		peerCfg.PeerAddressReturns("peer0.org1.com:7051")

		cfg, err := newEndpointConfig(epCfg, peerCfg)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		peers := cfg.ChannelPeers("")
		require.Len(t, peers, 1)
	})
}
