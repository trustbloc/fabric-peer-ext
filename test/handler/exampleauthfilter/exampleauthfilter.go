/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package exampleauthfilter

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	cb "github.com/hyperledger/fabric-protos-go/common"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/common/flogging"
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	gcommon "github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("ext_example_authfilter")

type gossipProvider interface {
	GetGossipService() gossipapi.GossipService
}

// AuthFilter is a sample Auth filter used in the BDD test. It demonstrates the handler registry and
// dependency injection for Auth filters.
type AuthFilter struct {
	next           pb.EndorserServer
	gossipProvider gossipProvider
}

// New returns a new example auth AuthFilter. The Gossip provider is supplied via dependency injection.
func New(gossip gossipProvider) *AuthFilter {
	return &AuthFilter{
		gossipProvider: gossip,
	}
}

// Name returns the unique name of the Auth filter
func (f *AuthFilter) Name() string {
	return "ExampleAuthFilter"
}

// Init initializes the Auth filter
func (f *AuthFilter) Init(next pb.EndorserServer) {
	logger.Info("Initialized example auth AuthFilter")

	f.next = next
}

// ProcessProposal is invoked during an endorsement (before the chaincode). This implementation
// looks for a specific chaincode name and function and, if satisfied, returns an error response
// with a message that contains the channel peer endpoints.
func (f *AuthFilter) ProcessProposal(ctx context.Context, signedProp *pb.SignedProposal) (*pb.ProposalResponse, error) {
	logger.Debugf("ProcessProposal in example auth AuthFilter was invoked. Signed proposal: %s", signedProp)

	channelID, ccName, funcName, err := getChannelCCAndFunc(signedProp)
	if err != nil {
		return nil, err
	}

	if ccName == "e2e_cc" && funcName == "authFilterError" {
		logger.Debugf("Returning peers in channel as an error message")

		gossip := f.gossipProvider.GetGossipService()

		var peersString string
		for _, p := range gossip.PeersOfChannel(gcommon.ChannelID(channelID)) {
			peersString += p.Endpoint + " "
		}

		return &pb.ProposalResponse{
			Response: &pb.Response{
				Status:  shim.ERROR,
				Message: fmt.Sprintf("Peers in channel [%s]: %s", channelID, peersString),
			},
		}, nil
	}

	return f.next.ProcessProposal(ctx, signedProp)
}

func getChannelCCAndFunc(signedProp *pb.SignedProposal) (string, string, string, error) {
	proposal, err := protoutil.UnmarshalProposal(signedProp.ProposalBytes)
	if err != nil {
		return "", "", "", err
	}

	channelID, cis, err := getChaincodeInvocationSpec(proposal)
	if err != nil {
		return "", "", "", err
	}

	if cis == nil || cis.ChaincodeSpec == nil || cis.ChaincodeSpec.ChaincodeId == nil || cis.ChaincodeSpec.Input == nil {
		return "", "", "", fmt.Errorf("invalid cc invocation spec: %s", err)
	}

	ccName := cis.ChaincodeSpec.ChaincodeId.Name
	funcName := ""

	if len(cis.ChaincodeSpec.Input.Args) != 0 {
		funcName = string(cis.ChaincodeSpec.Input.Args[0])
	}

	return channelID, ccName, funcName, nil
}

func getChaincodeInvocationSpec(prop *pb.Proposal) (string, *pb.ChaincodeInvocationSpec, error) {
	if prop == nil {
		return "", nil, errors.New("proposal is nil")
	}

	header, err := getHeader(prop.Header)
	if err != nil {
		return "", nil, err
	}

	channelHeader, err := protoutil.UnmarshalChannelHeader(header.ChannelHeader)
	if err != nil {
		return "", nil, err
	}

	ccPropPayload, err := getChaincodeProposalPayload(prop.Payload)
	if err != nil {
		return "", nil, err
	}

	cis := &pb.ChaincodeInvocationSpec{}
	err = proto.Unmarshal(ccPropPayload.Input, cis)
	if err != nil {
		return "", nil, err
	}

	return channelHeader.ChannelId, cis, nil
}

func getHeader(bytes []byte) (*cb.Header, error) {
	hdr := &cb.Header{}
	err := proto.Unmarshal(bytes, hdr)
	return hdr, errors.Wrap(err, "error unmarshaling Header")
}

func getChaincodeProposalPayload(bytes []byte) (*pb.ChaincodeProposalPayload, error) {
	cpp := &pb.ChaincodeProposalPayload{}
	err := proto.Unmarshal(bytes, cpp)
	return cpp, errors.Wrap(err, "error unmarshaling ChaincodeProposalPayload")
}
