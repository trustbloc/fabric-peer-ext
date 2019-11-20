/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"github.com/bluele/gcache"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/extensions/endorser/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/common"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/state"
)

// Manager manages a set of configuration services - one per channel
type Manager struct {
	ledgerProvider   common.LedgerProvider
	bpProvider       api.BlockPublisherProvider
	serviceByChannel gcache.Cache
}

// NewSvcMgr creates a new config service manager
func NewSvcMgr(ledgerProvider common.LedgerProvider, blockPublisherProvider api.BlockPublisherProvider) *Manager {
	logger.Infof("Creating configuration service manager")

	m := &Manager{
		ledgerProvider: ledgerProvider,
		bpProvider:     blockPublisherProvider,
	}

	m.serviceByChannel = gcache.New(0).LoaderFunc(func(channelID interface{}) (i interface{}, e error) {
		return m.newService(channelID.(string)), nil
	}).Build()

	return m
}

// ForChannel returns a config service for the given channel.
func (c *Manager) ForChannel(channelID string) config.Service {
	s, err := c.serviceByChannel.Get(channelID)
	if err != nil {
		// This should never happen since the loader func never returns an error
		panic(err)
	}
	return s.(config.Service)
}

func (c *Manager) newService(channelID string) config.Service {
	return New(
		channelID,
		state.NewQERetrieverProvider(channelID, c.newQEProvider(channelID)),
		c.bpProvider.ForChannel(channelID),
	)
}

type qeProvider struct {
	ledger ledger.PeerLedger
}

func (c *Manager) newQEProvider(channelID string) *qeProvider {
	return &qeProvider{ledger: c.ledgerProvider.GetLedger(channelID)}
}

func (p *qeProvider) GetQueryExecutorForLedger(cid string) (ledger.QueryExecutor, error) {
	return p.ledger.NewQueryExecutor()
}
