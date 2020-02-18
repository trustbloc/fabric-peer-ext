/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mgr

import (
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
)

type keyValueMap map[config.Key]*config.Value

// newKeyValueMap constructs a map of Key's to Value's from the given Config
func newKeyValueMap(cfg *config.Config, txID string) (keyValueMap, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	cbk := make(keyValueMap)
	cbk.populate(cfg, txID)
	return cbk, nil
}

func (c keyValueMap) populate(config *config.Config, txID string) {
	c.populatePeers(config, txID)
	c.populateApps(config, txID)
}

func (c keyValueMap) populatePeers(config *config.Config, txID string) {
	logger.Debugf("[%s] Adding peer configs ...", txID)
	for _, peer := range config.Peers {
		c.populatePeerApps(config.MspID, peer, txID)
	}
}

func (c keyValueMap) populatePeerApps(mspID string, peer *config.Peer, txID string) {
	logger.Debugf("[%s] Adding app configs for peer [%s] ...", txID, peer.PeerID)
	for _, app := range peer.Apps {
		c.populatePeerApp(mspID, peer.PeerID, app, txID)
	}
}

func (c keyValueMap) populateApps(config *config.Config, txID string) {
	logger.Debugf("[%s] Adding app configs ...", txID)
	for _, app := range config.Apps {
		c.populateApp(config.MspID, app, txID)
	}
}

func (c keyValueMap) populateApp(mspID string, app *config.App, txID string) {
	if len(app.Components) > 0 {
		c.populateComponents(mspID, app, txID)
	}

	if app.Config != "" {
		logger.Debugf("[%s] ... adding config for app [%s] ...", txID, app.AppName)
		c[*config.NewAppKey(mspID, app.AppName, app.Version)] = config.NewValue(txID, app.Config, app.Format, app.Tags...)
	}
}

func (c keyValueMap) populatePeerApp(mspID, peerID string, app *config.App, txID string) {
	logger.Debugf("[%s] ... adding config for peer [%s] and app [%s] ...", txID, peerID, app.AppName)
	if len(app.Components) > 0 {
		c.populatePeerComponents(mspID, peerID, app, txID)
	}

	if app.Config != "" {
		logger.Debugf("[%s] ... adding config for peer [%s] and app [%s] ...", txID, peerID, app.AppName)
		c[*config.NewPeerKey(mspID, peerID, app.AppName, app.Version)] = config.NewValue(txID, app.Config, app.Format, app.Tags...)
	}
}

func (c keyValueMap) populateComponents(mspID string, app *config.App, txID string) {
	logger.Debugf("[%s] ... adding components for app [%s] ...", txID, app.AppName)
	for _, comp := range app.Components {
		logger.Debugf("[%s] ... adding component [%s:%s] for app [%s] ...", txID, comp.Name, comp.Version, app.AppName)
		c[*config.NewComponentKey(mspID, app.AppName, app.Version, comp.Name, comp.Version)] = config.NewValue(txID, comp.Config, comp.Format, comp.Tags...)
	}
}

func (c keyValueMap) populatePeerComponents(mspID, peerID string, app *config.App, txID string) {
	logger.Debugf("[%s] ... adding components for peer [%s] and app [%s] ...", txID, peerID, app.AppName)
	for _, comp := range app.Components {
		logger.Debugf("[%s] ... adding component [%s:%s] for peer [%s] and app [%s] ...", txID, comp.Name, comp.Version, peerID, app.AppName)
		c[*config.NewPeerComponentKey(mspID, peerID, app.AppName, app.Version, comp.Name, comp.Version)] = config.NewValue(txID, comp.Config, comp.Format, comp.Tags...)
	}
}
