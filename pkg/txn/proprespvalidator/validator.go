/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package proprespvalidator

import (
	"bytes"
	"fmt"

	"github.com/bluele/gcache"
	"github.com/hyperledger/fabric/extensions/gossip/api"

	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric-protos-go/common"
	mp "github.com/hyperledger/fabric-protos-go/msp"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	commonerrors "github.com/hyperledger/fabric/common/errors"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	validation "github.com/hyperledger/fabric/core/handlers/validation/api/policies"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/pkg/errors"
)

var logger = logging.NewLogger("txdelegationfilter")

const (
	errDuplicateIdentity = "Endorsement policy evaluation failure might be caused by duplicated identities"
)

type queryExecutorProvider interface {
	NewQueryExecutor() (ledger.QueryExecutor, error)
}

type validator struct {
	channelID             string
	queryExecutorProvider queryExecutorProvider
	policyEvaluator       validation.PolicyEvaluator
	idDeserializer        msp.IdentityDeserializer
	ccDataCache           gcache.Cache
}

func newValidator(channelID string, qeProvider queryExecutorProvider, pe validation.PolicyEvaluator, des msp.IdentityDeserializer, blockPublisher api.BlockPublisher) *validator {
	v := &validator{
		channelID:             channelID,
		queryExecutorProvider: qeProvider,
		policyEvaluator:       pe,
		idDeserializer:        des,
		ccDataCache: gcache.New(0).LoaderFunc(func(ccName interface{}) (interface{}, error) {
			return createCCDefinition(ccName.(string), qeProvider)
		}).Build(),
	}

	blockPublisher.AddCCUpgradeHandler(func(txMetadata api.TxMetadata, chaincodeName string) error {
		logger.Infof("[%s] Chaincode [%s] was upgraded. Resetting chaincode data cache", channelID, chaincodeName)
		v.ccDataCache.Purge()
		return nil
	})
	return v
}

// Validate executes vscc validation for proposal responses
func (v *validator) Validate(proposal *pb.SignedProposal, proposalResponses []*pb.ProposalResponse) (pb.TxValidationCode, error) {
	// Validate that the proposal responses match and that the returned status is SUCCESS
	err := v.validateResponses(proposalResponses)
	if err != nil {
		return pb.TxValidationCode_INVALID_OTHER_REASON, err
	}

	// Validate that the endorsement policy is satisfied
	endorsements := make([]*pb.Endorsement, len(proposalResponses))
	for i, pr := range proposalResponses {
		endorsements[i] = pr.Endorsement
	}

	return v.validateEndorsements(proposal, proposalResponses[0].Payload, endorsements)
}

func (v *validator) validateEndorsements(proposal *pb.SignedProposal, prespBytes []byte, endorsements []*pb.Endorsement) (pb.TxValidationCode, error) {
	chaincodeAction, err := getChaincodeAction(prespBytes)
	if err != nil {
		return pb.TxValidationCode_INVALID_OTHER_REASON, err
	}

	code, err := validateChaincodeAction(proposal, chaincodeAction)
	if err != nil {
		return code, err
	}

	ccID := chaincodeAction.ChaincodeId.Name

	var wrNamespace []string

	// Need to validate the policy of the invoked chaincode
	wrNamespace = append(wrNamespace, ccID)

	// Get the namespaces of all the chaincodes that were written to - the policy of these chaincodes also need to be validated.
	wrAdditionalNamespaces, code, err := getNamespacesFromWriteSets(chaincodeAction)
	if err != nil {
		return code, err
	}
	wrNamespace = append(wrNamespace, wrAdditionalNamespaces...)

	// Validate each write according to its chaincode's endorsement policy
	for _, ns := range wrNamespace {
		code, err := v.validateEndorsementPolicy(ns, chaincodeAction, prespBytes, endorsements)
		if err != nil {
			return code, err
		}
	}

	return pb.TxValidationCode_VALID, nil
}

