/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dcas

import (
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	"github.com/pkg/errors"
)

// Validator is an off-ledger validator that validates the CAS key against the value
func Validator(_, _, _, key string, value []byte) error {
	if value == nil {
		return errors.Errorf("nil value for key [%s]", key)
	}
	expectedKey := GetCASKey(value)
	if key != expectedKey {
		return errors.Errorf("Invalid CAS key [%s] - the key should be the hash of the value", key)
	}
	return nil
}

// Decorator is an off-ledger decorator that ensures the given key is the hash of the value. If the key is not
// specified then it is generated. If the key is provided then it is validated against the value.
func Decorator(key *storeapi.Key, value *storeapi.ExpiringValue) (*storeapi.Key, *storeapi.ExpiringValue, error) {
	dcasKey, err := validateCASKey(key.Key, value.Value)
	if err != nil {
		return nil, nil, err
	}

	// The key needs to be base58 encoded since Fabric doesn't allow
	// certain characters to be used in the key.
	newKey := &storeapi.Key{
		EndorsedAtTxID: key.EndorsedAtTxID,
		Namespace:      key.Namespace,
		Collection:     key.Collection,
		Key:            Base58Encode(dcasKey),
	}

	return newKey, value, nil
}

// KeyDecorator is an off-ledger decorator that ensures the given key is base58 encoded
// since Fabric doesn't allow certain characters to be used in the key.
func KeyDecorator(key *storeapi.Key) (*storeapi.Key, error) {
	return &storeapi.Key{
		EndorsedAtTxID: key.EndorsedAtTxID,
		Namespace:      key.Namespace,
		Collection:     key.Collection,
		Key:            Base58Encode(key.Key),
	}, nil
}

func validateCASKey(key string, value []byte) (string, error) {
	if value == nil {
		return "", errors.Errorf("attempt to put nil value for key [%s]", key)
	}

	casKey := GetCASKey(value)
	if key != "" && key != casKey {
		return casKey, errors.New("invalid CAS key - the key should be the hash of the value")
	}
	return casKey, nil
}
