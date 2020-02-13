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

// ValidatorForKey returns the first validator that indicates it can perform validation on the given key.
func (r *Registry) ValidatorForKey(key *config.Key) config.Validator {
	r.mutex.RLock()
	validators := r.validators
	r.mutex.RUnlock()

	for _, v := range validators {
		if v.CanValidate(key) {
			return &keyValueValidator{appValidator: v}
		}
	}

	return &keyValueValidator{}
}

// Register registers a configuration validator
func (r *Registry) Register(v config.Validator) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.validators = append(r.validators, v)
}

type keyValueValidator struct {
	appValidator config.Validator
}

// Validate performs basic validation on the key and value and then delegates
// to the application-specific validator (if any).
func (v *keyValueValidator) Validate(key *config.Key, value *config.Value) error {
	logger.Debugf("Validating key %s", key)

	if err := key.Validate(); err != nil {
		return err
	}

	logger.Debugf("Validating value %s", value)

	if value.Format == "" {
		return errors.New("field 'Format' must not be empty")
	}

	if value.Config == "" {
		return errors.New("field 'Config' must not be empty")
	}

	if v.appValidator != nil {
		return v.appValidator.Validate(key, value)
	}

	return nil
}

// CanValidate always returns true
func (v *keyValueValidator) CanValidate(key *config.Key) bool {
	return true
}
