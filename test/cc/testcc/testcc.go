/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package testcc

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/common/flogging"
	ccapi "github.com/hyperledger/fabric/extensions/chaincode/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
	configsvc "github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/service"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/api"
)

var logger = flogging.MustGetLogger("testcc")

const (
	ccName       = "testcc"
	ccVersion    = "v1"
	generalMSPID = "general"
)

type function func(shim.ChaincodeStubInterface, [][]byte) pb.Response

type peerConfig interface {
	PeerID() string
	MSPID() string
}

type configServiceProvider interface {
	ForChannel(channelID string) config.Service
}

type txnServiceProvider interface {
	ForChannel(channelID string) (api.Service, error)
}

type TestCC struct {
	functionRegistry map[string]function
	localMSPID       string
	localPeerID      string
	mutex            sync.RWMutex
	configProvider   configServiceProvider
	txnProvider      txnServiceProvider
	config           map[config.Key]*config.Value
}

// New returns a new test chaincode
func New(configServiceProvider configServiceProvider, peerConfig peerConfig, txnProvider txnServiceProvider) *TestCC {
	cc := &TestCC{
		config:         make(map[config.Key]*config.Value),
		localMSPID:     peerConfig.MSPID(),
		localPeerID:    peerConfig.PeerID(),
		configProvider: configServiceProvider,
		txnProvider:    txnProvider,
	}
	cc.initFunctionRegistry()
	return cc
}

// Name returns the name of this chaincode
func (cc *TestCC) Name() string { return ccName }

// Version returns the version of this chaincode
func (cc *TestCC) Version() string { return ccVersion }

// Chaincode returns the chaincode implementation
func (cc *TestCC) Chaincode() shim.Chaincode { return cc }

// GetDBArtifacts returns DB artifacts. For this chaincode there are no artifacts.
func (cc *TestCC) GetDBArtifacts() map[string]*ccapi.DBArtifacts { return nil }

// Init will be deprecated in a future Fabric release
func (cc *TestCC) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

// ChannelJoined is called when the peer joins a channel
func (cc *TestCC) ChannelJoined(channelID string) {
	logger.Infof("[%s] Registering for config update events for local MSP [%s] and local peer [%s] and app [%s]", channelID, cc.localMSPID, cc.localPeerID, ccName)
	cc.configProvider.ForChannel(channelID).AddUpdateHandler(func(kv *config.KeyValue) {
		if kv.MspID == cc.localMSPID && kv.PeerID == cc.localPeerID {
			cc.updateConfig(kv.Key, kv.Value)
		}
	})
}

// Invoke invokes the config SCC
func (cc *TestCC) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
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

	logger.Debugf("Invoking function [%s] with args: %s", functionName, functionArgs)
	return f(stub, functionArgs)
}

// getConfig returns the config for the given key from the local cache if it's targeted for the local peer.
// If a request is made for data owned by another MSP and/or peer then an empty value is returned.
// If a request is made for the "general" MSP then the config is retrieved from the config service.
func (cc *TestCC) getConfig(stub shim.ChaincodeStubInterface, args [][]byte) pb.Response {
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
		value, err := cc.configProvider.ForChannel(stub.GetChannelID()).Get(key)
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

	if key.MspID != cc.localMSPID && key.PeerID != cc.localPeerID {
		return shim.Error(fmt.Sprintf("this peer [%s] does not have access to config for key [%s]", cc.localPeerID, key))
	}

	logger.Infof("Retrieving value for key [%s] from the local cache...", key)
	value := cc.getComponentConfig(key)
	if value != nil {
		logger.Infof("... got value for key [%s] from the local cache: %s", key, value)
		return shim.Success([]byte(value.Config))
	}
	logger.Infof("... value for key [%s] not found in the local cache", key)
	return shim.Success(nil)
}

