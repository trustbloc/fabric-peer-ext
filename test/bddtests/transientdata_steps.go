/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"github.com/DATA-DOG/godog"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/common"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-test-common/bddtests"
)

var logger = logging.NewLogger("test-logger")

// TransientDataSteps ...
type TransientDataSteps struct {
	BDDContext *bddtests.BDDContext
	content    string
	address    string
}

// NewTransientDataSteps returns transient data BDD steps
func NewTransientDataSteps(context *bddtests.BDDContext) *TransientDataSteps {
	return &TransientDataSteps{BDDContext: context}
}

// DefineTransientCollectionConfig defines a new transient data collection configuration
func (d *TransientDataSteps) DefineTransientCollectionConfig(id, name, policy string, requiredPeerCount, maxPeerCount int32, timeToLive string) {
	d.BDDContext.DefineCollectionConfig(id,
		func(channelID string) (*common.CollectionConfig, error) {
			sigPolicy, err := d.newChaincodePolicy(policy, channelID)
			if err != nil {
				return nil, errors.Wrapf(err, "error creating collection policy for collection [%s]", name)
			}
			return newTransientCollectionConfig(name, requiredPeerCount, maxPeerCount, timeToLive, sigPolicy), nil
		},
	)
}

func (d *TransientDataSteps) defineTransientCollectionConfig(id, collection, policy string, requiredPeerCount int, maxPeerCount int, timeToLive string) error {
	logger.Infof("Defining transient collection config [%s] for collection [%s] - policy=[%s], requiredPeerCount=[%d], maxPeerCount=[%d], timeToLive=[%s]", id, collection, policy, requiredPeerCount, maxPeerCount, timeToLive)
	d.DefineTransientCollectionConfig(id, collection, policy, int32(requiredPeerCount), int32(maxPeerCount), timeToLive)
	return nil
}

func (d *TransientDataSteps) newChaincodePolicy(ccPolicy, channelID string) (*common.SignaturePolicyEnvelope, error) {
	return bddtests.NewChaincodePolicy(d.BDDContext, ccPolicy, channelID)
}

// RegisterSteps registers transient data steps
func (d *TransientDataSteps) RegisterSteps(s *godog.Suite) {
	s.BeforeScenario(d.BDDContext.BeforeScenario)
	s.AfterScenario(d.BDDContext.AfterScenario)
	s.Step(`^transient collection config "([^"]*)" is defined for collection "([^"]*)" as policy="([^"]*)", requiredPeerCount=(\d+), maxPeerCount=(\d+), and timeToLive=([^"]*)$`, d.defineTransientCollectionConfig)
}

func newTransientCollectionConfig(collName string, requiredPeerCount, maxPeerCount int32, timeToLive string, policy *common.SignaturePolicyEnvelope) *common.CollectionConfig {
	return &common.CollectionConfig{
		Payload: &common.CollectionConfig_StaticCollectionConfig{
			StaticCollectionConfig: &common.StaticCollectionConfig{
				Name:              collName,
				Type:              common.CollectionType_COL_TRANSIENT,
				RequiredPeerCount: requiredPeerCount,
				MaximumPeerCount:  maxPeerCount,
				TimeToLive:        timeToLive,
				MemberOrgsPolicy: &common.CollectionPolicyConfig{
					Payload: &common.CollectionPolicyConfig_SignaturePolicy{
						SignaturePolicy: policy,
					},
				},
			},
		},
	}
}
