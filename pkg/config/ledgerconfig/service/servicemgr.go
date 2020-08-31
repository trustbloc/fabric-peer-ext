/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"github.com/bluele/gcache"

	"github.com/hyperledger/fabric/extensions/endorser/api"

	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/state"
	extstatedb "github.com/trustbloc/fabric-peer-ext/pkg/statedb"
)

type stateDBProvider interface {
	StateDBForChannel(channelID string) extstatedb.StateDB
}

// Manager manages a set of configuration services - one per channel
type Manager struct {
	stateDBProvider
	bpProvider       api.BlockPublisherProvider
	serviceByChannel gcache.Cache
}

// NewSvcMgr creates a new config service manager
func NewSvcMgr(stateDBProvider stateDBProvider, blockPublisherProvider api.BlockPublisherProvider) *Manager {
	logger.Infof("Creating configuration service manager")

	m := &Manager{
		stateDBProvider: stateDBProvider,
		bpProvider:      blockPublisherProvider,
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
		state.NewQERetrieverProvider(c.StateDBForChannel(channelID)),
		c.bpProvider.ForChannel(channelID),
	)
}
