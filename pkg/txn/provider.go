/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package txn

import (
	"github.com/bluele/gcache"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/api"
)

var logger = flogging.MustGetLogger("ext_txn")

// Provider is a transaction service provider
type Provider struct {
	services gcache.Cache
}

type configServiceProvider interface {
	ForChannel(channelID string) config.Service
}

// NewProvider returns a new transaction service provider
func NewProvider(configProvider configServiceProvider, peerConfig api.PeerConfig) *Provider {
	logger.Debug("Creating transaction service provider")

	return &Provider{
		services: gcache.New(0).LoaderFunc(func(chID interface{}) (interface{}, error) {
			channelID := chID.(string)
			return New(channelID, peerConfig, configProvider.ForChannel(channelID))
		}).Build(),
	}
}

// ForChannel returns the transaction service for the given channel
func (p *Provider) ForChannel(channelID string) (api.Service, error) {
	svc, err := p.services.Get(channelID)
	if err != nil {
		return nil, err
	}

	return svc.(api.Service), nil
}
