/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	"time"
)

// Value is a data value
type Value struct {
	Value      []byte
	TxID       string
	ExpiryTime time.Time
}

// KeyValue is a struct to store a key value pair
type KeyValue struct {
	*Value
	Key string
}

// NewKeyValue returns a new key
func NewKeyValue(key string, value []byte, txID string, expiryTime time.Time) *KeyValue {
	return &KeyValue{
		Key:   key,
		Value: &Value{Value: value, TxID: txID, ExpiryTime: expiryTime},
	}
}

// String returns the string representation of the key
func (k *KeyValue) String() string {
	return k.Key
}

// DB persists collection data.
type DB interface {
	// Put stores the given set of keys/values. If expiry time is 0 then the data lives forever.
	Put(keyVal ...*KeyValue) error

	// Get returns the value for the given key or nil if the key doesn't exist
	Get(key string) (*Value, error)

	// Get returns the values for multiple keys. The values are returned in the same order as the keys.
	GetMultiple(keys ...string) ([]*Value, error)

	// DeleteExpiredKeys deletes all of the expired keys
	DeleteExpiredKeys() error
}

// DBProvider returns the persister for the given namespace/collection
type DBProvider interface {
	// GetDB return the DB for the given namespace/collection
	GetDB(ns, coll string) (DB, error)

	// Close closes the DB provider
	Close()
}
