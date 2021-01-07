/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validationpolicy

import (
	"encoding/binary"
	"sync"

	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/common/policydsl"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/pkg/errors"

	"github.com/trustbloc/fabric-peer-ext/pkg/common/discovery"
	"github.com/trustbloc/fabric-peer-ext/pkg/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/validation/validationresults"
)

var logger = flogging.MustGetLogger("ext_validation")

// traceLogger is used for very detailed logging
var traceLogger = flogging.MustGetLogger("ext_validation_trace")

// policy contains the validation policy parameters
type policy struct {
	// committerTransactionThreshold is the threshold at which the committer will validate the block, i.e. if
	// the number of transactions is less than this threshold then the committer will validate the block,
	// even though the committer does not have the validator role. If set to 0 then the committer won't perform validation
	// unless there are no validators available.
	committerTransactionThreshold int

	// singlePeerTransactionThreshold is the threshold at which only a single peer with the validator role will
	// validate the block, i.e. if the number of transactions is less than this threshold then only a single validator
	// is selected for validation.
	singlePeerTransactionThreshold int
}

// PolicyEvaluator evaluates the validation policy
type PolicyEvaluator struct {
	policy *policy
	*discovery.Discovery
	channelID       string
	policyValidator policies.Policy
	policyProvider  policies.Provider
	mutex           sync.RWMutex
}

// New returns a new Validation PolicyEvaluator
func New(channelID string, disc *discovery.Discovery, pp policies.Provider) *PolicyEvaluator {
	policy := &policy{
		committerTransactionThreshold:  config.GetValidationCommitterTransactionThreshold(),
		singlePeerTransactionThreshold: config.GetValidationSinglePeerTransactionThreshold(),
	}

	logger.Infof("[%s] Creating new policy evaluator...", channelID)

	return &PolicyEvaluator{
		channelID:      channelID,
		Discovery:      disc,
		policy:         policy,
		policyProvider: pp,
	}
}

// GetValidatingPeers returns the set of peers that are involved in validating the given block
func (p *PolicyEvaluator) GetValidatingPeers(block *cb.Block) (discovery.PeerGroup, error) {
	groups, err := newPeerGroupSelector(p.channelID, p.policy, p.Discovery, block).groups()
	if err != nil {
		return nil, err
	}

	peerMap := make(map[string]*discovery.Member)
	for _, pg := range groups {
		for _, p := range pg {
			peerMap[p.Endpoint] = p
		}
	}

	var peers []*discovery.Member
	for _, p := range peerMap {
		peers = append(peers, p)
	}

	return peers, nil
}

// GetTxFilter returns the transaction filter that determines whether or not the local peer
// should validate the transaction at a given index.
func (p *PolicyEvaluator) GetTxFilter(block *cb.Block) TxFilter {
	groups, err := newPeerGroupSelector(p.channelID, p.policy, p.Discovery, block).groups()
	if err != nil {
		logger.Warningf("Error calculating peer groups for block %d: %s. Will validate all transactions.", block.Header.Number, err)
		return TxFilterAcceptAll
	}

	logger.Debugf("[%s] All validator groups for block %d: %s", p.channelID, block.Header.Number, groups)

	return func(txIdx int) bool {
		peerGroupForTx := groups[txIdx%len(groups)]

		traceLogger.Debugf("[%s] Validator group for block %d, TxIdx [%d] is %s", p.channelID, block.Header.Number, txIdx, peerGroupForTx)

		return peerGroupForTx.ContainsLocal()
	}
}

// ValidateResults validates that the given results have come from a reliable source
// and that the validation policy has been satisfied.
func (p *PolicyEvaluator) ValidateResults(results []*validationresults.Results) error {
	// If one of the results in the set came from this peer then no need to validate.
	// If one of the results in the set is from another peer in our own org then validate
	// the signature to ensure the result came from our org.)
	for _, result := range results {
		if result.Local {
			logger.Debugf("[%s] No need to validate since results for block %d originated locally", p.channelID, result.BlockNumber)

			return nil
		}

		if result.MSPID != p.Self().MSPID {
			logger.Debugf("[%s] Ignoring validation results for block %d which came from [%s] which is in another org [%s]", p.channelID, results[0].BlockNumber, result.Endpoint, result.MSPID)

			continue
		}

		logger.Debugf("[%s] Validating results for block %d that came from [%s] which is in our own org", p.channelID, result.BlockNumber, result.Endpoint)

		policyValidator, err := p.getPolicyValidator()
		if err != nil {
			return errors.WithMessagef(err, "unable to get policy validator")
		}

		return policyValidator.EvaluateSignedData(getSignatureSet([]*validationresults.Results{result}))
	}

	return errors.Errorf("[%s] No validation results from the local org for block %d", p.channelID, results[0].BlockNumber)
}

func (p *PolicyEvaluator) getPolicyValidator() (policies.Policy, error) {
	var policyValidator policies.Policy

	p.mutex.RLock()
	policyValidator = p.policyValidator
	p.mutex.RUnlock()

	if policyValidator != nil {
		return policyValidator, nil
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.policyValidator != nil {
		return p.policyValidator, nil
	}

	// 'local-org' policy
	policyBytes, err := proto.Marshal(policydsl.SignedByMspMember(p.Self().MSPID))
	if err != nil {
		return nil, errors.WithMessage(err, "error marshaling 'local-org' policy")
	}

	policyValidator, _, err = p.policyProvider.NewPolicy(policyBytes)
	if err != nil {
		return nil, errors.WithMessage(err, "error creating 'local-org' policy evaluator")
	}

	p.policyValidator = policyValidator

	return policyValidator, nil
}

func getSignatureSet(validationResults []*validationresults.Results) []*protoutil.SignedData {
	var sigSet []*protoutil.SignedData

	for _, vr := range validationResults {
		signedData := &protoutil.SignedData{
			Data:      GetDataToSign(vr.BlockNumber, vr.TxFlags, vr.Identity),
			Signature: vr.Signature,
			Identity:  vr.Identity,
		}
		sigSet = append(sigSet, signedData)
	}

	return sigSet
}

// GetDataToSign appends the block number, Tx Flags, and the serialized identity
// into a byte buffer to be signed by the validator.
func GetDataToSign(blockNum uint64, txFlags, identity []byte) []byte {
	blockNumBytes := make([]byte, binary.MaxVarintLen64)
	binary.PutUvarint(blockNumBytes, blockNum)
	data := append(blockNumBytes, txFlags...)
	return append(data, identity...)
}

// TxFilter determines which transactions are to be validated
type TxFilter func(txIdx int) bool

// TxFilterAcceptAll accepts all transactions
var TxFilterAcceptAll = func(int) bool {
	return true
}
