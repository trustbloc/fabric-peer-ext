/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"fmt"
)

// Key is used to uniquely identify a specific application configuration and is used as the
// key when persisting to a state store.
type Key struct {
	// MspID is the ID of the MSP that owns the data
	MspID string
	// PeerID is the (optional) ID of the peer with which the data is associated
	PeerID string
	// AppName is the name of the application that owns the data
	AppName string
	// AppVersion is the version of the application config
	AppVersion string
	// ComponentName is the (optional) name of the application component
	ComponentName string
	// ComponentVersion is the (optional) version of the application component config
	ComponentVersion string
}

// String returns a readable string for the key
func (k Key) String() string {
	return fmt.Sprintf("(MSP:%s),(Peer:%s),(Apps:%s),(AppVersion:%s),(Comp:%s),(CompVersion:%s)", k.MspID, k.PeerID, k.AppName, k.AppVersion, k.ComponentName, k.ComponentVersion)
}

// NewPeerKey creates a config key using mspID, peerID, appName and appVersion
func NewPeerKey(mspID, peerID, appName, appVersion string) *Key {
	return &Key{
		MspID:      mspID,
		PeerID:     peerID,
		AppName:    appName,
		AppVersion: appVersion,
	}
}

// NewPeerComponentKey creates a config key using mspID, peerID, appName, appVersion, componentName and componentVersion
func NewPeerComponentKey(mspID, peerID, appName, appVersion, componentName, componentVersion string) *Key {
	return &Key{
		MspID:            mspID,
		PeerID:           peerID,
		AppName:          appName,
		AppVersion:       appVersion,
		ComponentName:    componentName,
		ComponentVersion: componentVersion,
	}
}

// NewAppKey creates a config key using mspID, appName and appVersion
func NewAppKey(mspID, appName, appVersion string) *Key {
	return &Key{
		MspID:      mspID,
		AppName:    appName,
		AppVersion: appVersion,
	}
}

// NewComponentKey creates a config key using mspID, appName, appVersion, componentName and componentVersion
func NewComponentKey(mspID, appName, appVersion, componentName, componentVersion string) *Key {
	return &Key{
		MspID:            mspID,
		AppName:          appName,
		AppVersion:       appVersion,
		ComponentName:    componentName,
		ComponentVersion: componentVersion,
	}
}

// Format specifies the format of the configuration
type Format string

const (
	// FormatYAML indicates that the configuration is in YAML format
	FormatYAML Format = "YAML"

	// FormatJSON indicates that the configuration is in JSON format
	FormatJSON Format = "JSON"

	// FormatOther indicates that the configuration is in an application-specific format
	FormatOther Format = "OTHER"
)

// Value contains the configuration data and is persisted as a JSON document in the store.
type Value struct {
	// TxID is the ID of the transaction in which the config was stored/updated
	TxID string
	// Format describes the format (type) of the config data
	Format Format
	// Config contains the actual configuration
	Config string
}

// NewValue returns a new config Value
func NewValue(txID string, config string, format Format) *Value {
	return &Value{
		TxID:   txID,
		Config: config,
		Format: format,
	}
}

// String returns a readable string for the value
func (v *Value) String() string {
	return fmt.Sprintf("(TxID:%s),(Config:%s),(Format:%s)", v.TxID, v.Config, v.Format)
}

// KeyValue contains the key and the value for the key
type KeyValue struct {
	*Key
	*Value
}

// NewKeyValue returns a new KeyValue
func NewKeyValue(key *Key, value *Value) *KeyValue {
	return &KeyValue{
		Key:   key,
		Value: value,
	}
}

// String returns a readable string for the key-value
func (kv *KeyValue) String() string {
	return fmt.Sprintf("[%s]=[%s]", kv.Key, kv.Value)
}
