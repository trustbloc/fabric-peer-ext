/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
)

// Request contains the data required for endorsments
type Request struct {
	// ChaincodeID identifies the chaincode to invoke
	ChaincodeID string

	// Args to pass to the chaincode
	Args [][]byte

	// TransientData map (optional)
	TransientData map[string][]byte

	// Targets for the transaction (optional)
	Targets []fab.Peer

	// ChaincodeIDs contains all of the chaincodes that should be taken into consideration when choosing endorsers.
	// For example, if target chaincode, chaincodeX, invokes chaincodeY (using CC-to-CC invocation) then "chaincodeY"
	// should be specified.
	// If empty then only the target chaincode is considered.
	ChaincodeIDs []string
}

// PeerConfig contains peer configuration
type PeerConfig interface {
	PeerID() string
	MSPID() string
	PeerAddress() string
	MSPConfigPath() string
	TLSCertPath() string
}
