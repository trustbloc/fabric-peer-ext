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
	"github.com/trustbloc/fabric-peer-ext/pkg/common/reference"
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
	refCount  *reference.Counter
	channelID string
	sdk       *fabsdk.FabricSDK
}

// New returns a new instance of an SDK client for the given channel
func New(channelID, userName string, peerConfig api.PeerConfig, sdkCfgBytes []byte, format config.Format) (*Client, error) {
	configProvider, endpointConfig, err := GetEndpointConfig(sdkCfgBytes, format)
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

	c := &Client{
		ChannelClient: chClient,
		channelID:     channelID,
		sdk:           sdk,
	}

	c.refCount = reference.NewCounter(c.close)

	return c, nil
}

// Query sends an endorsement proposal request to one or more peers (according to policy and options) but does not
// send the proposal responses to the orderer.
func (c *Client) Query(request channel.Request, options ...channel.RequestOption) (channel.Response, error) {
	_, err := c.refCount.Increment()
	if err != nil {
		return channel.Response{}, err
	}
	defer c.decrementCounter()

	return c.ChannelClient.Query(request, options...)
}

// Execute sends an endorsement proposal request to one or more peers (according to policy and options) and
// then sends the proposal responses to the orderer.
func (c *Client) Execute(request channel.Request, options ...channel.RequestOption) (channel.Response, error) {
	_, err := c.refCount.Increment()
	if err != nil {
		return channel.Response{}, err
	}
	defer c.decrementCounter()

	return c.ChannelClient.Execute(request, options...)
}

// Close will close the SDK after all references have been released.
func (c *Client) Close() {
	c.refCount.Close()
}

func (c *Client) close() {
	if c.sdk != nil {
		logger.Debugf("[%s] Closing the SDK", c.channelID)
		c.sdk.Close()
	}
}

func (c *Client) decrementCounter() {
	_, err := c.refCount.Decrement()
	if err != nil {
		logger.Warning(err.Error())
	}
}

func orgFromMSPID(endpointConfig fabapi.EndpointConfig, peerCfg api.PeerConfig) (string, error) {
	for orgName, org := range endpointConfig.NetworkConfig().Organizations {
		if org.MSPID == peerCfg.MSPID() {
			return orgName, nil
		}
	}

	return "", errors.Errorf("org not configured for MSP [%s]", peerCfg.MSPID())
}

// GetEndpointConfig unmarshals the given bytes and returns the SDK endpoint config and config provider.
func GetEndpointConfig(configBytes []byte, format config.Format) (core.ConfigProvider, fabapi.EndpointConfig, error) {
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
	return channel.New(sdk.ChannelContext(channelID, fabsdk.WithUser(userName), fabsdk.WithOrg(org)))
}