func (v *validator) validateEndorsementPolicy(ccID string, chaincodeAction *pb.ChaincodeAction, prespBytes []byte, endorsements []*pb.Endorsement) (pb.TxValidationCode, error) {
	// Get latest chaincode version, vscc and validate policy
	ccd, err := v.getCCDefinition(ccID)
	if err != nil {
		return pb.TxValidationCode_INVALID_OTHER_REASON, err
	}

	// if the namespace corresponds to the cc that was originally
	// invoked, we check that the version of the cc that was
	// invoked corresponds to the version that lscc has returned
	if ccID == chaincodeAction.ChaincodeId.Name && ccd.Version != chaincodeAction.ChaincodeId.Version {
		return pb.TxValidationCode_EXPIRED_CHAINCODE, errors.Errorf("chaincode %s:%s/%s didn't match %s:%s/%s in lscc", ccID, chaincodeAction.ChaincodeId.Version, v.channelID, ccd.Name, ccd.Version, v.channelID)
	}

	if err := v.validatePolicyForCC(prespBytes, endorsements, ccd.Policy); err != nil {
		switch err.(type) {
		case *commonerrors.VSCCEndorsementPolicyError:
			return pb.TxValidationCode_ENDORSEMENT_POLICY_FAILURE, err
		default:
			return pb.TxValidationCode_INVALID_OTHER_REASON, err
		}
	}

	return pb.TxValidationCode_VALID, nil
}

func (v *validator) validatePolicyForCC(prespBytes []byte, endorsements []*pb.Endorsement, policyBytes []byte) error {
	signatureSet, err := deduplicateIdentity(prespBytes, endorsements)
	if err != nil {
		return policyErr(err)
	}

	// evaluate the signature set against the policy
	err = v.policyEvaluator.Evaluate(policyBytes, signatureSet)
	if err == nil {
		return nil
	}

	logger.Warnf("Endorsement policy failure err: %s", err.Error())
	if len(signatureSet) < len(endorsements) {
		// Warning: duplicated identities exist, endorsement failure might be cause by this reason
		return policyErr(errors.New(errDuplicateIdentity))
	}
	return policyErr(fmt.Errorf("VSCC error: endorsement policy failure, err: %s", err))
}

func (v *validator) getCCDefinition(ccid string) (*ccprovider.ChaincodeData, error) {
	ccData, err := v.ccDataCache.Get(ccid)
	if err != nil {
		return nil, err
	}

	return ccData.(*ccprovider.ChaincodeData), nil
}

func (v *validator) validateResponses(proposalResponses []*pb.ProposalResponse) error {
	if len(proposalResponses) == 0 {
		return errors.New("no proposal responses")
	}

	var r1 *pb.ProposalResponse
	for _, r := range proposalResponses {
		if err := v.validateResponse(r); err != nil {
			return err
		}
		if r1 == nil {
			r1 = r
		} else if !matches(r1, r) {
			return errors.New("one or more proposal responses do not match")
		}
	}

	return nil
}

func (v *validator) validateResponse(response *pb.ProposalResponse) error {
	if response.Response == nil {
		return errors.New("nil chaincode response")
	}
	if response.Response.Status < int32(cb.Status_SUCCESS) || response.Response.Status >= int32(cb.Status_BAD_REQUEST) {
		return errors.New("failed chaincode response status")
	}
	if err := v.validateSignature(response); err != nil {
		return err
	}
	return nil
}

func (v *validator) validateSignature(res *pb.ProposalResponse) error {
	if res.GetEndorsement() == nil {
		return errors.New("missing endorsement in proposal response")
	}
	creatorID := res.GetEndorsement().Endorser

	identity, err := v.idDeserializer.DeserializeIdentity(creatorID)
	if err != nil {
		return errors.WithMessage(err, "unable to deserialize identity")
	}

	if err := identity.Validate(); err != nil {
		return errors.WithMessage(err, "the creator certificate is not valid")
	}

	// check the signature against the endorser and payload hash
	digest := append(res.GetPayload(), res.GetEndorsement().Endorser...)

	// validate the signature
	if err := identity.Verify(digest, res.GetEndorsement().Signature); err != nil {
		return errors.WithMessage(err, "the creator's signature over the proposal response is not valid")
	}

	return nil
}

