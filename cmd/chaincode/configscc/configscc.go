/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package configscc

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/scc"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
	ledgerconfig "github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/mgr"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/service"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/state"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/state/api"
)

var logger = flogging.MustGetLogger("configscc")

const (
	sccPath = "github.com/trustbloc/fabric-peer-ext/cmd/chaincode/configscc"
)

type configMgr interface {
	Query(key *config.Criteria) ([]*config.KeyValue, error)
	Save(txID string, config *config.Config) error
	Delete(criteria *config.Criteria) error
}

type function func(shim.ChaincodeStubInterface, [][]byte) pb.Response

type configSCC struct {
	functionRegistry map[string]function
}

// New returns a new configuration system chaincode
func New() scc.SelfDescribingSysCC {
	cc := &configSCC{}
	cc.initFunctionRegistry()
	return cc
}

func (scc *configSCC) Name() string              { return service.ConfigNS }
func (scc *configSCC) Path() string              { return sccPath }
func (scc *configSCC) InitArgs() [][]byte        { return nil }
func (scc *configSCC) Chaincode() shim.Chaincode { return scc }
func (scc *configSCC) InvokableExternal() bool   { return true }
func (scc *configSCC) InvokableCC2CC() bool      { return true }
func (scc *configSCC) Enabled() bool             { return true }

// Init initializes the config SCC
func (scc *configSCC) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

// Invoke invokes the config SCC
func (scc *configSCC) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	args := stub.GetArgs()
	if len(args) == 0 {
		return shim.Error(fmt.Sprintf("Function not provided. Expecting one of [%s]", scc.functionSet()))
	}

	functionName := string(args[0])
	f, ok := scc.functionRegistry[functionName]
	if !ok {
		return shim.Error(fmt.Sprintf("Invalid function: [%s]. Expecting one of [%s]", functionName, scc.functionSet()))
	}

	functionArgs := args[1:]

	logger.Debugf("Invoking f [%s] with args: %s", functionName, functionArgs)
	return f(stub, functionArgs)
}

// put saves configuration to the ledger
// args[0] - Is the JSON marshalled Config
func (scc *configSCC) put(stub shim.ChaincodeStubInterface, args [][]byte) pb.Response {
	if len(args) == 0 {
		return shim.Error("config is empty - cannot be saved")
	}

	config := &config.Config{}
	if err := json.Unmarshal(args[0], config); err != nil {
		logger.Errorf("Error unmarshalling config: %s", err)
		return shim.Error(fmt.Sprintf("Error unmarshalling config: %s", err))
	}

	mgr := getConfigMgr(service.ConfigNS, state.NewShimStoreProvider(stub))
	if err := mgr.Save(stub.GetTxID(), config); err != nil {
		logger.Errorf("Error saving config: %s", err)
		return shim.Error(fmt.Sprintf("Error saving config: %s", err))
	}

	return shim.Success(nil)
}

// get retrieves configuration from the ledger
// args[0] - Is the JSON marshalled Criteria
func (scc *configSCC) get(stub shim.ChaincodeStubInterface, args [][]byte) pb.Response {
	if len(args) == 0 {
		return shim.Error("criteria not provided")
	}

	criteria, err := unmarshalCriteria(args[0])
	if err != nil {
		logger.Errorf("Error unmarshalling criteria: %s", err)
		return shim.Error(err.Error())
	}

	config, err := getConfigMgr(service.ConfigNS, state.NewShimStoreProvider(stub)).Query(criteria)
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
func (scc *configSCC) remove(stub shim.ChaincodeStubInterface, args [][]byte) pb.Response {
	if len(args) == 0 {
		return shim.Error("criteria not provided")
	}

	criteria, err := unmarshalCriteria(args[0])
	if err != nil {
		logger.Errorf("Error unmarshalling criteria: %s", err)
		return shim.Error(err.Error())
	}

	if err := getConfigMgr(service.ConfigNS, state.NewShimStoreProvider(stub)).Delete(criteria); err != nil {
		logger.Errorf("Error deleting config for criteria [%s]: %s", criteria, err)
		return shim.Error(fmt.Sprintf("Error deleting config: %s", err))
	}

	return shim.Success(nil)
}

func (scc *configSCC) initFunctionRegistry() {
	scc.functionRegistry = make(map[string]function)
	scc.functionRegistry["save"] = scc.put
	scc.functionRegistry["get"] = scc.get
	scc.functionRegistry["delete"] = scc.remove
}

// functionSet returns a string enumerating all available functions
func (scc *configSCC) functionSet() string {
	var functionNames string
	for name := range scc.functionRegistry {
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
var getConfigMgr = func(ns string, sp api.StoreProvider) configMgr {
	return ledgerconfig.NewUpdateManager(ns, sp)
}

// marshalJSON returns the JSON representation of the given value. This variable may be overridden by unit tests.
var marshalJSON = func(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// unmarshalJSON returns the unmarshalled value for the given JSON. This variable may be overridden by unit tests.
var unmarshalJSON = func(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