func (cc *TestCC) queryConfig(stub shim.ChaincodeStubInterface, args [][]byte) pb.Response {
	if len(args) < 1 {
		return shim.Error("expecting config criteria")
	}

	criteria := &config.Criteria{}
	err := json.Unmarshal(args[0], criteria)
	if err != nil {
		return shim.Error(fmt.Sprintf("invalid criteria [%s]: %s", args[0], err))
	}

	logger.Infof("Retrieving value for criteria [%s] from the config service...", criteria)
	results, err := cc.configProvider.ForChannel(stub.GetChannelID()).Query(criteria)
	if err != nil {
		return shim.Error(fmt.Sprintf("error getting value for criteria [%s]: %s", criteria, err))
	}

	logger.Infof("... got results for criteria [%s] from the config service: %+v", criteria, results)

	resultBytes, err := json.Marshal(results)
	if err != nil {
		return shim.Error(fmt.Sprintf("error marshalling results: %s", err))
	}

	return shim.Success(resultBytes)
}

func (cc *TestCC) endorse(stub shim.ChaincodeStubInterface, args [][]byte) pb.Response {
	if len(args) < 1 {
		return shim.Error("expecting chaincode name and optional args")
	}

	req := &api.Request{
		ChaincodeID: string(args[0]),
		Args:        args[1:],
	}

	channelID := stub.GetChannelID()

	txnSvc, err := cc.txnProvider.ForChannel(channelID)
	if err != nil {
		logger.Errorf("Error getting transaction service for channel [%s]: %s", channelID, err)
		return shim.Error(fmt.Sprintf("Error getting transaction service for channel [%s]: %s", channelID, err))
	}

	logger.Infof("Executing Endorse on channel [%s]...", channelID)

	resp, err := txnSvc.Endorse(req)
	if err != nil {
		logger.Errorf("Error returned from Endorse: %s", err)
		return shim.Error(fmt.Sprintf("Error returned from Endorse: %s", err))
	}

	logger.Infof("... Endorse succeeded on channel [%s]", channelID)

	return shim.Success(resp.Payload)
}

func (cc *TestCC) endorseAndCommit(stub shim.ChaincodeStubInterface, args [][]byte) pb.Response {
	if len(args) < 1 {
		return shim.Error("expecting chaincode name and optional args")
	}

	req := &api.Request{
		ChaincodeID: string(args[0]),
		Args:        args[1:],
	}

	// Perform the transaction in the background so as not to cause a deadlock in the current invocation
	go func(channelID string, req *api.Request) {
		txnSvc, err := cc.txnProvider.ForChannel(channelID)
		if err != nil {
			logger.Errorf("Error getting transaction service for channel [%s]: %s", channelID, err)
			return
		}

		logger.Infof("Executing EndorseAndCommit on channel [%s]...", channelID)

		if _, err := txnSvc.EndorseAndCommit(req); err != nil {
			logger.Errorf("Error returned from EndorseAndCommit: %s", err)
			return
		}

		logger.Infof("... EndorseAndCommit succeeded on channel [%s]", channelID)
	}(stub.GetChannelID(), req)

	return shim.Success(nil)
}

func (cc *TestCC) updateConfig(key *config.Key, value *config.Value) {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	if value == nil {
		delete(cc.config, *key)
		logger.Infof("Key [%s] was deleted from local cache", key)
	} else {
		cc.config[*key] = value
		logger.Infof("Value for key [%s] in local cache was updated to: %s", key, value)
	}
}

func (cc *TestCC) getComponentConfig(key *config.Key) *config.Value {
	cc.mutex.RLock()
	defer cc.mutex.RUnlock()
	return cc.config[*key]
}

func (cc *TestCC) initFunctionRegistry() {
	cc.functionRegistry = make(map[string]function)
	cc.functionRegistry["getconfig"] = cc.getConfig
	cc.functionRegistry["queryconfig"] = cc.queryConfig
	cc.functionRegistry["endorse"] = cc.endorse
	cc.functionRegistry["endorseandcommit"] = cc.endorseAndCommit
}

// functionSet returns a string enumerating all available functions
func (cc *TestCC) functionSet() string {
	var functionNames string
	for name := range cc.functionRegistry {
		if functionNames != "" {
			functionNames += ", "
		}
		functionNames += name
	}
	return functionNames
}
