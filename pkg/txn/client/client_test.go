/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"io/ioutil"
	"testing"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/api"
	clientmocks "github.com/trustbloc/fabric-peer-ext/pkg/txn/client/mocks"
)

//go:generate counterfeiter -o ./mocks/channelclient.gen.go --fake-name ChannelClient . ChannelClient
//go:generate counterfeiter -o ../../mocks/peerconfig.gen.go --fake-name PeerConfig ../api PeerConfig

func TestNew(t *testing.T) {
	peerCfg := &mocks.PeerConfig{}
	peerCfg.TLSCertPathReturns("./testdata/tls.crt")
	peerCfg.MSPIDReturns("Org1MSP")
	peerCfg.PeerAddressReturns("peer0.org1.com:7051")

	sdkCfgBytes, err := ioutil.ReadFile("./testdata/sdk-config.yaml")
	require.NoError(t, err)

	b := &clientmocks.BCCSP{}
	bKey := &clientmocks.BCCSPKey{}
	bKey.PrivateReturns(true)
	b.GetKeyReturns(bKey, nil)
	b.KeyImportReturns(bKey, nil)

	restoreNewCryptoSuite := newCryptoSuite
	newCryptoSuite = func(bccsp.BCCSP) *cryptoSuite {
		return &cryptoSuite{bccsp: b}
	}
	defer func() { newCryptoSuite = restoreNewCryptoSuite }()

	newChannelClient = func(channelID, userName, org string, sdk *fabsdk.FabricSDK) (client ChannelClient, err error) {
		chClient := &clientmocks.ChannelClient{}
		return chClient, nil
	}

	t.Run("success", func(t *testing.T) {
		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.NoError(t, err)
		require.NotNil(t, c)
		require.NotPanics(t, c.Close)
	})

	t.Run("Invalid SDK config -> error", func(t *testing.T) {
		c, err := New("channel1", "User1", peerCfg, []byte("sdk config"), "YAML")
		require.Error(t, err)
		require.Contains(t, err.Error(), "unmarshal errors")
		require.Nil(t, c)

		bytes, err := ioutil.ReadFile("./testdata/sdk-config-invalid.yaml")
		require.NoError(t, err)

		c, err = New("channel1", "User1", peerCfg, bytes, "YAML")
		require.Error(t, err)
		require.Contains(t, err.Error(), "org not configured for MSP")
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
		newChannelClient = func(channelID, userName, org string, sdk *fabsdk.FabricSDK) (client ChannelClient, err error) {
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

func TestClient_Query(t *testing.T) {
	peerCfg := &mocks.PeerConfig{}
	peerCfg.TLSCertPathReturns("./testdata/tls.crt")
	peerCfg.MSPIDReturns("Org1MSP")
	peerCfg.PeerAddressReturns("peer0.org1.com:7051")

	sdkCfgBytes, err := ioutil.ReadFile("./testdata/sdk-config.yaml")
	require.NoError(t, err)

	b := &clientmocks.BCCSP{}
	bKey := &clientmocks.BCCSPKey{}
	bKey.PrivateReturns(true)
	b.GetKeyReturns(bKey, nil)
	b.KeyImportReturns(bKey, nil)

	restoreNewCryptoSuite := newCryptoSuite
	newCryptoSuite = func(bccsp.BCCSP) *cryptoSuite {
		return &cryptoSuite{bccsp: b}
	}
	defer func() { newCryptoSuite = restoreNewCryptoSuite }()

	newChannelClient = func(channelID, userName, org string, sdk *fabsdk.FabricSDK) (client ChannelClient, err error) {
		chClient := &clientmocks.ChannelClient{}
		return chClient, nil
	}

	t.Run("Query -> success", func(t *testing.T) {
		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.NoError(t, err)
		require.NotNil(t, c)

		req := channel.Request{}
		_, err = c.Query(req)
		require.NoError(t, err)
	})

	t.Run("Query on closed client -> error", func(t *testing.T) {
		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.NoError(t, err)
		require.NotNil(t, c)
		c.Close()

		req := channel.Request{}
		_, err = c.Query(req)
		require.EqualError(t, err, "attempt to increment count on closed resource")
	})

	t.Run("Execute -> success", func(t *testing.T) {
		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.NoError(t, err)
		require.NotNil(t, c)

		req := channel.Request{}
		_, err = c.InvokeHandler(&mockHandler{}, req)
		require.NoError(t, err)
	})

	t.Run("Execute on closed client -> error", func(t *testing.T) {
		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.NoError(t, err)
		require.NotNil(t, c)
		c.Close()

		req := channel.Request{}
		_, err = c.InvokeHandler(&mockHandler{}, req)
		require.EqualError(t, err, "attempt to increment count on closed resource")
	})

	t.Run("Decrement counter -> error", func(t *testing.T) {
		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.NoError(t, err)
		require.NotNil(t, c)
		require.NotPanics(t, c.decrementCounter)
	})
}

type mockHandler struct {
}

func (c *mockHandler) Handle(*invoke.RequestContext, *invoke.ClientContext) {
}
