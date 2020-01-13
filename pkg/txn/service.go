/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package txn

import (
	"encoding/json"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/client"
)

const (
	configApp     = "txn"
	configVersion = "1"

	generalConfigComponent = "general"
	generalConfigVersion   = "1"

	sdkConfigComponent = "sdk"
	sdkConfigVersion   = "1"
)

type txnClient interface {
	Endorse(req *api.Request) (*channel.Response, error)
	EndorseAndCommit(req *api.Request) (*channel.Response, error)
}

// Service implements a Transaction service that gathers multiple endorsements (according to chaincode policy) and
// (optionally) sends the transaction to the Orderer.
type Service struct {
	channelID string
	c         txnClient
}

// New returns a new transaction service
func New(channelID string, peerConfig api.PeerConfig, configService config.Service) (*Service, error) {
	logger.Debugf("[%s] Creating TXN client", channelID)

	txnCfg, err := getTxnConfig(configService, peerConfig.MSPID())
	if err != nil {
		return nil, err
	}

	sdkCfg, err := getSDKConfig(configService, peerConfig.MSPID())
	if err != nil {
		return nil, err
	}

	c, err := newClient(channelID, txnCfg.User, peerConfig, []byte(sdkCfg.Config), sdkCfg.Format)
	if err != nil {
		return nil, err
	}

	logger.Debugf("[%s] Successfully created TXN client", channelID)

	return &Service{
		channelID: channelID,
		c:         c,
	}, nil
}

// Endorse collects endorsements according to chaincode policy
func (s *Service) Endorse(req *api.Request) (resp *channel.Response, err error) {
	return s.c.Endorse(req)
}

// EndorseAndCommit collects endorsements (according to chaincode policy) and sends the endorsements to the Orderer
func (s *Service) EndorseAndCommit(req *api.Request) (resp *channel.Response, err error) {
	return s.c.EndorseAndCommit(req)
}

type txnConfig struct {
	User string
}

func getTxnConfig(configService config.Service, mspID string) (*txnConfig, error) {
	txnCfgKey := config.NewComponentKey(mspID, configApp, configVersion, generalConfigComponent, generalConfigVersion)

	txnCfg, err := configService.Get(txnCfgKey)
	if err != nil {
		return nil, errors.WithMessagef(err, "cannot load config for sdkCfgKey %s", txnCfgKey)
	}

	txnConfig := &txnConfig{}
	err = json.Unmarshal([]byte(txnCfg.Config), txnConfig)
	if err != nil {
		return nil, errors.WithMessage(err, "error unmarshalling TXN config")
	}

	return txnConfig, nil
}

func getSDKConfig(configService config.Service, mspID string) (*config.Value, error) {
	sdkCfgKey := config.NewComponentKey(mspID, configApp, configVersion, sdkConfigComponent, sdkConfigVersion)

	sdkCfg, err := configService.Get(sdkCfgKey)
	if err != nil {
		return nil, errors.WithMessagef(err, "cannot load config for sdkCfgKey %s", sdkCfgKey)
	}

	return sdkCfg, nil
}

var newClient = func(channelID, userName string, peerConfig api.PeerConfig, sdkCfgBytes []byte, format config.Format) (txnClient, error) {
	return client.New(channelID, userName, peerConfig, sdkCfgBytes, format)
}