func getCCSpec(signedProp *pb.SignedProposal) (*pb.ChaincodeSpec, error) {
	prop, err := protoutil.UnmarshalProposal(signedProp.ProposalBytes)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to extract proposal from proposal bytes")
	}

	cis, err := getChaincodeInvocationSpec(prop)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get chaincode invocation spec")
	}

	if cis.ChaincodeSpec == nil {
		return nil, errors.New("chaincode spec is nil")
	}

	return cis.ChaincodeSpec, nil
}

func txWritesToNamespace(ns *rwsetutil.NsRwSet) bool {
	if ns.KvRwSet != nil && len(ns.KvRwSet.Writes) > 0 {
		return true
	}
	for _, c := range ns.CollHashedRwSets {
		if c.HashedRwSet != nil && len(c.HashedRwSet.HashedWrites) > 0 {
			return true
		}
	}
	return false
}

func deduplicateIdentity(prespBytes []byte, endorsements []*pb.Endorsement) ([]*protoutil.SignedData, error) {
	// build the signature set for the evaluation
	signatureSet := []*protoutil.SignedData{}
	signatureMap := make(map[string]struct{})
	// loop through each of the endorsements and build the signature set
	for _, endorsement := range endorsements {
		//unmarshal endorser bytes
		serializedIdentity := &mp.SerializedIdentity{}
		if err := proto.Unmarshal(endorsement.Endorser, serializedIdentity); err != nil {
			logger.Errorf("Unmarshal endorser error: %s", err)
			return nil, policyErr(fmt.Errorf("Unmarshal endorser error: %s", err))
		}
		identity := serializedIdentity.Mspid + string(serializedIdentity.IdBytes)
		if _, ok := signatureMap[identity]; ok {
			// Endorsement with the same identity has already been added
			logger.Warnf("Ignoring duplicated identity, Mspid: %s, pem:\n%s", serializedIdentity.Mspid, serializedIdentity.IdBytes)
			continue
		}
		data := make([]byte, len(prespBytes)+len(endorsement.Endorser))
		copy(data, prespBytes)
		copy(data[len(prespBytes):], endorsement.Endorser)
		signatureSet = append(signatureSet, &protoutil.SignedData{
			// set the data that is signed; concatenation of proposal response bytes and endorser ID
			Data: data,
			// set the identity that signs the message: it's the endorser
			Identity: endorsement.Endorser,
			// set the signature
			Signature: endorsement.Signature})
		signatureMap[identity] = struct{}{}
	}

	logger.Infof("Signature set is of size %d out of %d endorsement(s)", len(signatureSet), len(endorsements))
	return signatureSet, nil
}

func policyErr(err error) *commonerrors.VSCCEndorsementPolicyError {
	return &commonerrors.VSCCEndorsementPolicyError{
		Err: err,
	}
}

func getChaincodeAction(prespBytes []byte) (*pb.ChaincodeAction, error) {
	prp := &pb.ProposalResponsePayload{}
	if err := proto.Unmarshal(prespBytes, prp); err != nil {
		return nil, errors.WithMessage(err, "failed to unmarshal proposal response payload")
	}

	chaincodeAction := &pb.ChaincodeAction{}
	if err := proto.Unmarshal(prp.Extension, chaincodeAction); err != nil {
		return nil, err
	}

	return chaincodeAction, nil
}

func validateChaincodeAction(proposal *pb.SignedProposal, chaincodeAction *pb.ChaincodeAction) (pb.TxValidationCode, error) {
	cis, err := getCCSpec(proposal)
	if err != nil {
		return pb.TxValidationCode_INVALID_OTHER_REASON, err
	}

	if cis.ChaincodeId == nil {
		return pb.TxValidationCode_INVALID_OTHER_REASON, errors.New("nil ChaincodeId in header extension")
	}

	if chaincodeAction.ChaincodeId == nil {
		return pb.TxValidationCode_INVALID_OTHER_REASON, errors.New("nil ChaincodeId in ChaincodeAction")
	}

	// sanity check on ccID
	ccID := cis.ChaincodeId.Name
	if ccID == "" {
		return pb.TxValidationCode_INVALID_OTHER_REASON, errors.New("invalid chaincode ID in proposal")
	}
	if ccID != chaincodeAction.ChaincodeId.Name {
		return pb.TxValidationCode_INVALID_OTHER_REASON, errors.Errorf("inconsistent chaincode ID info (%s/%s)", ccID, chaincodeAction.ChaincodeId.Name)
	}
	if chaincodeAction.ChaincodeId.Version == "" {
		return pb.TxValidationCode_INVALID_OTHER_REASON, errors.New("invalid chaincode version in ChaincodeAction")
	}

	// chaincode event
	if chaincodeAction.Events != nil {
		ccEvent := &pb.ChaincodeEvent{}
		if err = proto.Unmarshal(chaincodeAction.Events, ccEvent); err != nil {
			return pb.TxValidationCode_INVALID_OTHER_REASON, errors.Wrapf(err, "invalid chaincode event")
		}
		if ccEvent.ChaincodeId != ccID {
			return pb.TxValidationCode_INVALID_OTHER_REASON, errors.Errorf("chaincode event chaincode id does not match chaincode action chaincode id")
		}
	}

	return pb.TxValidationCode_VALID, nil
}

