/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mgr

import (
	"encoding/json"
	"strings"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/compositekey"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
	state "github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/state/api"
)

var logger = flogging.MustGetLogger("ledgerconfig")

const (
	// keyDivider is used to separate key parts
	keyDivider = "!"

	// indexOrg is the name of the index to retrieve configurations per org
	indexMspID = "cfgmgmt-mspid"
)

type validatorRegistry interface {
	ValidatorForKey(key *config.Key) config.Validator
}

// UpdateManager allows you to query and update ledger configuration
type UpdateManager struct {
	*QueryManager
	validatorRegistry
	storeProvider state.StoreProvider
}

// NewUpdateManager returns a new configuration update manager
func NewUpdateManager(namespace string, sp state.StoreProvider, validatorRegistry validatorRegistry) *UpdateManager {
	return &UpdateManager{
		QueryManager:      NewQueryManager(namespace, sp),
		validatorRegistry: validatorRegistry,
		storeProvider:     sp,
	}
}

// Save saves the configuration to the ledger
func (m *UpdateManager) Save(txID string, cfg *config.Config) error {
	configMap, err := newKeyValueMap(cfg, txID)
	if err != nil {
		return err
	}

	err = m.validate(configMap)
	if err != nil {
		logger.Debugf("Received validation error for config %s: %s", cfg, err)
		return errors.WithMessage(err, "validation error")
	}

	return m.save(configMap)
}

// Delete deletes one or more configuration items according to the given Criteria.
func (m *UpdateManager) Delete(key *config.Criteria) error {
	configs, err := m.query(key)
	if err != nil {
		return err
	}

	store, err := m.storeProvider.GetStore()
	if err != nil {
		return err
	}
	defer store.Done()

	for _, value := range configs {
		logger.Debugf("... Deleting key [%s]", value.Key)
		if err := store.DelState(m.namespace, MarshalKey(value.Key)); err != nil {
			return err
		}
		if err := deleteIndex(store, m.namespace, value.Key); err != nil {
			return err
		}
	}
	return nil
}

//save saves keys/values to the repository.
func (m *UpdateManager) save(kvMap keyValueMap) error {
	store, err := m.storeProvider.GetStore()
	if err != nil {
		return err
	}
	defer store.Done()

	for key, value := range kvMap {
		strKey := marshalKey(key)
		valueBytes, err := json.Marshal(value)
		if err != nil {
			return errors.WithMessagef(err, "error marshalling config value")
		}
		logger.Debugf("Saving config [%s]=%s", strKey, valueBytes)
		err = store.PutState(m.namespace, strKey, valueBytes)
		if err != nil {
			return errors.WithMessagef(err, "error saving config [%s]", key)
		}
		if err := addIndex(store, m.namespace, key); err != nil {
			return err
		}
	}
	return nil
}

func (m *UpdateManager) validate(kvMap keyValueMap) error {
	for key, value := range kvMap {
		validator := m.ValidatorForKey(&key)
		if validator == nil {
			return nil
		}

		if err := validator.Validate(&key, value); err != nil {
			return err
		}
	}

	return nil
}

func addIndex(store state.StateStore, ns string, key config.Key) error {
	indexKey := getIndexKey(MarshalKey(&key), []string{key.MspID})
	logger.Debugf("Adding index [%s]", indexKey)
	err := store.PutState(ns, indexKey, []byte("{}"))
	if err != nil {
		return errors.WithMessage(err, "failed to create index")
	}
	return nil
}

func deleteIndex(store state.StateStore, ns string, key *config.Key) error {
	indexKey := getIndexKey(MarshalKey(key), []string{key.MspID})
	logger.Debugf("Deleting index [%s]", indexKey)
	err := store.DelState(ns, indexKey)
	if err != nil {
		return errors.WithMessage(err, "failed to delete index")
	}
	return nil
}

func getIndexKey(key string, fields []string) string {
	return compositekey.Create(indexMspID, append(fields, key))
}

// marshalKey marshals the key into a string that may be used as a key in the state store
func marshalKey(k config.Key) string {
	return strings.Join([]string{k.MspID, k.PeerID, k.AppName, k.AppVersion, k.ComponentName, k.ComponentVersion}, keyDivider)
}
