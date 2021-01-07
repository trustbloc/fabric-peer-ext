/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/policies"
	mspi "github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/pkg/errors"
)

// PolicyProvider is a mock policy provider
type PolicyProvider struct {
}

// NewPolicyProvider returns a new mock policy provider
func NewPolicyProvider() *PolicyProvider {
	return &PolicyProvider{}
}

// NewPolicy returns a new policy evaluator
func (pp *PolicyProvider) NewPolicy(policyBytes []byte) (policies.Policy, proto.Message, error) {
	return NewPolicyEvaluator(), nil, nil
}

// PolicyEvaluator is a mock policy evaluator
type PolicyEvaluator struct {
}

// NewPolicyEvaluator returns a new mock policy ealuator
func NewPolicyEvaluator() *PolicyEvaluator {
	return &PolicyEvaluator{}
}

// EvaluateSignedData takes a set of SignedData and evaluates whether this set of signatures satisfies
// the policy with the given bytes
func (pe *PolicyEvaluator) EvaluateSignedData(signatureSet []*protoutil.SignedData) error {
	for _, data := range signatureSet {
		if len(data.Data) == 0 {
			return errors.New("Got empty data")
		}
		if len(data.Signature) == 0 {
			return errors.New("Got empty signature")
		}
		if len(data.Identity) == 0 {
			return errors.New("Got empty identity")
		}
	}
	return nil
}

// EvaluateIdentities takes an array of identities and evaluates whether
// they satisfy the policy
func (pe *PolicyEvaluator) EvaluateIdentities(identities []mspi.Identity) error {
	panic("not implemented")
}