func getNamespacesFromWriteSets(chaincodeAction *pb.ChaincodeAction) ([]string, pb.TxValidationCode, error) {
	txRWSet := &rwsetutil.TxRwSet{}
	if err := txRWSet.FromProtoBytes(chaincodeAction.Results); err != nil {
		return nil, pb.TxValidationCode_BAD_RWSET, errors.WithMessage(err, "txRWSet.FromProtoBytes failed")
	}

	var wrNamespace []string
	for _, ns := range txRWSet.NsRwSets {
		if !txWritesToNamespace(ns) {
			continue
		}

		if ns.NameSpace == "lscc" {
			return nil, pb.TxValidationCode_ILLEGAL_WRITESET, errors.Errorf("chaincode %s attempted to write to the namespace of LSCC", chaincodeAction.ChaincodeId.Name)
		}

		// Check to make sure we did not already populate this chaincode
		// name to avoid checking the same namespace twice
		if ns.NameSpace != chaincodeAction.ChaincodeId.Name {
			wrNamespace = append(wrNamespace, ns.NameSpace)
		}
	}
	return wrNamespace, pb.TxValidationCode_VALID, nil
}

func createCCDefinition(ccid string, qeProvider queryExecutorProvider) (*ccprovider.ChaincodeData, error) {
	qe, err := qeProvider.NewQueryExecutor()
	if err != nil {
		return nil, errors.WithMessage(err, "could not retrieve QueryExecutor")
	}
	defer qe.Done()

	bytes, err := qe.GetState("lscc", ccid)
	if err != nil {
		return nil, &commonerrors.VSCCInfoLookupFailureError{
			Reason: fmt.Sprintf("Could not retrieve state for chaincode %s, error %s", ccid, err),
		}
	}

	if bytes == nil {
		return nil, errors.Errorf("lscc's state for [%s] not found.", ccid)
	}

	cd := &ccprovider.ChaincodeData{}
	err = proto.Unmarshal(bytes, cd)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshalling ChaincodeQueryResponse failed")
	}

	if cd.Vscc == "" {
		return nil, errors.Errorf("lscc's state for [%s] is invalid, vscc field must be set", ccid)
	}

	if len(cd.Policy) == 0 {
		return nil, errors.Errorf("lscc's state for [%s] is invalid, policy field must be set", ccid)
	}

	return cd, err
}

func matches(r1, r2 *pb.ProposalResponse) bool {
	return r1.Response.Status == r2.Response.Status &&
		bytes.Equal(r1.Payload, r2.Payload) &&
		bytes.Equal(r1.Response.Payload, r2.Response.Payload)
}

func getChaincodeInvocationSpec(proposal *pb.Proposal) (*pb.ChaincodeInvocationSpec, error) {
	cpp, err := protoutil.UnmarshalChaincodeProposalPayload(proposal.Payload)
	if err != nil {
		return nil, errors.WithMessage(err, "failed GetChaincodeInvocationSpec")
	}

	cis, err := protoutil.UnmarshalChaincodeInvocationSpec(cpp.Input)
	if err != nil {
		return nil, errors.WithMessage(err, "failed GetChaincodeInvocationSpec")
	}

	return cis, nil
}
