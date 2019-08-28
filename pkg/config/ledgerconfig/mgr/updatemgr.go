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

// UpdateManager allows you to query and update ledger configuration
type UpdateManager struct {
	*QueryManager
	storeProvider state.StoreProvider
}

// NewUpdateManager returns a new configuration update manager
func NewUpdateManager(namespace string, sp state.StoreProvider) *UpdateManager {
	return &UpdateManager{
		QueryManager:  NewQueryManager(namespace, sp),
		storeProvider: sp,
	}
}

// Save saves the configuration to the ledger
func (m *UpdateManager) Save(txID string, cfg *config.Config) error {
	configMap, err := newKeyValueMap(cfg, txID)
	if err != nil {
		return err
	}
	return m.save(configMap)
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

func addIndex(store state.StateStore, ns string, key config.Key) error {
	indexKey := getIndexKey(MarshalKey(&key), []string{key.MspID})
	logger.Debugf("Adding index [%s]", indexKey)
	err := store.PutState(ns, indexKey, []byte("{}"))
	if err != nil {
		return errors.WithMessage(err, "failed to create index")
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
