/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"sync"

	"github.com/pkg/errors"
	state "github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/state/api"
)

// Manager manages a set of configuration services - one per channel
type Manager struct {
	serviceByChannel map[string]*ConfigService
	mutex            sync.RWMutex
}

var svcMgr = newSvcMgr()

// GetSvcMgr returns the config service manager
func GetSvcMgr() *Manager {
	return svcMgr
}

func newSvcMgr() *Manager {
	return &Manager{
		serviceByChannel: make(map[string]*ConfigService),
	}
}

// Init initializes the ConfigService for the given channel
func (c *Manager) Init(channelID string, provider state.RetrieverProvider) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, ok := c.serviceByChannel[channelID]; ok {
		return errors.Errorf("Config service already exists for channel [%s]", channelID)
	}

	c.serviceByChannel[channelID] = New(channelID, provider)
	return nil
}

// ForChannel returns a config service for the given channel.
func (c *Manager) ForChannel(channelID string) *ConfigService {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.serviceByChannel[channelID]
}
