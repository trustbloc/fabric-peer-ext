/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

// PeerConfig contains peer configuration
type PeerConfig interface {
	MSPID() string
	PeerAddress() string
	MSPConfigPath() string
	TLSCertPath() string
}
