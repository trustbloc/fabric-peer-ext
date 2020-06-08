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

	// CommitType specifies how commits should be handled (default CommitOnWrite)
	CommitType CommitType

	// IgnoreNameSpaces ignore these namespaces in the write set when CommitType is CommitOnWrite
	IgnoreNameSpaces []Namespace

	// PeerFilter filters out peers using application-specific logic (optional)
	PeerFilter PeerFilter
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

// CommitType specifies how commits should be handled
type CommitType int

const (
	// CommitOnWrite indicates that the transaction should be committed only if
	// the consumer chaincode produces a write-set
	CommitOnWrite CommitType = iota

	// Commit indicates that the transaction should be committed
	Commit

	// NoCommit indicates that the transaction should not be committed
	NoCommit
)

// String returns the string value of CommitType
func (t CommitType) String() string {
	switch t {
	case CommitOnWrite:
		return "commitOnWrite"
	case Commit:
		return "commit"
	case NoCommit:
		return "noCommit"
	default:
		return "unknown"
	}
}

// Namespace contains a chaincode name and an optional set of private data collections to ignore
type Namespace struct {
	Name        string
	Collections []string
}

// Peer provides basic information about a peer
type Peer interface {
	MSPID() string
	Endpoint() string
}

// PeerFilter is applied to peers selected for endorsement and removes
// those groups that don't pass the filter acceptance test
type PeerFilter interface {
	Accept(peer Peer) bool
}
