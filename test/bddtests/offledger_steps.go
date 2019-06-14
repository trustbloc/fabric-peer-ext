/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"encoding/json"

	"github.com/DATA-DOG/godog"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/common"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-test-common/bddtests"
)

// OffLedgerSteps ...
type OffLedgerSteps struct {
	BDDContext *bddtests.BDDContext
	content    string
	address    string
}

// NewOffLedgerSteps ...
func NewOffLedgerSteps(context *bddtests.BDDContext) *OffLedgerSteps {
	return &OffLedgerSteps{BDDContext: context}
}

// DefineOffLedgerCollectionConfig defines a new off-ledger data collection configuration
func (d *OffLedgerSteps) DefineOffLedgerCollectionConfig(id, name, policy string, requiredPeerCount, maxPeerCount int32, timeToLive string) {
	d.BDDContext.DefineCollectionConfig(id,
		func(channelID string) (*common.CollectionConfig, error) {
			sigPolicy, err := d.newChaincodePolicy(policy, channelID)
			if err != nil {
				return nil, errors.Wrapf(err, "error creating collection policy for collection [%s]", name)
			}
			return newOffLedgerCollectionConfig(name, requiredPeerCount, maxPeerCount, timeToLive, sigPolicy), nil
		},
	)
}

// DefineDCASCollectionConfig defines a new DCAS collection configuration
func (d *OffLedgerSteps) DefineDCASCollectionConfig(id, name, policy string, requiredPeerCount, maxPeerCount int32, timeToLive string) {
	d.BDDContext.DefineCollectionConfig(id,
		func(channelID string) (*common.CollectionConfig, error) {
			sigPolicy, err := d.newChaincodePolicy(policy, channelID)
			if err != nil {
				return nil, errors.Wrapf(err, "error creating collection policy for collection [%s]", name)
			}
			return newDCASCollectionConfig(name, requiredPeerCount, maxPeerCount, timeToLive, sigPolicy), nil
		},
	)
}

func (d *OffLedgerSteps) setCASVariable(varName, value string) error {
	casKey := GetCASKey([]byte(value))
	bddtests.SetVar(varName, casKey)
	logger.Infof("Saving CAS key '%s' to variable '%s'", casKey, varName)
	return nil
}

func (d *OffLedgerSteps) defineOffLedgerCollectionConfig(id, collection, policy string, requiredPeerCount int, maxPeerCount int, timeToLive string) error {
	logger.Infof("Defining off-ledger collection config [%s] for collection [%s] - policy=[%s], requiredPeerCount=[%d], maxPeerCount=[%d], timeToLive=[%s]", id, collection, policy, requiredPeerCount, maxPeerCount, timeToLive)
	d.DefineOffLedgerCollectionConfig(id, collection, policy, int32(requiredPeerCount), int32(maxPeerCount), timeToLive)
	return nil
}

func (d *OffLedgerSteps) defineDCASCollectionConfig(id, collection, policy string, requiredPeerCount int, maxPeerCount int, timeToLive string) error {
	logger.Infof("Defining DCAS collection config [%s] for collection [%s] - policy=[%s], requiredPeerCount=[%d], maxPeerCount=[%d], timeToLive=[%s]", id, collection, policy, requiredPeerCount, maxPeerCount, timeToLive)
	d.DefineDCASCollectionConfig(id, collection, policy, int32(requiredPeerCount), int32(maxPeerCount), timeToLive)
	return nil
}

func (d *OffLedgerSteps) defineNewAccount(id, owner string, balance int, varName string) error {
	logger.Infof("Defining new account (%s:%s:%d)", id, owner, balance)

	account := &AccountOperation{
		OperationType: "create",
		ID:            id,
		Owner:         owner,
		Balance:       balance,
	}
	bytes, err := json.Marshal(account)
	if err != nil {
		return err
	}
	bddtests.SetVar(varName, string(bytes))
	return nil
}

