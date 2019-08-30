/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/mgr"
	state "github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/state/api"
)

var logger = flogging.MustGetLogger("ledgerconfig")

const (
	// ConfigNS is the namespace (chaincode name) under which configuration data is stored.
	ConfigNS = "configscc"
)

// ErrConfigNotFound indicates that the config for the given key was not found
var ErrConfigNotFound = errors.New("config not found")

type configMgr interface {
	Query(criteria *config.Criteria) ([]*config.KeyValue, error)
}

// ConfigService manages configuration data for a given channel
type ConfigService struct {
	channelID string
	configMgr configMgr
}

// New returns a new config service
func New(channelID string, retrieverProvider state.RetrieverProvider) *ConfigService {
	return &ConfigService{
		channelID: channelID,
		configMgr: mgr.NewQueryManager(ConfigNS, retrieverProvider),
	}
}

// Get returns the config bytes for the given criteria.
// If the key is not found then ErrConfigNotFound error is returned
func (s *ConfigService) Get(key *config.Key) (*config.Value, error) {
	err := key.Validate()
	if err != nil {
		return nil, err
	}
	results, err := s.configMgr.Query(config.CriteriaFromKey(key))
	if err != nil {
		return nil, err
	}

	logger.Debugf("Received results %s", results)

	if len(results) > 1 {
		return nil, errors.Errorf("received more than one result for key [%s]", key)
	}
	if len(results) == 0 {
		return nil, ErrConfigNotFound
	}

	return results[0].Value, nil
}
