/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"github.com/pkg/errors"
)

// Config contains zero or more application configurations and zero or more peer-specific application configurations
type Config struct {
	// MspID is the ID of the MSP
	MspID string
	// Peers contains configuration for zero or more peers
	Peers []*Peer
	// Apps contains configuration for zero or more application
	Apps []*App
}

// Validate validates the config
func (c *Config) Validate() error {
	if c.MspID == "" {
		return errors.New("field [MspID] is required")
	}
	if len(c.Peers) == 0 && len(c.Apps) == 0 {
		return errors.Errorf("config for MSP [%s] failed to validate: either one of the fields [Peers] or [Apps] is required", c.MspID)
	}
	if err := c.validatePeers(); err != nil {
		return errors.WithMessagef(err, "peer config for MSP [%s] failed to validate", c.MspID)
	}
	if err := c.validateApps(); err != nil {
		return errors.WithMessagef(err, "app config for MSP [%s] failed to validate", c.MspID)
	}
	return nil
}

func (c *Config) validatePeers() error {
	for _, p := range c.Peers {
		if err := p.Validate(); err != nil {
			return errors.WithMessagef(err, "peer [%s] failed to validate", p.PeerID)
		}
	}
	return nil
}

func (c *Config) validateApps() error {
	for _, app := range c.Apps {
		if err := app.Validate(); err != nil {
			return errors.WithMessagef(err, "app [%s] failed to validate", app.AppName)
		}
	}

	return nil
}

// Peer contains a collection of application configurations for a given peer
type Peer struct {
	// PeerID is the unique ID of the peer
	PeerID string
	// Apps contains configuration for one or more application
	Apps []*App
}

// Validate validates the peer config
func (p *Peer) Validate() error {
	if p.PeerID == "" {
		return errors.New("field [PeerID] is required")
	}
	if len(p.Apps) == 0 {
		return errors.New("field [Apps] is required")
	}
	for _, app := range p.Apps {
		if err := app.Validate(); err != nil {
			return errors.WithMessagef(err, "app [%s] in peer [%s] failed to validate", app.AppName, p.PeerID)
		}
	}

	return nil
}

// App contains the configuration for an application and/or multiple sub-components.
type App struct {
	// Name is the name of the application
	AppName string
	// Version is the version of the config
	Version string
	// Format describes the format of the data
	Format Format
	// Config contains the actual configuration
	Config string
	// Components zero or more component configs
	Components []*Component
}

// Validate validates the application configuration
func (app *App) Validate() error {
	if app.AppName == "" {
		return errors.New("field [Name] is required")
	}
	if app.Config == "" && len(app.Components) == 0 {
		return errors.New("either one of the fields [Config] or [Components] is required")
	}
	if len(app.Version) == 0 {
		return errors.New("field [Version] is required")
	}
	if app.Config != "" {
		if app.Format == "" {
			return errors.New("field [Format] is required")
		}
	} else {
		for _, comp := range app.Components {
			if err := comp.Validate(); err != nil {
				return errors.WithMessagef(err, "component [%s] in app [%s] failed to validate", comp.Name, app.AppName)
			}
		}
	}
	return nil
}

// Component contains the configuration for an application component.
type Component struct {
	// Name is the name of the component
	Name string
	// Version is the version of the config
	Version string
	// Format describes the format of the data
	Format Format
	// Config contains the actual configuration
	Config string
}

// Validate validates the Component
func (c *Component) Validate() error {
	if c.Name == "" {
		return errors.New("field [Name] is required")
	}
	if c.Config == "" {
		return errors.New("field [Config] is required")
	}
	if c.Format == "" {
		return errors.New("field [Format] is required")
	}
	if len(c.Version) == 0 {
		return errors.New("field [Version] is required")
	}
	return nil
}
