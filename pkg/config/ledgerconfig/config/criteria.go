/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"fmt"

	"github.com/pkg/errors"
)

// Criteria is used for configuration searches
type Criteria struct {
	// MspID is the ID of the MSP that owns the data
	MspID string
	// PeerID is the ID of the peer with which the data is associated
	PeerID string
	// AppName is the name of the application that owns the data
	AppName string
	// AppVersion is the version of the application config
	AppVersion string
	// ComponentName is the name of the application component
	ComponentName string
	// ComponentVersion is the version of the application component config
	ComponentVersion string
}

// String returns a readable string for the Criteria
func (c *Criteria) String() string {
	return fmt.Sprintf("(MSP:%s),(Peer:%s),(App:%s),(AppVersion:%s),(Comp:%s),(CompVersion:%s)", c.MspID, c.PeerID, c.AppName, c.AppVersion, c.ComponentName, c.ComponentVersion)
}

// Validate ensures that the criteria is valid
func (c *Criteria) Validate() error {
	if c.MspID == "" {
		return errors.New("field [MspID] is required")
	}
	if c.AppVersion != "" && c.AppName == "" {
		return errors.New("field [Name] is required")
	}
	if c.ComponentName != "" && c.AppName == "" {
		return errors.New("field [Name] is required")
	}
	if c.ComponentVersion != "" && c.ComponentName == "" {
		return errors.New("field [ComponentName] is required")
	}
	return nil
}

// IsUnique validates that the criteria has all of the necessary parts to uniquely identify the config
func (c *Criteria) IsUnique() bool {
	return c.MspID != "" && (c.isPeerAppKey() || c.isPeerAppComponentKey() || c.isAppKey() || c.isAppComponentKey())
}

// AsKey transforms the Criteria into a Key. If the Criteria is not unique then
// an error is returned (since a key must uniquely identify a config item).
func (c *Criteria) AsKey() (*Key, error) {
	if !c.IsUnique() {
		return nil, errors.New("criteria is not unique")
	}
	return &Key{
		MspID:            c.MspID,
		AppName:          c.AppName,
		PeerID:           c.PeerID,
		AppVersion:       c.AppVersion,
		ComponentName:    c.ComponentName,
		ComponentVersion: c.ComponentVersion,
	}, nil
}

func (c *Criteria) isAppKey() bool {
	return c.AppName != "" && c.AppVersion != "" && c.ComponentName == "" && c.ComponentVersion == ""
}

func (c *Criteria) isAppComponentKey() bool {
	return c.AppName != "" && c.AppVersion != "" && c.ComponentName != "" && c.ComponentVersion != ""
}

func (c *Criteria) isPeerAppKey() bool {
	return c.PeerID != "" && c.AppName != "" && c.AppVersion != "" && c.ComponentName == "" && c.ComponentVersion == ""
}

func (c *Criteria) isPeerAppComponentKey() bool {
	return c.PeerID != "" && c.AppName != "" && c.AppVersion != "" && c.ComponentName != "" && c.ComponentVersion != ""
}
