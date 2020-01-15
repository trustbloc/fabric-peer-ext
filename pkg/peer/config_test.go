/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package peer

import (
	"testing"

	viper "github.com/spf13/viper2015"
	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	mspID := "Org1MSP"
	peerID := "peer1"
	peerAddress := "peer1.org1.com"
	mspConfigPath := "./msp"
	certFile := "./cert"

	viper.Set("peer.localMspId", mspID)
	viper.Set("peer.id", peerID)
	viper.Set("peer.address", peerAddress)
	viper.Set("peer.mspConfigPath", mspConfigPath)
	viper.Set("peer.tls.cert.file", certFile)

	c := newConfig()
	require.Equal(t, mspID, c.MSPID())
	require.Equal(t, peerID, c.PeerID())
	require.Equal(t, peerAddress, c.PeerAddress())
	require.Equal(t, mspConfigPath, c.MSPConfigPath())
	require.Equal(t, certFile, c.TLSCertPath())
}