func (d *OffLedgerSteps) updateAccountBalance(varName string, balance int) error {
	logger.Infof("Updating account balance (%s:%d)", varName, balance)

	strAccountBytes, ok := bddtests.GetVar(varName)
	if !ok {
		return errors.Errorf("account not found in variable [%s]", varName)
	}

	account := &AccountOperation{}
	err := json.Unmarshal([]byte(strAccountBytes), account)
	if err != nil {
		return errors.WithMessagef(err, "error unmarshalling account [%s]", varName)
	}

	account.OperationType = "update"
	bytes, err := json.Marshal(account)
	if err != nil {
		return errors.WithMessagef(err, "error marshalling account %v", account)
	}
	bddtests.SetVar(varName, string(bytes))
	return nil
}

func (d *OffLedgerSteps) checkAccountQueryResponse(varName string, numItems int) error {
	strValues, ok := bddtests.GetVar(varName)
	if !ok {
		return errors.Errorf("no value stored in variable [%s]", varName)
	}

	var values [][]byte
	if err := json.Unmarshal([]byte(strValues), &values); err != nil {
		return err
	}

	if len(values) != numItems {
		return errors.Errorf("expecting %d accounts but got %d", numItems, len(values))
	}

	for _, accountBytes := range values {
		account := &AccountOperation{}
		if err := json.Unmarshal(accountBytes, account); err != nil {
			return errors.Errorf("error unmarshalling account: %s", err)
		}
	}
	return nil
}

func (d *OffLedgerSteps) newChaincodePolicy(ccPolicy, channelID string) (*common.SignaturePolicyEnvelope, error) {
	return bddtests.NewChaincodePolicy(d.BDDContext, ccPolicy, channelID)
}

func newOffLedgerCollectionConfig(collName string, requiredPeerCount, maxPeerCount int32, timeToLive string, policy *common.SignaturePolicyEnvelope) *common.CollectionConfig {
	return &common.CollectionConfig{
		Payload: &common.CollectionConfig_StaticCollectionConfig{
			StaticCollectionConfig: &common.StaticCollectionConfig{
				Name:              collName,
				Type:              common.CollectionType_COL_OFFLEDGER,
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

func newDCASCollectionConfig(collName string, requiredPeerCount, maxPeerCount int32, timeToLive string, policy *common.SignaturePolicyEnvelope) *common.CollectionConfig {
	return &common.CollectionConfig{
		Payload: &common.CollectionConfig_StaticCollectionConfig{
			StaticCollectionConfig: &common.StaticCollectionConfig{
				Name:              collName,
				Type:              common.CollectionType_COL_DCAS,
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

// RegisterSteps registers off-ledger steps
func (d *OffLedgerSteps) RegisterSteps(s *godog.Suite) {
	s.BeforeScenario(d.BDDContext.BeforeScenario)
	s.AfterScenario(d.BDDContext.AfterScenario)
	s.Step(`^variable "([^"]*)" is assigned the CAS key of value "([^"]*)"$`, d.setCASVariable)
	s.Step(`^off-ledger collection config "([^"]*)" is defined for collection "([^"]*)" as policy="([^"]*)", requiredPeerCount=(\d+), maxPeerCount=(\d+), and timeToLive=([^"]*)$`, d.defineOffLedgerCollectionConfig)
	s.Step(`^DCAS collection config "([^"]*)" is defined for collection "([^"]*)" as policy="([^"]*)", requiredPeerCount=(\d+), maxPeerCount=(\d+), and timeToLive=([^"]*)$`, d.defineDCASCollectionConfig)
	s.Step(`^the account with ID "([^"]*)", owner "([^"]*)" and a balance of (\d+) is created and stored to variable "([^"]*)"$`, d.defineNewAccount)
	s.Step(`^the account stored in variable "([^"]*)" is updated with a balance of (\d+)$`, d.updateAccountBalance)
	s.Step(`^the variable "([^"]*)" contains (\d+) accounts$`, d.checkAccountQueryResponse)
}

// AccountOperation contains account information
type AccountOperation struct {
	OperationType string `json:"operationType"`
	ID            string `json:"id"`
	Owner         string `json:"owner"`
	Balance       int    `json:"balance"`
}
