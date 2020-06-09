/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
)

// Service provides functions to collect endorsements and send endorsements to the Orderer
type Service interface {
	// Endorse collects endorsements according to chaincode policy
	Endorse(req *Request) (resp *channel.Response, err error)

	// EndorseAndCommit collects endorsements (according to chaincode policy) and sends the endorsements to the Orderer.
	// Returns the response and true if the transaction was committed.
	EndorseAndCommit(req *Request) (resp *channel.Response, committed bool, err error)

	// CommitEndorsements commits the provided endorsements. First the endorsements are verified for signature and policy,
	// and then the endorsements are sent to the Orderer.
	CommitEndorsements(req *CommitRequest) (*channel.Response, bool, error)

	// SigningIdentity returns the serialized identity of the proposal signer
	SigningIdentity() ([]byte, error)
}
