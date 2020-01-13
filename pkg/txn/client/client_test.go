/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"io/ioutil"
	"testing"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/client/mocks"
	txnmocks "github.com/trustbloc/fabric-peer-ext/pkg/txn/mocks"
)

//go:generate counterfeiter -o ./mocks/channelclient.gen.go --fake-name ChannelClient . channelClient

func TestNew(t *testing.T) {
	peerCfg := &txnmocks.PeerConfig{}
	peerCfg.TLSCertPathReturns("./testdata/tls.crt")
	peerCfg.MSPIDReturns("Org1MSP")
	peerCfg.PeerAddressReturns("peer0.org1.com:7051")

	sdkCfgBytes, err := ioutil.ReadFile("./testdata/sdk-config.yaml")
	require.NoError(t, err)

	b := &mocks.BCCSP{}
	bKey := &mocks.BCCSPKey{}
	bKey.PrivateReturns(true)
	b.GetKeyReturns(bKey, nil)
	b.KeyImportReturns(bKey, nil)

	restoreNewCryptoSuite := newCryptoSuite
	newCryptoSuite = func(bccsp.BCCSP) *cryptoSuite {
		return &cryptoSuite{bccsp: b}
	}
	defer func() { newCryptoSuite = restoreNewCryptoSuite }()

	newChannelClient = func(channelID, userName, org string, sdk *fabsdk.FabricSDK) (client channelClient, err error) {
		chClient := &mocks.ChannelClient{}
		return chClient, nil
	}

	t.Run("success", func(t *testing.T) {
		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.NoError(t, err)
		require.NotNil(t, c)
	})

	t.Run("Invalid SDK config -> error", func(t *testing.T) {
		c, err := New("channel1", "User1", peerCfg, []byte("sdk config"), "YAML")
		require.Error(t, err)
		require.Nil(t, c)
	})

	t.Run("Invalid org -> error", func(t *testing.T) {
		peerCfg.MSPIDReturns("invalid-msp")
		defer func() {
			peerCfg.MSPIDReturns("Org1MSP")
		}()

		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.Error(t, err)
		require.Contains(t, err.Error(), "org not configured for MSP")
		require.Nil(t, c)
	})

	t.Run("newSDK -> error", func(t *testing.T) {
		errExpected := errors.New("injected SDK error")
		restoreNewSDK := newSDK
		newSDK = func(channelID string, configProvider core.ConfigProvider, config fab.EndpointConfig, peerCfg api.PeerConfig) (sdk *fabsdk.FabricSDK, err error) {
			return nil, errExpected
		}
		defer func() {
			newSDK = restoreNewSDK
		}()

		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.Nil(t, c)
	})

	t.Run("newEndpointConfig -> error", func(t *testing.T) {
		errExpected := errors.New("injected endpoint config error")
		restoreNewEndpointConfig := newEndpointConfig
		newEndpointConfig = func(cfg fab.EndpointConfig, peerCfg api.PeerConfig) (config *endpointConfig, err error) {
			return nil, errExpected
		}
		defer func() {
			newEndpointConfig = restoreNewEndpointConfig
		}()

		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.Nil(t, c)
	})

	t.Run("newChannelClient -> error", func(t *testing.T) {
		errExpected := errors.New("injected channel client error")
		restoreNewChannelClient := newChannelClient
		newChannelClient = func(channelID, userName, org string, sdk *fabsdk.FabricSDK) (client channelClient, err error) {
			return nil, errExpected
		}
		defer func() {
			newChannelClient = restoreNewChannelClient
		}()

		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.Nil(t, c)
	})
}

func TestClient_EndorseAndCommit(t *testing.T) {
	peerCfg := &txnmocks.PeerConfig{}
	peerCfg.TLSCertPathReturns("./testdata/tls.crt")
	peerCfg.MSPIDReturns("Org1MSP")
	peerCfg.PeerAddressReturns("peer0.org1.com:7051")

	sdkCfgBytes, err := ioutil.ReadFile("./testdata/sdk-config.yaml")
	require.NoError(t, err)

	b := &mocks.BCCSP{}
	bKey := &mocks.BCCSPKey{}
	bKey.PrivateReturns(true)
	b.GetKeyReturns(bKey, nil)
	b.KeyImportReturns(bKey, nil)

	restoreNewCryptoSuite := newCryptoSuite
	newCryptoSuite = func(bccsp.BCCSP) *cryptoSuite {
		return &cryptoSuite{bccsp: b}
	}
	defer func() { newCryptoSuite = restoreNewCryptoSuite }()

	chClient := &mocks.ChannelClient{}
	newChannelClient = func(channelID, userName, org string, sdk *fabsdk.FabricSDK) (client channelClient, err error) {
		return chClient, nil
	}

	c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
	require.NoError(t, err)
	require.NotNil(t, c)

	t.Run("Endorse -> success", func(t *testing.T) {
		req := &api.Request{
			Args: [][]byte{[]byte("arg1")},
		}
		resp, err := c.Endorse(req)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("Endorse -> error", func(t *testing.T) {
		errExpected := errors.New("injected query error")
		chClient.QueryReturns(channel.Response{}, errExpected)

		defer func() {
			chClient.QueryReturns(channel.Response{}, nil)
		}()

		req := &api.Request{
			Args: [][]byte{[]byte("arg1")},
		}
		resp, err := c.Endorse(req)
		require.EqualError(t, err, errExpected.Error())
		require.Nil(t, resp)
	})

	t.Run("EndorseAndCommit -> success", func(t *testing.T) {
		req := &api.Request{
			Args: [][]byte{[]byte("arg1")},
		}
		resp, err := c.EndorseAndCommit(req)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("EndorseAndCommit -> error", func(t *testing.T) {
		errExpected := errors.New("injected query error")
		chClient.ExecuteReturns(channel.Response{}, errExpected)

		defer func() {
			chClient.QueryReturns(channel.Response{}, nil)
		}()

		req := &api.Request{
			Args: [][]byte{[]byte("arg1")},
		}
		resp, err := c.EndorseAndCommit(req)
		require.EqualError(t, err, errExpected.Error())
		require.Nil(t, resp)
	})
}
