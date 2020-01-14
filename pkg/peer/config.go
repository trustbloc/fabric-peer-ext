/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package peer

import (
	viper "github.com/spf13/viper2015"
)

type config struct {
	peerID        string
	peerAddress   string
	mspID         string
	mspConfigPath string
	certPath      string
}

func newConfig() *config {
	return &config{
		mspID:         viper.GetString("peer.localMspId"),
		peerID:        viper.GetString("peer.id"),
		peerAddress:   viper.GetString("peer.address"),
		mspConfigPath: viper.GetString("peer.mspConfigPath"),
		certPath:      viper.GetString("peer.tls.cert.file"),
	}
}

// PeerID returns the ID of the peer
func (c *config) PeerID() string {
	return c.peerID
}

// MSPID returns the MSP ID of the peer
func (c *config) MSPID() string {
	return c.mspID
}

// PeerAddress returns the address (host:port) of the peer
func (c *config) PeerAddress() string {
	return c.peerAddress
}

// MSPConfigPath returns the MSP config path for peer
func (c *config) MSPConfigPath() string {
	return c.mspConfigPath
}

// TLSCertPath returns absolute path to the TLS certificate
func (c *config) TLSCertPath() string {
	return c.certPath
}
