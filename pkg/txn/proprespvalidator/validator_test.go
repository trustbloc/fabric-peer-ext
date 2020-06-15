/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package proprespvalidator

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric-protos-go/common"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/gossip/blockpublisher"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
	txnmocks "github.com/trustbloc/fabric-peer-ext/pkg/txn/mocks"
)

const (
	ccLSCC    = "lscc"
	channelID = "testchannel"

	ccID1 = "chaincode1"
	ccID2 = "chaincode2"
	ccID3 = "chaincode3"

	ccVersion1 = "v1"
	ccVersion2 = "v2"

	org1MSP = "Org1MSP"
	org2MSP = "Org2MSP"

	lsccID       = "lscc"
	upgradeEvent = "upgrade"
)

func TestValidator_ValidateProposalResponses_InvalidInput(t *testing.T) {
	cdBuilder := txnmocks.NewChaincodeDataBuilder().
		Name(ccID1).
		Version(ccVersion1).
		VSCC("vscc").
		Policy("OutOf(1,'Org1MSP.member')")

	qe := mocks.NewQueryExecutor().WithState(ccLSCC, ccID1, cdBuilder.BuildBytes())
	idd := &mocks.IdentityDeserializer{}
	identity := txnmocks.NewIdentity()
	idd.DeserializeIdentityReturns(identity, nil)
	bp := blockpublisher.New(channelID)

	v := newValidator(channelID, newQueryExecutorProvider(qe), newMockPolicyEvaluator(), idd, bp)
	require.NotNil(t, v)

	t.Run("No Proposal Responses -> error", func(t *testing.T) {
		proposal := &pb.SignedProposal{}
		var responses []*pb.ProposalResponse

		code, err := v.Validate(proposal, responses)
		require.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		require.EqualError(t, err, "no proposal responses")
	})

	t.Run("Nil chaincode response -> error", func(t *testing.T) {
		pBuilder := txnmocks.NewProposalBuilder().ChannelID(channelID).MSPID(org1MSP).ChaincodeID(ccID1)
		rBuilder := txnmocks.NewProposalResponsesBuilder()
		rBuilder.ProposalResponse()

		code, err := v.Validate(pBuilder.Build(), rBuilder.Build())
		require.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		require.EqualError(t, err, "nil chaincode response")
	})

	t.Run("Failed chaincode response status -> error", func(t *testing.T) {
		pBuilder := txnmocks.NewProposalBuilder().ChannelID(channelID).MSPID(org1MSP).ChaincodeID(ccID1)
		rBuilder := txnmocks.NewProposalResponsesBuilder()
		rBuilder.ProposalResponse().Response().Status(int32(cb.Status_BAD_REQUEST))

		code, err := v.Validate(pBuilder.Build(), rBuilder.Build())
		require.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		require.EqualError(t, err, "failed chaincode response status")
	})

	t.Run("No endorsement in response -> error", func(t *testing.T) {
		pBuilder := txnmocks.NewProposalBuilder().ChaincodeID(ccID1)

		responsesBuilder := txnmocks.NewProposalResponsesBuilder()
		rBuilder := responsesBuilder.ProposalResponse()
		rBuilder.Response().Status(int32(cb.Status_SUCCESS))
		rBuilder.Payload().ChaincodeAction().ChaincodeID(ccID2, "", ccVersion1)

		code, err := v.Validate(pBuilder.Build(), responsesBuilder.Build())
		require.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		require.EqualError(t, err, "missing endorsement in proposal response")
	})

	t.Run("Nil chaincode ID in ChaincodeAction -> error", func(t *testing.T) {
		pBuilder := txnmocks.NewProposalBuilder().ChannelID(channelID).MSPID(org1MSP).ChaincodeID(ccID1)
		responsesBuilder := txnmocks.NewProposalResponsesBuilder()
		rBuilder := responsesBuilder.ProposalResponse()
		rBuilder.Response().Status(int32(cb.Status_SUCCESS))
		rBuilder.Endorsement()

		code, err := v.Validate(pBuilder.Build(), responsesBuilder.Build())
		require.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		require.EqualError(t, err, "nil ChaincodeId in ChaincodeAction")
	})

	t.Run("Invalid chaincode version -> error", func(t *testing.T) {
		pBuilder := txnmocks.NewProposalBuilder().ChannelID(channelID).MSPID(org1MSP).ChaincodeID(ccID1)
		responsesBuilder := txnmocks.NewProposalResponsesBuilder()
		rBuilder := responsesBuilder.ProposalResponse()
		rBuilder.Response().Status(int32(cb.Status_SUCCESS))
		rBuilder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", "")
		rBuilder.Endorsement()

		code, err := v.Validate(pBuilder.Build(), responsesBuilder.Build())
		require.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		require.EqualError(t, err, "invalid chaincode version in ChaincodeAction")
	})

	t.Run("Invalid chaincode ID in proposal -> error", func(t *testing.T) {
		pBuilder := txnmocks.NewProposalBuilder()

		responsesBuilder := txnmocks.NewProposalResponsesBuilder()
		rBuilder := responsesBuilder.ProposalResponse()
		rBuilder.Response().Status(int32(cb.Status_SUCCESS))
		rBuilder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		rBuilder.Endorsement()

		code, err := v.Validate(pBuilder.Build(), responsesBuilder.Build())
		require.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		require.EqualError(t, err, "invalid chaincode ID in proposal")
	})

	t.Run("inconsistent chaincode ID info -> error", func(t *testing.T) {
		pBuilder := txnmocks.NewProposalBuilder().ChaincodeID(ccID1)

		responsesBuilder := txnmocks.NewProposalResponsesBuilder()
		rBuilder := responsesBuilder.ProposalResponse()
		rBuilder.Response().Status(int32(cb.Status_SUCCESS))
		rBuilder.Payload().ChaincodeAction().ChaincodeID(ccID2, "", ccVersion1)
		rBuilder.Endorsement()

		code, err := v.Validate(pBuilder.Build(), responsesBuilder.Build())
		require.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		require.EqualError(t, err, "inconsistent chaincode ID info (chaincode1/chaincode2)")
	})

	t.Run("Chaincode event chaincode ID does not match -> error", func(t *testing.T) {
		pBuilder := txnmocks.NewProposalBuilder().ChaincodeID(ccID1)

		responsesBuilder := txnmocks.NewProposalResponsesBuilder()
		rBuilder := responsesBuilder.ProposalResponse()
		rBuilder.Response().Status(int32(cb.Status_SUCCESS))
		rBuilder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1).Events().ChaincodeID(ccID2)
		rBuilder.Endorsement()

		code, err := v.Validate(pBuilder.Build(), responsesBuilder.Build())
		require.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		require.EqualError(t, err, "chaincode event chaincode id does not match chaincode action chaincode id")
	})

	t.Run("Endorsement mismatch -> error", func(t *testing.T) {
		pBuilder := txnmocks.NewProposalBuilder().ChaincodeID(ccID1)

		responsesBuilder := txnmocks.NewProposalResponsesBuilder()
		r1Builder := responsesBuilder.ProposalResponse()
		r1Builder.Response().Status(int32(cb.Status_SUCCESS))
		r1Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		r1Builder.Endorsement()
		r2Builder := responsesBuilder.ProposalResponse()
		r2Builder.Response().Status(int32(cb.Status_SUCCESS)).Payload([]byte("some payload"))
		r2Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		r2Builder.Endorsement()

		code, err := v.Validate(pBuilder.Build(), responsesBuilder.Build())
		require.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		require.EqualError(t, err, "one or more proposal responses do not match")
	})

	t.Run("Invalid signature -> error", func(t *testing.T) {
		identity.WithError(fmt.Errorf("invalid signature"))
		defer identity.WithError(nil)

		pBuilder := txnmocks.NewProposalBuilder().ChaincodeID(ccID1)

		responsesBuilder := txnmocks.NewProposalResponsesBuilder()
		rBuilder := responsesBuilder.ProposalResponse()
		rBuilder.Response().Status(int32(cb.Status_SUCCESS))
		rBuilder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		rBuilder.Endorsement()

		code, err := v.Validate(pBuilder.Build(), responsesBuilder.Build())
		require.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		require.EqualError(t, err, "the creator certificate is not valid: invalid signature")
	})

	t.Run("Invalid endorser -> error", func(t *testing.T) {
		pBuilder := txnmocks.NewProposalBuilder().ChaincodeID(ccID1)

		responsesBuilder := txnmocks.NewProposalResponsesBuilder()
		rBuilder := responsesBuilder.ProposalResponse()
		rBuilder.Response().Status(int32(cb.Status_SUCCESS))
		rBuilder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		rBuilder.Endorsement()

		responses := responsesBuilder.Build()
		for _, r := range responses {
			r.Endorsement.Endorser = []byte("invalid endorser")
		}

		code, err := v.Validate(pBuilder.Build(), responses)
		require.Equal(t, pb.TxValidationCode_ENDORSEMENT_POLICY_FAILURE, code)
		require.Contains(t, err.Error(), "Unmarshal endorser error")
	})
}

