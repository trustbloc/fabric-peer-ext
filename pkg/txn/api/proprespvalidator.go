/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	pb "github.com/hyperledger/fabric-protos-go/peer"
)

// ProposalResponseValidator validates proposal responses
type ProposalResponseValidator interface {
	Validate(proposal *pb.SignedProposal, proposalResponses []*pb.ProposalResponse) (pb.TxValidationCode, error)
}
