/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package configcc

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/common/flogging"
	ccapi "github.com/hyperledger/fabric/extensions/chaincode/api"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
	ledgerconfig "github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/mgr"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/service"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/state"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/state/api"
)

var logger = flogging.MustGetLogger("configcc")

const (
	version = "v1"
)

type configMgr interface {
	Query(key *config.Criteria) ([]*config.KeyValue, error)
	Save(txID string, config *config.Config) error
	Delete(criteria *config.Criteria) error
}

type function func(shim.ChaincodeStubInterface, [][]byte) pb.Response

type configCC struct {
	validatorRegistry configValidatorRegistry
	functionRegistry  map[string]function
}

type configValidatorRegistry interface {
	ValidatorForKey(key *config.Key) config.Validator
}

// New returns a new configuration system chaincode
func New(validatorRegistry configValidatorRegistry) ccapi.UserCC {
	cc := &configCC{validatorRegistry: validatorRegistry}
	cc.initFunctionRegistry()
	return cc
}

// Name returns the name of this chaincode
func (cc *configCC) Name() string { return service.ConfigNS }

// Version returns the version of this chaincode
func (cc *configCC) Version() string { return version }

// Chaincode returns the chaincode implementation
func (cc *configCC) Chaincode() shim.Chaincode { return cc }

// GetDBArtifacts returns DB artifacts. For this chaincode there are no artifacts.
func (cc *configCC) GetDBArtifacts() map[string]*ccapi.DBArtifacts { return nil }

// Init will be deprecated in a future release
func (cc *configCC) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

// Invoke invokes the config SCC
func (cc *configCC) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	args := stub.GetArgs()
	if len(args) == 0 {
		return shim.Error(fmt.Sprintf("Function not provided. Expecting one of [%s]", cc.functionSet()))
	}

	functionName := string(args[0])
	f, ok := cc.functionRegistry[functionName]
	if !ok {
		return shim.Error(fmt.Sprintf("Invalid function: [%s]. Expecting one of [%s]", functionName, cc.functionSet()))
	}

	functionArgs := args[1:]

	logger.Debugf("Invoking f [%s] with args: %s", functionName, functionArgs)
	return f(stub, functionArgs)
}

// put saves configuration to the ledger
// args[0] - Is the JSON marshalled Config
func (cc *configCC) put(stub shim.ChaincodeStubInterface, args [][]byte) pb.Response {
	if len(args) == 0 {
		return shim.Error("config is empty - cannot be saved")
	}

	config := &config.Config{}
	if err := json.Unmarshal(args[0], config); err != nil {
		logger.Errorf("Error unmarshalling config: %s", err)
		return shim.Error(fmt.Sprintf("Error unmarshalling config: %s", err))
	}

	mgr := getConfigMgr(service.ConfigNS, state.NewShimStoreProvider(stub), cc.validatorRegistry)
	if err := mgr.Save(stub.GetTxID(), config); err != nil {
		logger.Errorf("Error saving config: %s", err)
		return shim.Error(fmt.Sprintf("Error saving config: %s", err))
	}

	return shim.Success(nil)
}

// get retrieves configuration from the ledger
// args[0] - Is the JSON marshalled Criteria
func (cc *configCC) get(stub shim.ChaincodeStubInterface, args [][]byte) pb.Response {
	if len(args) == 0 {
		return shim.Error("criteria not provided")
	}

	criteria, err := unmarshalCriteria(args[0])
	if err != nil {
		logger.Errorf("Error unmarshalling criteria: %s", err)
		return shim.Error(err.Error())
	}

	config, err := getConfigMgr(service.ConfigNS, state.NewShimStoreProvider(stub), cc.validatorRegistry).Query(criteria)
	if err != nil {
		logger.Errorf("Error getting config for criteria [%s]: %s", criteria, err)
		return shim.Error(fmt.Sprintf("error retrieving config: %s", err))
	}

	payload, err := marshalJSON(config)
	if err != nil {
		logger.Errorf("Error marshalling config: %s", err)
		return shim.Error(fmt.Sprintf("error marshalling config: %s", err))
	}

	return shim.Success(payload)
}

// remove deletes configuration from the ledger
// args[0] - Is the JSON marshalled Criteria
func (cc *configCC) remove(stub shim.ChaincodeStubInterface, args [][]byte) pb.Response {
	if len(args) == 0 {
		return shim.Error("criteria not provided")
	}

	criteria, err := unmarshalCriteria(args[0])
	if err != nil {
		logger.Errorf("Error unmarshalling criteria: %s", err)
		return shim.Error(err.Error())
	}

	if err := getConfigMgr(service.ConfigNS, state.NewShimStoreProvider(stub), cc.validatorRegistry).Delete(criteria); err != nil {
		logger.Errorf("Error deleting config for criteria [%s]: %s", criteria, err)
		return shim.Error(fmt.Sprintf("Error deleting config: %s", err))
	}

	return shim.Success(nil)
}

func (cc *configCC) initFunctionRegistry() {
	cc.functionRegistry = make(map[string]function)
	cc.functionRegistry["save"] = cc.put
	cc.functionRegistry["get"] = cc.get
	cc.functionRegistry["delete"] = cc.remove
}

// functionSet returns a string enumerating all available functions
func (cc *configCC) functionSet() string {
	var functionNames string
	for name := range cc.functionRegistry {
		if functionNames != "" {
			functionNames += ", "
		}
		functionNames += name
	}
	return functionNames
}

// unmarshalCriteria unmarshals the Criteria from the given JSON byte array
func unmarshalCriteria(bytes []byte) (*config.Criteria, error) {
	criteria := &config.Criteria{}
	if err := unmarshalJSON(bytes, criteria); err != nil {
		return criteria, errors.WithMessagef(err, "error unmarshalling criteria %s", bytes)
	}

	return criteria, nil
}

// getConfigMgr returns the config manager. This variable may be overridden by unit tests.
var getConfigMgr = func(ns string, sp api.StoreProvider, cfgValidatorRegistry configValidatorRegistry) configMgr {
	return ledgerconfig.NewUpdateManager(ns, sp, cfgValidatorRegistry)
}

// marshalJSON returns the JSON representation of the given value. This variable may be overridden by unit tests.
var marshalJSON = func(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// unmarshalJSON returns the unmarshalled value for the given JSON. This variable may be overridden by unit tests.
var unmarshalJSON = func(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
