/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	fabapi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	sdkconfig "github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/api"
)

var logger = flogging.MustGetLogger("ext_txn")

// ChannelClient defines functions to collect endorsements and send them to the orderer
type ChannelClient interface {
	Query(request channel.Request, options ...channel.RequestOption) (channel.Response, error)
	Execute(request channel.Request, options ...channel.RequestOption) (channel.Response, error)
}

// Client holds an SDK client instance
type Client struct {
	ChannelClient
	channelID string
}

// New returns a new instance of an SDK client for the given channel
func New(channelID, userName string, peerConfig api.PeerConfig, sdkCfgBytes []byte, format config.Format) (*Client, error) {
	configProvider, endpointConfig, err := getEndpointConfig(sdkCfgBytes, format)
	if err != nil {
		return nil, err
	}

	org, err := orgFromMSPID(endpointConfig, peerConfig)
	if err != nil {
		return nil, err
	}

	customEndpointConfig, err := newEndpointConfig(endpointConfig, peerConfig)
	if err != nil {
		return nil, err
	}

	sdk, err := newSDK(channelID, configProvider, customEndpointConfig, peerConfig)
	if err != nil {
		return nil, err
	}

	chClient, err := newChannelClient(channelID, userName, org, sdk)
	if err != nil {
		return nil, err
	}

	return &Client{
		ChannelClient: chClient,
		channelID:     channelID,
	}, nil
}

func orgFromMSPID(endpointConfig fabapi.EndpointConfig, peerCfg api.PeerConfig) (string, error) {
	for orgName, org := range endpointConfig.NetworkConfig().Organizations {
		if org.MSPID == peerCfg.MSPID() {
			return orgName, nil
		}
	}

	return "", errors.Errorf("org not configured for MSP [%s]", peerCfg.MSPID())
}

func getEndpointConfig(configBytes []byte, format config.Format) (core.ConfigProvider, fabapi.EndpointConfig, error) {
	configProvider := func() ([]core.ConfigBackend, error) {
		// Make sure the buffer is created each time it is called, otherwise
		// there will be no data left in the buffer the second time it's called
		return sdkconfig.FromRaw(configBytes, string(format))()
	}

	configBackends, err := configProvider()
	if err != nil {
		return nil, nil, err
	}

	endpointConfig, err := fab.ConfigFromBackend(configBackends...)
	if err != nil {
		return nil, nil, err
	}

	return configProvider, endpointConfig, nil
}

var newSDK = func(channelID string, configProvider core.ConfigProvider, config fabapi.EndpointConfig, peerCfg api.PeerConfig) (*fabsdk.FabricSDK, error) {
	sdk, err := fabsdk.New(
		configProvider,
		fabsdk.WithEndpointConfig(config),
		fabsdk.WithCorePkg(newCorePkg()),
		fabsdk.WithMSPPkg(newMSPPkg(peerCfg.MSPConfigPath())),
	)
	if err != nil {
		return nil, errors.WithMessagef(err, "Error creating SDK on channel [%s]", channelID)
	}

	return sdk, nil
}

var newChannelClient = func(channelID, userName, org string, sdk *fabsdk.FabricSDK) (ChannelClient, error) {
	chClient, err := channel.New(sdk.ChannelContext(channelID, fabsdk.WithUser(userName), fabsdk.WithOrg(org)))
	if err != nil {
		return nil, err
	}

	return chClient, nil
}
