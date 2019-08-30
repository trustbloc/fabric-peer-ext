/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
)

// ConfigManager manages configuration in ledger
type ConfigManager struct {
	queryResults map[config.Criteria][]*config.KeyValue
	err          error
}

// NewConfigMgr returns a new configuration manager
func NewConfigMgr() *ConfigManager {
	return &ConfigManager{
		queryResults: make(map[config.Criteria][]*config.KeyValue),
	}
}

// WithError injects an error
func (m *ConfigManager) WithError(err error) *ConfigManager {
	m.err = err
	return m
}

// WithQueryResults sets the results for the given key
func (m *ConfigManager) WithQueryResults(criteria *config.Criteria, results []*config.KeyValue) *ConfigManager {
	m.queryResults[*criteria] = results
	return m
}

// Query retrieves configuration based on the provided config key.
func (m *ConfigManager) Query(criteria *config.Criteria) ([]*config.KeyValue, error) {
	return m.queryResults[*criteria], m.err
}

// Save saves the configuration to the ledger. The submitted payload should be in form of Config
func (m *ConfigManager) Save(txID string, config *config.Config) error {
	return m.err
}

// Delete deletes configuration from the ledger.
func (m *ConfigManager) Delete(key *config.Criteria) error {
	return m.err
}
