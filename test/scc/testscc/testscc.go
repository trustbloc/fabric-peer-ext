/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package testscc

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/scc"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/spf13/viper"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
	configsvc "github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/service"
)

var logger = flogging.MustGetLogger("testscc")

const (
	sccName        = "testscc"
	sccPath        = "github.com/trustbloc/fabric-peer-ext/cmd/chaincodes/testscc"
	peerConfigName = "core"
	envPrefix      = "core"
	peerConfigPath = "/etc/hyperledger/fabric"
	generalMSPID   = "general"
)

type function func(shim.ChaincodeStubInterface, [][]byte) pb.Response

type configServiceProvider interface {
	ForChannel(channelID string) config.Service
}

type testSCC struct {
	functionRegistry map[string]function
	localMSPID       string
	localPeerID      string
	mutex            sync.RWMutex
	configProvider   configServiceProvider
	config           map[config.Key]*config.Value
}

// New returns a new test system chaincode
func New(configServiceProvider configServiceProvider) scc.SelfDescribingSysCC {
	peerConfig, err := newPeerViper()
	if err != nil {
		panic("Error reading peer config: " + err.Error())
	}

	cc := &testSCC{
		config:         make(map[config.Key]*config.Value),
		localMSPID:     peerConfig.GetString("peer.localMspId"),
		localPeerID:    peerConfig.GetString("peer.id"),
		configProvider: configServiceProvider,
	}
	cc.initFunctionRegistry()
	return cc
}

func (scc *testSCC) Name() string              { return sccName }
func (scc *testSCC) Path() string              { return sccPath }
func (scc *testSCC) InitArgs() [][]byte        { return nil }
func (scc *testSCC) Chaincode() shim.Chaincode { return scc }
func (scc *testSCC) InvokableExternal() bool   { return true }
func (scc *testSCC) InvokableCC2CC() bool      { return true }
func (scc *testSCC) Enabled() bool             { return true }

// Init initializes the config SCC
func (scc *testSCC) Init(stub shim.ChaincodeStubInterface) pb.Response {
	if stub.GetChannelID() != "" {
		logger.Infof("Registering for config update events for local MSP [%s] and local peer [%s] and app [%s]", scc.localMSPID, scc.localPeerID, sccName)
		scc.configProvider.ForChannel(stub.GetChannelID()).AddUpdateHandler(func(kv *config.KeyValue) {
			if kv.MspID == scc.localMSPID && kv.PeerID == scc.localPeerID {
				scc.updateConfig(kv.Key, kv.Value)
			}
		})
	}
	return shim.Success(nil)
}

// Invoke invokes the config SCC
func (scc *testSCC) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
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

// getConfig returns the config for the given key from the local cache if it's targeted for the local peer.
// If a request is made for data owned by another MSP and/or peer then an empty value is returned.
// If a request is made for the "general" MSP then the config is retrieved from the config service.
func (scc *testSCC) getConfig(stub shim.ChaincodeStubInterface, args [][]byte) pb.Response {
	if len(args) < 1 {
		return shim.Error("expecting config key")
	}

	key := &config.Key{}
	err := json.Unmarshal(args[0], key)
	if err != nil {
		return shim.Error(fmt.Sprintf("invalid key [%s]: %s", args[0], err))
	}

	if key.MspID == generalMSPID {
		logger.Infof("Retrieving value for key [%s] from the config service...", key)
		value, err := scc.configProvider.ForChannel(stub.GetChannelID()).Get(key)
		if err != nil {
			if err == configsvc.ErrConfigNotFound {
				logger.Infof("... value for key [%s] not found in the config service", key)
				return shim.Success(nil)
			}
			return shim.Error(fmt.Sprintf("error getting value for key [%s]: %s", key, err))
		}
		logger.Infof("... got value for key [%s] from the config service: %s", key, value)
		return shim.Success([]byte(value.Config))
	}

	if key.MspID != scc.localMSPID && key.PeerID != scc.localPeerID {
		return shim.Error(fmt.Sprintf("this peer [%s] does not have access to config for key [%s]", scc.localPeerID, key))
	}

	logger.Infof("Retrieving value for key [%s] from the local cache...", key)
	value := scc.getComponentConfig(key)
	if value != nil {
		logger.Infof("... got value for key [%s] from the local cache: %s", key, value)
		return shim.Success([]byte(value.Config))
	}
	logger.Infof("... value for key [%s] not found in the local cache", key)
	return shim.Success(nil)
}

func (scc *testSCC) updateConfig(key *config.Key, value *config.Value) {
	scc.mutex.Lock()
	defer scc.mutex.Unlock()

	if value == nil {
		delete(scc.config, *key)
		logger.Infof("Key [%s] was deleted from local cache", key)
	} else {
		scc.config[*key] = value
		logger.Infof("Value for key [%s] in local cache was updated to: %s", key, value)
	}
}

func (scc *testSCC) getComponentConfig(key *config.Key) *config.Value {
	scc.mutex.RLock()
	defer scc.mutex.RUnlock()
	return scc.config[*key]
}

func (scc *testSCC) initFunctionRegistry() {
	scc.functionRegistry = make(map[string]function)
	scc.functionRegistry["getconfig"] = scc.getConfig
}

// functionSet returns a string enumerating all available functions
func (scc *testSCC) functionSet() string {
	var functionNames string
	for name := range scc.functionRegistry {
		if functionNames != "" {
			functionNames += ", "
		}
		functionNames += name
	}
	return functionNames
}

func newPeerViper() (*viper.Viper, error) {
	peerViper := viper.New()
	peerViper.AddConfigPath(peerConfigPath)
	peerViper.SetConfigName(peerConfigName)
	peerViper.SetEnvPrefix(envPrefix)
	peerViper.AutomaticEnv()
	peerViper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := peerViper.ReadInConfig(); err != nil {
		return nil, err
	}
	return peerViper, nil
}
