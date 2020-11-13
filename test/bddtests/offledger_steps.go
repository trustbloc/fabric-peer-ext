/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"encoding/json"
	"sort"

	"github.com/cucumber/godog"
	"github.com/hyperledger/fabric-protos-go/common"
	pb "github.com/hyperledger/fabric-protos-go/peer"
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
		func(channelID string) (*pb.CollectionConfig, error) {
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
		func(channelID string) (*pb.CollectionConfig, error) {
			sigPolicy, err := d.newChaincodePolicy(policy, channelID)
			if err != nil {
				return nil, errors.Wrapf(err, "error creating collection policy for collection [%s]", name)
			}
			return newDCASCollectionConfig(name, requiredPeerCount, maxPeerCount, timeToLive, sigPolicy), nil
		},
	)
}

func (d *OffLedgerSteps) setCASVariable(varName, value string) error {
	casKey, err := getCASKey([]byte(value))
	if err != nil {
		return err
	}

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
		Order:         1,
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
	account.Balance = balance
	account.Order++ // Increment the operation order since we sort the operations based on this field

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

	var kvs []*KV
	if err := json.Unmarshal([]byte(strValues), &kvs); err != nil {
		return errors.Errorf("error unmarshalling Key-Values: %s", err)
	}

	if len(kvs) != numItems {
		return errors.Errorf("expecting %d accounts but got %d", numItems, len(kvs))
	}

	return nil
}

func (d *OffLedgerSteps) validateAccountAtIndex(varName string, idx int, keyExpr string, id string, owner string, balance int) error {
	strValues, ok := bddtests.GetVar(varName)
	if !ok {
		return errors.Errorf("no value stored in variable [%s]", varName)
	}

	var kvs []*KV
	if err := json.Unmarshal([]byte(strValues), &kvs); err != nil {
		return errors.Errorf("error unmarshalling Key-Values: %s", err)
	}

	if len(kvs) <= idx {
		return errors.Errorf("index %d out of range - only have %d queryResults", idx, len(kvs))
	}

	queryResults, err := unmarshalAccountOperations(kvs)
	if err != nil {
		return err
	}

	queryResult := queryResults[idx]

	logger.Infof("Got account key [%s]", queryResult.key)

	keys, err := bddtests.ResolveAllVars(keyExpr)
	if err != nil {
		return errors.WithMessagef(err, "invalid key expression [%s]", keyExpr)
	}
	if len(keys) != 1 {
		return errors.Errorf("expecting only one key but got %d", len(keys))
	}

	key := keys[0]
	if queryResult.key != key {
		return errors.Errorf("expecting key [%s] but got [%s]", key, queryResult.key)
	}

	account := queryResult.account
	if account.ID != id {
		return errors.Errorf("expecting account ID [%s] but got [%s]", id, account.ID)
	}
	if account.Owner != owner {
		return errors.Errorf("expecting account owner [%s] but got [%s]", owner, account.Owner)
	}
	if account.Balance != balance {
		return errors.Errorf("expecting account balance [%d] but got [%d]", balance, account.Balance)
	}
	return nil
}

func (d *OffLedgerSteps) newChaincodePolicy(ccPolicy, channelID string) (*common.SignaturePolicyEnvelope, error) {
	return bddtests.NewChaincodePolicy(d.BDDContext, ccPolicy, channelID)
}

func newOffLedgerCollectionConfig(collName string, requiredPeerCount, maxPeerCount int32, timeToLive string, policy *common.SignaturePolicyEnvelope) *pb.CollectionConfig {
	return &pb.CollectionConfig{
		Payload: &pb.CollectionConfig_StaticCollectionConfig{
			StaticCollectionConfig: &pb.StaticCollectionConfig{
				Name:              collName,
				Type:              pb.CollectionType_COL_OFFLEDGER,
				RequiredPeerCount: requiredPeerCount,
				MaximumPeerCount:  maxPeerCount,
				TimeToLive:        timeToLive,
				MemberOrgsPolicy: &pb.CollectionPolicyConfig{
					Payload: &pb.CollectionPolicyConfig_SignaturePolicy{
						SignaturePolicy: policy,
					},
				},
			},
		},
	}
}

func newDCASCollectionConfig(collName string, requiredPeerCount, maxPeerCount int32, timeToLive string, policy *common.SignaturePolicyEnvelope) *pb.CollectionConfig {
	return &pb.CollectionConfig{
		Payload: &pb.CollectionConfig_StaticCollectionConfig{
			StaticCollectionConfig: &pb.StaticCollectionConfig{
				Name:              collName,
				Type:              pb.CollectionType_COL_DCAS,
				RequiredPeerCount: requiredPeerCount,
				MaximumPeerCount:  maxPeerCount,
				TimeToLive:        timeToLive,
				MemberOrgsPolicy: &pb.CollectionPolicyConfig{
					Payload: &pb.CollectionPolicyConfig_SignaturePolicy{
						SignaturePolicy: policy,
					},
				},
			},
		},
	}
}

func (d *OffLedgerSteps) setVariable(varName, value string) error {
	if err := bddtests.ResolveVarsInExpression(&value); err != nil {
		return err
	}

	logger.Infof("Setting var [%s] to [%s]", varName, value)

	bddtests.SetVar(varName, value)

	return nil
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
	s.Step(`^the variable "([^"]*)" contains an account at index (\d+) with Key "([^"]*)", ID "([^"]*)", Owner "([^"]*)", and Balance (\d+)$`, d.validateAccountAtIndex)
	s.Step(`^variable "([^"]*)" is assigned the uncanonicalized JSON value '([^']*)'$`, d.setVariable)
}

// AccountOperation contains account information
type AccountOperation struct {
	OperationType string `json:"operationType"`
	ID            string `json:"id"`
	Owner         string `json:"owner"`
	Balance       int    `json:"balance"`
	Order         int    `json:"order"`
}

// KV -- QueryResult for range/execute query. Holds a key and corresponding value.
type KV struct {
	Namespace string `protobuf:"bytes,1,opt,name=namespace,proto3" json:"namespace,omitempty"`
	Key       string `protobuf:"bytes,2,opt,name=key,proto3" json:"key,omitempty"`
	Value     []byte `protobuf:"bytes,3,opt,name=value,proto3" json:"value,omitempty"`
}

type accountOperationInfo struct {
	key     string
	account *AccountOperation
}

// unmarshalAccountOperations unmarshals the account operations and returns them in sorted order
func unmarshalAccountOperations(kvs []*KV) ([]*accountOperationInfo, error) {
	accounts := make([]*accountOperationInfo, len(kvs))
	for i, kv := range kvs {
		account := &AccountOperation{}
		if err := json.Unmarshal(kv.Value, account); err != nil {
			return nil, errors.Errorf("error unmarshalling account: %s", err)
		}
		accounts[i] = &accountOperationInfo{
			key:     kv.Key,
			account: account,
		}
	}
	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].account.Order < accounts[j].account.Order
	})
	return accounts, nil
}
