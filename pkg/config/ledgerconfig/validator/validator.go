/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator

import (
	"errors"
	"sync"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
)

var logger = flogging.MustGetLogger("ledgerconfig")

// Registry contains a registry of application configuration validators. The validators are registered on startup
// and are invoked before a new config value is persisted.
type Registry struct {
	validators []config.Validator
	mutex      sync.RWMutex
}

// NewRegistry returns a new configuration validator registry
func NewRegistry() *Registry {
	logger.Infof("Creating config validator registry")
	return &Registry{}
}

// Validate invokes the registered validators to validate the key and value.
// An error is returned in the case of invalid config
func (r *Registry) Validate(kv *config.KeyValue) error {
	// Perform basic validation of the key/value
	if err := validate(kv); err != nil {
		return err
	}

	r.mutex.RLock()
	validators := r.validators
	r.mutex.RUnlock()

	for _, v := range validators {
		if err := v.Validate(kv); err != nil {
			return err
		}
	}

	return nil
}

// Register registers a configuration validator
func (r *Registry) Register(v config.Validator) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.validators = append(r.validators, v)
}

func validate(kv *config.KeyValue) error {
	logger.Debugf("Validating key %s", kv.Key)

	if err := kv.Validate(); err != nil {
		return err
	}

	logger.Debugf("Validating value %s", kv.Value)

	if kv.Value.Format == "" {
		return errors.New("field 'Format' must not be empty")
	}

	if kv.Value.Config == "" {
		return errors.New("field 'Config' must not be empty")
	}

	return nil
}
