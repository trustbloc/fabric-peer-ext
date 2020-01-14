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

	// InvocationChain contains meta-data that's used by some Selection Service implementations
	// to choose endorsers that satisfy the endorsement policies of all chaincodes involved
	// in an invocation chain (i.e. for CC-to-CC invocations).
	// Each chaincode may also be associated with a set of private data collection names
	// which are used by some Selection Services (e.g. Fabric Selection) to exclude endorsers
	// that do NOT have read access to the collections.
	// The invoked chaincode (specified by ChaincodeID) may optionally be added to the invocation
	// chain along with any collections, otherwise it may be omitted.
	InvocationChain []*ChaincodeCall
}

// ChaincodeCall ...
type ChaincodeCall struct {
	ChaincodeName string
	Collections   []string
}

// PeerConfig contains peer configuration
type PeerConfig interface {
	PeerID() string
	MSPID() string
	PeerAddress() string
	MSPConfigPath() string
	TLSCertPath() string
}