func TestValidator_ValidateProposalResponses_ChaincodeDataError(t *testing.T) {
	pBuilder := txnmocks.NewProposalBuilder().ChannelID(channelID).MSPID(org1MSP).ChaincodeID(ccID1)

	qe := mocks.NewQueryExecutor()
	idd := &mocks.IdentityDeserializer{}
	identity1 := txnmocks.NewIdentity()
	idd.DeserializeIdentityReturns(identity1, nil)
	bp := blockpublisher.New(channelID)

	v := newValidator(channelID, newQueryExecutorProvider(qe), newMockPolicyEvaluator(), idd, bp)

	require.NotNil(t, v)

	t.Run("Get chaincode data -> error", func(t *testing.T) {
		id1Bytes, err := identity1.Serialize()
		require.NoError(t, err)

		responsesBuilder := txnmocks.NewProposalResponsesBuilder()
		r1Builder := responsesBuilder.ProposalResponse()
		r1Builder.Response().Status(int32(cb.Status_SUCCESS)).Payload([]byte("payload"))
		r1Builder.Endorsement().Endorser(id1Bytes).Signature([]byte("id1"))
		ca1Builder := r1Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		ca1Builder.Results().Namespace(ccID2).Write("key1", []byte("value1"))

		code, err := v.Validate(pBuilder.Build(), responsesBuilder.Build())
		require.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		require.Error(t, err)
		require.Contains(t, err.Error(), "lscc's state for [chaincode1] not found")
	})

	t.Run("No VSCC in chaincode data -> error", func(t *testing.T) {
		cdBuilder := txnmocks.NewChaincodeDataBuilder().
			Name(ccID1).
			Version(ccVersion1).
			Policy("OutOf(1,'Org1MSP.member')")
		qe.WithState(ccLSCC, ccID1, cdBuilder.BuildBytes())

		id1Bytes, err := identity1.Serialize()
		require.NoError(t, err)

		responsesBuilder := txnmocks.NewProposalResponsesBuilder()
		r1Builder := responsesBuilder.ProposalResponse()
		r1Builder.Response().Status(int32(cb.Status_SUCCESS)).Payload([]byte("payload"))
		r1Builder.Endorsement().Endorser(id1Bytes).Signature([]byte("id1"))
		ca1Builder := r1Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		ca1Builder.Results().Namespace(ccID2).Write("key1", []byte("value1"))

		code, err := v.Validate(pBuilder.Build(), responsesBuilder.Build())
		require.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		require.Error(t, err)
		require.Contains(t, err.Error(), "vscc field must be set")
	})

	t.Run("No policy in chaincode data -> error", func(t *testing.T) {
		cdBuilder := txnmocks.NewChaincodeDataBuilder().
			Name(ccID1).
			Version(ccVersion1).
			VSCC("vscc")
		qe.WithState(ccLSCC, ccID1, cdBuilder.BuildBytes())

		id1Bytes, err := identity1.Serialize()
		require.NoError(t, err)

		responsesBuilder := txnmocks.NewProposalResponsesBuilder()
		r1Builder := responsesBuilder.ProposalResponse()
		r1Builder.Response().Status(int32(cb.Status_SUCCESS)).Payload([]byte("payload"))
		r1Builder.Endorsement().Endorser(id1Bytes).Signature([]byte("id1"))
		ca1Builder := r1Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		ca1Builder.Results().Namespace(ccID2).Write("key1", []byte("value1"))

		code, err := v.Validate(pBuilder.Build(), responsesBuilder.Build())
		require.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		require.Error(t, err)
		require.Contains(t, err.Error(), "policy field must be set")
	})

	t.Run("Expired chaincode -> error", func(t *testing.T) {
		cdBuilder := txnmocks.NewChaincodeDataBuilder().
			Name(ccID1).
			Version(ccVersion2).
			VSCC("vscc").
			Policy("OutOf(1,'Org1MSP.member')")
		qe.WithState(ccLSCC, ccID1, cdBuilder.BuildBytes())

		id1Bytes, err := identity1.Serialize()
		require.NoError(t, err)

		responsesBuilder := txnmocks.NewProposalResponsesBuilder()
		r1Builder := responsesBuilder.ProposalResponse()
		r1Builder.Response().Status(int32(cb.Status_SUCCESS)).Payload([]byte("payload"))
		r1Builder.Endorsement().Endorser(id1Bytes).Signature([]byte("id1"))
		ca1Builder := r1Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		ca1Builder.Results().Namespace(ccID2).Write("key1", []byte("value1"))

		code, err := v.Validate(pBuilder.Build(), responsesBuilder.Build())
		require.Equal(t, pb.TxValidationCode_EXPIRED_CHAINCODE, code)
		require.Error(t, err)
		require.Contains(t, err.Error(), "chaincode chaincode1:v1/testchannel didn't match chaincode1:v2/testchannel")
	})
}

