/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dcas

import (
	"github.com/btcsuite/btcutil/base58"
	"github.com/hyperledger/fabric/common/flogging"
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("ext_offledger")

// Validator is an off-ledger validator that validates the CAS key against the value
func Validator(_, _, _, key string, value []byte) error {
	return validateCASKey(key, value)
}

// Decorator is an off-ledger decorator that ensures the key is the hash of the value.
var Decorator = &decorator{}

type decorator struct {
}

// BeforeSave ensures that the given key is the base58 encoded hash of the value.
func (d *decorator) BeforeSave(key *storeapi.Key, value *storeapi.ExpiringValue) (*storeapi.Key, *storeapi.ExpiringValue, error) {
	if err := validateCASKey(key.Key, value.Value); err != nil {
		return nil, nil, err
	}
	return key, value, nil
}

// BeforeLoad returns the key.
func (d *decorator) BeforeLoad(key *storeapi.Key) (*storeapi.Key, error) {
	return key, nil
}

// AfterQuery returns the key/value.
func (d *decorator) AfterQuery(key *storeapi.Key, value *storeapi.ExpiringValue) (*storeapi.Key, *storeapi.ExpiringValue, error) {
	return key, value, nil
}

func validateCASKey(key string, value []byte) error {
	if value == nil {
		return errors.Errorf("attempt to put nil value for key [%s]", key)
	}

	casKey := base58.Encode(getCASKey(value))
	logger.Debugf("Validating key [%s] against value [%s]", key, value)
	if key != casKey {
		return errors.Errorf("invalid CAS key - the key should be the hash of the value [%s]", casKey)
	}

	return nil
}
