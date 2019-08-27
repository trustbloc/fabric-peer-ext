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
	if err := cbk.populate(cfg, txID); err != nil {
		return nil, err
	}
	return cbk, nil
}

func (c keyValueMap) populate(config *config.Config, txID string) error {
	err := c.populatePeers(config, txID)
	if err != nil {
		return err
	}
	return c.populateApps(config, txID)
}

func (c keyValueMap) populatePeers(config *config.Config, txID string) error {
	logger.Debugf("Adding peer config ...")
	for _, peer := range config.Peers {
		if err := c.populatePeerApps(config.MspID, peer, txID); err != nil {
			return err
		}
	}
	return nil
}

func (c keyValueMap) populatePeerApps(mspID string, peer *config.Peer, txID string) error {
	logger.Debugf("... adding app configs for peer [%s] ...", peer.PeerID)
	for _, app := range peer.Apps {
		if err := c.populatePeerApp(mspID, peer.PeerID, app, txID); err != nil {
			return err
		}
	}
	return nil
}

func (c keyValueMap) populateApps(config *config.Config, txID string) error {
	logger.Debugf("... adding app configs ...")
	for _, app := range config.Apps {
		if err := c.populateApp(config.MspID, app, txID); err != nil {
			return err
		}
	}
	return nil
}

func (c keyValueMap) populateApp(mspID string, app *config.App, txID string) error {
	if len(app.Components) > 0 {
		return c.populateComponents(mspID, app, txID)
	}

	logger.Debugf("... adding config for app [%s] ...", app.AppName)

	c[*config.NewAppKey(mspID, app.AppName, app.Version)] = config.NewValue(txID, app.Config, app.Format)
	return nil
}

func (c keyValueMap) populatePeerApp(mspID, peerID string, app *config.App, txID string) error {
	logger.Debugf("... adding config for peer [%s] and app [%s] ...", peerID, app.AppName)
	if len(app.Components) > 0 {
		return c.populatePeerComponents(mspID, peerID, app, txID)
	}

	logger.Debugf("... adding config for peer [%s] and app [%s] ...", peerID, app.AppName)
	c[*config.NewPeerKey(mspID, peerID, app.AppName, app.Version)] = config.NewValue(txID, app.Config, app.Format)
	return nil
}

func (c keyValueMap) populateComponents(mspID string, app *config.App, txID string) error {
	logger.Debugf("... adding components for app [%s] ...", app.AppName)
	for _, comp := range app.Components {
		logger.Debugf("... adding component [%s:%s] for app [%s] ...", comp.Name, comp.Version, app.AppName)
		c[*config.NewComponentKey(mspID, app.AppName, app.Version, comp.Name, comp.Version)] = config.NewValue(txID, comp.Config, comp.Format)
	}
	return nil
}

func (c keyValueMap) populatePeerComponents(mspID, peerID string, app *config.App, txID string) error {
	logger.Debugf("... adding components for peer [%s] and app [%s] ...", peerID, app.AppName)
	for _, comp := range app.Components {
		logger.Debugf("... adding component [%s:%s] for peer [%s] and app [%s] ...", comp.Name, comp.Version, peerID, app.AppName)
		c[*config.NewPeerComponentKey(mspID, peerID, app.AppName, app.Version, comp.Name, comp.Version)] = config.NewValue(txID, comp.Config, comp.Format)
	}
	return nil
}
