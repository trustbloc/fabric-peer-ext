/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

// UpdateHandler handles updates/deletes of config keys
type UpdateHandler func(kv *KeyValue)

// Service defines the operations of a configuration service
type Service interface {
	Get(key *Key) (*Value, error)
	Query(criteria *Criteria) ([]*KeyValue, error)
	AddUpdateHandler(handler UpdateHandler)
}

// Validator validates application-specific configuration
type Validator interface {
	// Validate validates the key and value and returns an error in the case of invalid config
	Validate(key *Key, value *Value) error
	// CanValidate returns true if the validator is able to validate the given config key
	CanValidate(key *Key) bool
}
