/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package txn

import (
	"encoding/json"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
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

// Service implements a Transaction service that gathers multiple endorsements (according to chaincode policy) and
// (optionally) sends the transaction to the Orderer.
type Service struct {
	channelID string
	c         client.ChannelClient
}

// New returns a new transaction service
func New(channelID string, peerConfig api.PeerConfig, configService config.Service) (*Service, error) {
	logger.Debugf("[%s] Creating TXN service", channelID)

	txnCfg, err := getTxnConfig(configService, peerConfig)
	if err != nil {
		return nil, err
	}

	sdkCfg, err := getSDKConfig(configService, peerConfig)
	if err != nil {
		return nil, err
	}

	c, err := newClient(channelID, txnCfg.User, peerConfig, []byte(sdkCfg.Config), sdkCfg.Format)
	if err != nil {
		return nil, err
	}

	return &Service{
		channelID: channelID,
		c:         c,
	}, nil
}

// Endorse collects endorsements according to chaincode policy
func (s *Service) Endorse(req *api.Request) (*channel.Response, error) {
	var fcn string
	if len(req.Args) > 0 {
		fcn = string(req.Args[0])
	}

	resp, err := s.c.Query(channel.Request{
		ChaincodeID:     req.ChaincodeID,
		Fcn:             fcn,
		Args:            req.Args[1:],
		TransientMap:    req.TransientData,
		InvocationChain: asInvocationChain(req.InvocationChain),
	})
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// EndorseAndCommit collects endorsements (according to chaincode policy) and sends the endorsements to the Orderer
func (s *Service) EndorseAndCommit(req *api.Request) (*channel.Response, error) {
	var fcn string
	if len(req.Args) > 0 {
		fcn = string(req.Args[0])
	}

	resp, err := s.c.Execute(channel.Request{
		ChaincodeID:     req.ChaincodeID,
		Fcn:             fcn,
		Args:            req.Args[1:],
		TransientMap:    req.TransientData,
		InvocationChain: asInvocationChain(req.InvocationChain),
	})
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

type closable interface {
	Close()
}

// Close releases the resources for this service
func (s *Service) Close() {
	closableClient, ok := s.c.(closable)
	if ok {
		logger.Debugf("[%s] Closing client", s.channelID)
		closableClient.Close()
	}
}

type txnConfig struct {
	User string
}

func getTxnConfig(configService config.Service, peerConfig api.PeerConfig) (*txnConfig, error) {
	txnCfgKey := config.NewPeerComponentKey(peerConfig.MSPID(), peerConfig.PeerID(), configApp, configVersion, generalConfigComponent, generalConfigVersion)

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

func getSDKConfig(configService config.Service, peerConfig api.PeerConfig) (*config.Value, error) {
	sdkCfgKey := config.NewPeerComponentKey(peerConfig.MSPID(), peerConfig.PeerID(), configApp, configVersion, sdkConfigComponent, sdkConfigVersion)

	sdkCfg, err := configService.Get(sdkCfgKey)
	if err != nil {
		return nil, errors.WithMessagef(err, "cannot load config for sdkCfgKey %s", sdkCfgKey)
	}

	return sdkCfg, nil
}

func asInvocationChain(chain []*api.ChaincodeCall) []*fab.ChaincodeCall {
	invocationChain := make([]*fab.ChaincodeCall, len(chain))
	for i, call := range chain {
		invocationChain[i] = &fab.ChaincodeCall{
			ID:          call.ChaincodeName,
			Collections: call.Collections,
		}
	}
	return invocationChain
}

var newClient = func(channelID, userName string, peerConfig api.PeerConfig, sdkCfgBytes []byte, format config.Format) (client.ChannelClient, error) {
	return client.New(channelID, userName, peerConfig, sdkCfgBytes, format)
}
