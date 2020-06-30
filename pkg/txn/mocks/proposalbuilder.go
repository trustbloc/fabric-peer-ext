/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/msp"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/bccsp/utils"
	putils "github.com/hyperledger/fabric/protoutil"
)

var certPem = `-----BEGIN CERTIFICATE-----
MIICCjCCAbGgAwIBAgIQOcq9Om9VwUe9hGN0TTGw1DAKBggqhkjOPQQDAjBYMQsw
CQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTEWMBQGA1UEBxMNU2FuIEZy
YW5jaXNjbzENMAsGA1UEChMET3JnMTENMAsGA1UEAxMET3JnMTAeFw0xNzA1MDgw
OTMwMzRaFw0yNzA1MDYwOTMwMzRaMGUxCzAJBgNVBAYTAlVTMRMwEQYDVQQIEwpD
YWxpZm9ybmlhMRYwFAYDVQQHEw1TYW4gRnJhbmNpc2NvMRUwEwYDVQQKEwxPcmcx
LXNlcnZlcjExEjAQBgNVBAMTCWxvY2FsaG9zdDBZMBMGByqGSM49AgEGCCqGSM49
AwEHA0IABAm+2CZhbmsnA+HKQynXKz7fVZvvwlv/DdNg3Mdg7lIcP2z0b07/eAZ5
0chdJNcjNAd/QAj/mmGG4dObeo4oTKGjUDBOMA4GA1UdDwEB/wQEAwIFoDAdBgNV
HSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwDAYDVR0TAQH/BAIwADAPBgNVHSME
CDAGgAQBAgMEMAoGCCqGSM49BAMCA0cAMEQCIG55RvN4Boa0WS9UcIb/tI2YrAT8
EZd/oNnZYlbxxyvdAiB6sU9xAn4oYIW9xtrrOISv3YRg8rkCEATsagQfH8SiLg==
-----END CERTIFICATE-----`

var keyPem = []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEICfXQtVmdQAlp/l9umWJqCXNTDurmciDNmGHPpxHwUK/oAoGCCqGSM49
AwEHoUQDQgAECb7YJmFuaycD4cpDKdcrPt9Vm+/CW/8N02Dcx2DuUhw/bPRvTv94
BnnRyF0k1yM0B39ACP+aYYbh05t6jihMoQ==
-----END EC PRIVATE KEY-----`)

// ProposalBuilder builds a mock signed proposal
type ProposalBuilder struct {
	channelID    string
	chaincodeID  string
	mspID        string
	args         [][]byte
	transientMap map[string][]byte
}

// NewProposalBuilder returns a mock proposal builder
func NewProposalBuilder() *ProposalBuilder {
	return &ProposalBuilder{
		transientMap: make(map[string][]byte),
	}
}

// ChannelID sets the channel ID for the proposal
func (b *ProposalBuilder) ChannelID(value string) *ProposalBuilder {
	b.channelID = value
	return b
}

// ChaincodeID sets the chaincode ID for the proposal
func (b *ProposalBuilder) ChaincodeID(value string) *ProposalBuilder {
	b.chaincodeID = value
	return b
}

// MSPID sets the MSP ID of the creator
func (b *ProposalBuilder) MSPID(value string) *ProposalBuilder {
	b.mspID = value
	return b
}

// Args adds chaincode arguments
func (b *ProposalBuilder) Args(args ...[]byte) *ProposalBuilder {
	b.args = args
	return b
}

// TransientArg adds a transient key-value
func (b *ProposalBuilder) TransientArg(key string, value []byte) *ProposalBuilder {
	b.transientMap[key] = value
	return b
}

// Build returns the signed proposal
func (b *ProposalBuilder) Build() *pb.SignedProposal {
	// create invocation spec to target a chaincode with arguments
	ccis := &pb.ChaincodeInvocationSpec{ChaincodeSpec: &pb.ChaincodeSpec{
		Type: pb.ChaincodeSpec_GOLANG, ChaincodeId: &pb.ChaincodeID{Name: b.chaincodeID},
		Input: &pb.ChaincodeInput{Args: b.args}}}

	sID := &msp.SerializedIdentity{Mspid: b.mspID, IdBytes: []byte(certPem)}
	creator, err := proto.Marshal(sID)
	if err != nil {
		panic(err)
	}

	proposal, _, err := putils.CreateChaincodeProposalWithTransient(
		cb.HeaderType_ENDORSER_TRANSACTION, b.channelID, ccis, creator, b.transientMap)
	if err != nil {
		panic(fmt.Sprintf("Could not create chaincode proposal, err %s", err))
	}

	proposalBytes, err := proto.Marshal(proposal)
	if err != nil {
		panic(fmt.Sprintf("Error marshalling proposal: %s", err))
	}

	// sign proposal bytes
	block, _ := pem.Decode(keyPem)
	lowLevelKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		panic(err)
	}

	signature, err := signECDSA(lowLevelKey, proposalBytes)
	if err != nil {
		panic(err)
	}

	return &pb.SignedProposal{ProposalBytes: proposalBytes, Signature: signature}
}

func signECDSA(k *ecdsa.PrivateKey, digest []byte) (signature []byte, err error) {
	hash := sha256.New()
	_, err = hash.Write(digest)
	if err != nil {
		return nil, err
	}
	r, s, err := ecdsa.Sign(rand.Reader, k, hash.Sum(nil))
	if err != nil {
		return nil, err
	}

	s, err = utils.ToLowS(&k.PublicKey, s)
	if err != nil {
		return nil, err
	}

	return utils.MarshalECDSASignature(r, s)
}