func TestValidator_ValidateProposalResponses(t *testing.T) {
	pBuilder := txnmocks.NewProposalBuilder().ChannelID(channelID).MSPID(org1MSP).ChaincodeID(ccID1)

	qe := mocks.NewQueryExecutor()
	identity1 := txnmocks.NewIdentity().WithMSPID(org1MSP)
	identity2 := txnmocks.NewIdentity().WithMSPID(org2MSP)

	policyEvaluator := newMockPolicyEvaluator()

	idd := &mocks.IdentityDeserializer{}
	idd.DeserializeIdentityReturns(identity1, nil)
	bp := blockpublisher.New(channelID)

	v := newValidator(channelID, newQueryExecutorProvider(qe), policyEvaluator, idd, bp)
	require.NotNil(t, v)

	t.Run("Nil response -> error", func(t *testing.T) {
		code, err := v.Validate(pBuilder.Build(), []*pb.ProposalResponse{{}})
		require.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		require.EqualError(t, err, "nil chaincode response")
	})

	t.Run("Duplicate identities -> error", func(t *testing.T) {
		cdBuilder := txnmocks.NewChaincodeDataBuilder().
			Name(ccID1).
			Version(ccVersion1).
			VSCC("vscc").
			Policy("OutOf(2,'Org1MSP.member','Org2MSP.member')")

		qe.WithState(ccLSCC, ccID1, cdBuilder.BuildBytes())

		policyBytes := cdBuilder.Build().Policy
		policyEvaluator.WithError(policyBytes, fmt.Errorf("signature set did not satisfy policy"))
		defer func() { policyEvaluator.WithError(policyBytes, nil) }()

		responsesBuilder := txnmocks.NewProposalResponsesBuilder()
		r1Builder := responsesBuilder.ProposalResponse()
		r1Builder.Response().Status(int32(cb.Status_SUCCESS)).Payload([]byte("payload"))
		r1Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		r1Builder.Endorsement()
		r2Builder := responsesBuilder.ProposalResponse()
		r2Builder.Response().Status(int32(cb.Status_SUCCESS)).Payload([]byte("payload"))
		r2Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		r2Builder.Endorsement()

		code, err := v.Validate(pBuilder.Build(), responsesBuilder.Build())
		require.Equal(t, pb.TxValidationCode_ENDORSEMENT_POLICY_FAILURE, code)
		require.EqualError(t, err, errDuplicateIdentity)
	})

	t.Run("Policy satisfied -> success", func(t *testing.T) {
		cdBuilder := txnmocks.NewChaincodeDataBuilder().
			Name(ccID1).
			Version(ccVersion1).
			VSCC("vscc").
			Policy("OutOf(2,'Org1MSP.member','Org2MSP.member')")
		qe.WithState(ccLSCC, ccID1, cdBuilder.BuildBytes())

		id1Bytes, err := identity1.Serialize()
		require.NoError(t, err)
		id2Bytes, err := identity2.Serialize()
		require.NoError(t, err)

		responsesBuilder := txnmocks.NewProposalResponsesBuilder()
		r1Builder := responsesBuilder.ProposalResponse()
		r1Builder.Response().Status(int32(cb.Status_SUCCESS)).Payload([]byte("payload"))
		r1Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		r1Builder.Endorsement().Endorser(id1Bytes).Signature([]byte("id1"))
		r2Builder := responsesBuilder.ProposalResponse()
		r2Builder.Response().Status(int32(cb.Status_SUCCESS)).Payload([]byte("payload"))
		r2Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		r2Builder.Endorsement().Endorser(id2Bytes).Signature([]byte("id2"))

		code, err := v.Validate(pBuilder.Build(), responsesBuilder.Build())
		require.Equal(t, pb.TxValidationCode_VALID, code)
		require.NoError(t, err)
	})

	t.Run("Identity deserializer error -> error", func(t *testing.T) {
		errExpected := errors.New("injected deserializer error")
		idd := &mocks.IdentityDeserializer{}
		idd.DeserializeIdentityReturns(nil, errExpected)
		bp := blockpublisher.New(channelID)

		v := newValidator(channelID, newQueryExecutorProvider(qe), policyEvaluator, idd, bp)
		require.NotNil(t, v)

		cdBuilder := txnmocks.NewChaincodeDataBuilder().
			Name(ccID1).
			Version(ccVersion1).
			VSCC("vscc").
			Policy("OutOf(2,'Org1MSP.member','Org2MSP.member')")
		qe.WithState(ccLSCC, ccID1, cdBuilder.BuildBytes())

		responsesBuilder := txnmocks.NewProposalResponsesBuilder()
		r1Builder := responsesBuilder.ProposalResponse()
		r1Builder.Response().Status(int32(cb.Status_SUCCESS)).Payload([]byte("payload"))
		r1Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		r1Builder.Endorsement().Endorser([]byte("id bytes")).Signature([]byte("id1"))

		code, err := v.Validate(pBuilder.Build(), responsesBuilder.Build())
		require.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
	})

	t.Run("Identity error -> error", func(t *testing.T) {
		errExpected := errors.New("injected identity error")
		identity1 := txnmocks.NewIdentity().WithError(errExpected)
		idd := &mocks.IdentityDeserializer{}
		idd.DeserializeIdentityReturns(identity1, nil)
		bp := blockpublisher.New(channelID)

		v := newValidator(channelID, newQueryExecutorProvider(qe), policyEvaluator, idd, bp)
		require.NotNil(t, v)

		cdBuilder := txnmocks.NewChaincodeDataBuilder().
			Name(ccID1).
			Version(ccVersion1).
			VSCC("vscc").
			Policy("OutOf(2,'Org1MSP.member','Org2MSP.member')")
		qe.WithState(ccLSCC, ccID1, cdBuilder.BuildBytes())

		responsesBuilder := txnmocks.NewProposalResponsesBuilder()
		r1Builder := responsesBuilder.ProposalResponse()
		r1Builder.Response().Status(int32(cb.Status_SUCCESS)).Payload([]byte("payload"))
		r1Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		r1Builder.Endorsement().Endorser([]byte("id bytes")).Signature([]byte("id1"))

		code, err := v.Validate(pBuilder.Build(), responsesBuilder.Build())
		require.Equal(t, pb.TxValidationCode_INVALID_OTHER_REASON, code)
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
	})

	t.Run("CC-to-CC -> error", func(t *testing.T) {
		cd1Builder := txnmocks.NewChaincodeDataBuilder().
			Name(ccID1).
			Version(ccVersion1).
			VSCC("vscc").
			Policy("OutOf(1,'Org1MSP.member')")
		qe.WithState(ccLSCC, ccID1, cd1Builder.BuildBytes())

		cd2Builder := txnmocks.NewChaincodeDataBuilder().
			Name(ccID2).
			Version(ccVersion1).
			VSCC("vscc").
			Policy("OutOf(2,'Org1MSP.member','Org2MSP.member')")
		qe.WithState(ccLSCC, ccID2, cd2Builder.BuildBytes())

		// Set an error on CC2 policy evaluation
		policyBytes := cd2Builder.Build().Policy
		policyEvaluator.WithError(policyBytes, fmt.Errorf("signature set did not satisfy policy"))
		defer func() { policyEvaluator.WithError(policyBytes, nil) }()

		id1Bytes, err := identity1.Serialize()
		require.NoError(t, err)

		responsesBuilder := txnmocks.NewProposalResponsesBuilder()
		r1Builder := responsesBuilder.ProposalResponse()
		r1Builder.Response().Status(int32(cb.Status_SUCCESS)).Payload([]byte("payload"))
		r1Builder.Endorsement().Endorser(id1Bytes).Signature([]byte("id1"))
		ca1Builder := r1Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		resultsBuilder := ca1Builder.Results()
		resultsBuilder.Namespace(ccID1).Read("key1", 1000, 1)
		resultsBuilder.Namespace(ccID2).Write("key1", []byte("value1"))

		code, err := v.Validate(pBuilder.Build(), responsesBuilder.Build())
		require.Equal(t, pb.TxValidationCode_ENDORSEMENT_POLICY_FAILURE, code)
		require.EqualError(t, err, "VSCC error: endorsement policy failure, err: signature set did not satisfy policy")
	})

	t.Run("CC-to-CC -> success", func(t *testing.T) {
		cd1Builder := txnmocks.NewChaincodeDataBuilder().
			Name(ccID1).
			Version(ccVersion1).
			VSCC("vscc").
			Policy("OutOf(1,'Org1MSP.member')")
		qe.WithState(ccLSCC, ccID1, cd1Builder.BuildBytes())

		cd2Builder := txnmocks.NewChaincodeDataBuilder().
			Name(ccID2).
			Version(ccVersion1).
			VSCC("vscc").
			Policy("OutOf(2,'Org1MSP.member','Org2MSP.member')")
		qe.WithState(ccLSCC, ccID2, cd2Builder.BuildBytes())

		cd3Builder := txnmocks.NewChaincodeDataBuilder().
			Name(ccID3).
			Version(ccVersion1).
			VSCC("vscc").
			Policy("OutOf(2,'Org1MSP.member','Org2MSP.member')")
		qe.WithState(ccLSCC, ccID3, cd3Builder.BuildBytes())

		id1Bytes, err := identity1.Serialize()
		require.NoError(t, err)

		responsesBuilder := txnmocks.NewProposalResponsesBuilder()
		r1Builder := responsesBuilder.ProposalResponse()
		r1Builder.Response().Status(int32(cb.Status_SUCCESS)).Payload([]byte("payload"))
		r1Builder.Endorsement().Endorser(id1Bytes).Signature([]byte("id1"))
		ca1Builder := r1Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		resultsBuilder := ca1Builder.Results()
		resultsBuilder.Namespace(ccID1).Read("key1", 1000, 1)
		resultsBuilder.Namespace(ccID2).Write("key1", []byte("value1"))

		code, err := v.Validate(pBuilder.Build(), responsesBuilder.Build())
		require.Equal(t, pb.TxValidationCode_VALID, code)
		require.NoError(t, err)
	})

	t.Run("Write to LSCC -> error", func(t *testing.T) {
		cd1Builder := txnmocks.NewChaincodeDataBuilder().
			Name(ccID1).
			Version(ccVersion1).
			VSCC("vscc").
			Policy("OutOf(1,'Org1MSP.member')")
		qe.WithState(ccLSCC, ccID1, cd1Builder.BuildBytes())

		id1Bytes, err := identity1.Serialize()
		require.NoError(t, err)

		responsesBuilder := txnmocks.NewProposalResponsesBuilder()
		r1Builder := responsesBuilder.ProposalResponse()
		r1Builder.Response().Status(int32(cb.Status_SUCCESS)).Payload([]byte("payload"))
		r1Builder.Endorsement().Endorser(id1Bytes).Signature([]byte("id1"))
		ca1Builder := r1Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		ca1Builder.Results().Namespace(ccLSCC).Write("key1", []byte("value1"))

		code, err := v.Validate(pBuilder.Build(), responsesBuilder.Build())
		require.Equal(t, pb.TxValidationCode_ILLEGAL_WRITESET, code)
		require.EqualError(t, err, "chaincode chaincode1 attempted to write to the namespace of LSCC")
	})

	t.Run("Reset Validator -> success", func(t *testing.T) {
		cdBuilder := txnmocks.NewChaincodeDataBuilder().
			Name(ccID1).
			Version(ccVersion1).
			VSCC("vscc").
			Policy("OutOf(2,'Org1MSP.member','Org2MSP.member')")
		qe.WithState(ccLSCC, ccID1, cdBuilder.BuildBytes())

		id1Bytes, err := identity1.Serialize()
		require.NoError(t, err)
		id2Bytes, err := identity2.Serialize()
		require.NoError(t, err)

		responsesBuilder := txnmocks.NewProposalResponsesBuilder()
		r1Builder := responsesBuilder.ProposalResponse()
		r1Builder.Response().Status(int32(cb.Status_SUCCESS)).Payload([]byte("payload"))
		r1Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		r1Builder.Endorsement().Endorser(id1Bytes).Signature([]byte("id1"))
		r2Builder := responsesBuilder.ProposalResponse()
		r2Builder.Response().Status(int32(cb.Status_SUCCESS)).Payload([]byte("payload"))
		r2Builder.Payload().ChaincodeAction().ChaincodeID(ccID1, "", ccVersion1)
		r2Builder.Endorsement().Endorser(id2Bytes).Signature([]byte("id2"))

		code, err := v.Validate(pBuilder.Build(), responsesBuilder.Build())
		require.Equal(t, pb.TxValidationCode_VALID, code)
		require.NoError(t, err)

		b := mocks.NewBlockBuilder(channelID, 1000)
		lceBytes, err := proto.Marshal(&pb.LifecycleEvent{ChaincodeName: ccID2})
		require.NoError(t, err)
		require.NotNil(t, lceBytes)

		b.Transaction("tx1", pb.TxValidationCode_VALID).
			ChaincodeAction(lsccID).ChaincodeEvent(upgradeEvent, lceBytes)

		bp.Publish(b.Build())

		time.Sleep(100 * time.Millisecond)

		code, err = v.Validate(pBuilder.Build(), responsesBuilder.Build())
		require.Equal(t, pb.TxValidationCode_VALID, code)
		require.NoError(t, err)
	})
}

type qeProvider struct {
	qe  *mocks.QueryExecutor
	err error
}

func newQueryExecutorProvider(qe *mocks.QueryExecutor) *qeProvider {
	return &qeProvider{
		qe: qe,
	}
}

func (p *qeProvider) NewQueryExecutor() (ledger.QueryExecutor, error) {
	return p.qe, nil
}

type mockPolicyEvaluator struct {
	err map[string]error
}

func newMockPolicyEvaluator() *mockPolicyEvaluator {
	return &mockPolicyEvaluator{
		err: make(map[string]error),
	}
}

func (pe *mockPolicyEvaluator) WithError(policyBytes []byte, err error) *mockPolicyEvaluator {
	pe.err[string(policyBytes)] = err
	return pe
}

func (pe *mockPolicyEvaluator) Evaluate(policyBytes []byte, signatureSet []*protoutil.SignedData) error {
	return pe.err[string(policyBytes)]
}
