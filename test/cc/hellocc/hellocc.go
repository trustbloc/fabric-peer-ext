/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hellocc

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	gproto "github.com/hyperledger/fabric-protos-go/gossip"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/common/flogging"
	ccapi "github.com/hyperledger/fabric/extensions/chaincode/api"
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"

	"github.com/trustbloc/fabric-peer-ext/pkg/common"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/discovery"
	"github.com/trustbloc/fabric-peer-ext/pkg/gossip/appdata"
)

var logger = flogging.MustGetLogger("hellocc")

const (
	v1 = "v1"

	sayHelloFunc = "sayhello"

	HelloDataType     = "Hello!"
	gossipMaxAttempts = 2
)

// AppDataHandlerRegistry is used to register a Gossip handler for application-specific data
type AppDataHandlerRegistry interface {
	Register(dataType string, handler appdata.Handler) error
}

// GossipProvider provides a Gossip service
type GossipProvider interface {
	GetGossipService() gossipapi.GossipService
}

// HelloCC is a sample chaincode that broadcasts a Hello message to other peers on a channel
// and the other peers respond with a hello
type HelloCC struct {
	GossipProvider
	name string
}

// New returns a new Hello chaincode instance
func New(name string, handlerRegistry AppDataHandlerRegistry, gossipProvider GossipProvider) *HelloCC {
	logger.Infof("Creating chaincode [%s]", name)

	cc := &HelloCC{
		GossipProvider: gossipProvider,
		name:           name,
	}

	if err := handlerRegistry.Register(HelloDataType, cc.handleRequest); err != nil {
		panic(err)
	}

	return cc
}

// Name returns the name of this chaincode
func (cc *HelloCC) Name() string { return cc.name }

// Version returns the version of the chaincode
func (cc *HelloCC) Version() string { return v1 }

// Chaincode returns the chaincode
func (cc *HelloCC) Chaincode() shim.Chaincode { return cc }

// GetDBArtifacts returns Couch DB indexes (if applicable)
func (cc *HelloCC) GetDBArtifacts(collNames []string) map[string]*ccapi.DBArtifacts {
	return nil
}

// Init is not used
func (cc *HelloCC) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

// Invoke invokes the chaincode with a given function
func (cc *HelloCC) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	if function == "" {
		return shim.Error("Expecting function")
	}

	if function == sayHelloFunc {
		return cc.sayHello(stub, args)
	}

	return shim.Error(fmt.Sprintf("Unknown function [%s]. Expecting '%s'", function, sayHelloFunc))
}

// sayHello broadcasts the given 'hello' message over Gossip to other peers and returns their responses.
func (h *HelloCC) sayHello(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		return shim.Error("Invalid args. Expecting hello message and number of peers")
	}

	msg := args[0]
	maxPeers := args[1]

	gossipMaxPeers, err := strconv.Atoi(maxPeers)
	if err != nil {
		return shim.Error(err.Error())
	}

	logger.Infof("Saying hello to %d peers with message '%s'", maxPeers, msg)

	d := discovery.New(stub.GetChannelID(), h.GetGossipService())

	myName := d.Self().Endpoint
	peersToGreet := d.GetMembers(func(m *discovery.Member) bool { return !m.Local })

	peerEndpoints := make([]string, len(peersToGreet))
	for i, p := range peersToGreet {
		peerEndpoints[i] = p.Endpoint
	}

	helloRequest := &helloRequest{
		Name:    myName,
		Peers:   peerEndpoints,
		Message: msg,
	}

	reqBytes, err := json.Marshal(helloRequest)
	if err != nil {
		panic(err)
	}

	retriever := appdata.NewRetriever(stub.GetChannelID(), h.GetGossipService(), gossipMaxAttempts, gossipMaxPeers)

	ctxt, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	responses, err := retriever.Retrieve(ctxt,
		&appdata.Request{
			DataType: HelloDataType,
			Payload:  reqBytes,
		},
		h.responseHandler(peerEndpoints),
		func(values common.Values) bool {
			return len(values) == len(peersToGreet) && values.AllSet()
		},
	)
	if err != nil {
		logger.Errorf("Got error requesting data from remote peersToGreet: %s", err)

		return shim.Error(err.Error())
	}

	logger.Infof("Got responses: %+v", responses)

	responsesBytes, err := json.Marshal(responses)
	if err != nil {
		return shim.Error(err.Error())
	}

	logger.Infof("Returning: %s", responsesBytes)

	return shim.Success(responsesBytes)
}

// handleRequest is a server-side handler for the 'Hello!' Gossip request. This function
// responds to the peer with a hello response.
func (h *HelloCC) handleRequest(channelID string, req *gproto.AppDataRequest) ([]byte, error) {
	logger.Infof("[%s] Got hello request: %s", channelID, req.Request)

	var helloRequest helloRequest
	err := json.Unmarshal(req.Request, &helloRequest)
	if err != nil {
		return nil, err
	}

	response := helloResponse{
		Name:    discovery.New(channelID, h.GetGossipService()).Self().Endpoint,
		Message: fmt.Sprintf("Hello %s! I received your message '%s'.", helloRequest.Name, helloRequest.Message),
	}

	respBytes, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	logger.Infof("[%s] Returning hello response: %s", channelID, respBytes)

	return respBytes, nil
}

// responseHandler is a client-side handler of a response from a peer. It ensures that
// the response from the peer is added to Values in the same order as the given peer endpoints.
func (h *HelloCC) responseHandler(peerEndpoints []string) appdata.ResponseHandler {
	return func(response []byte) (common.Values, error) {
		logger.Infof("Handling response: %s", response)

		helloResponse := &helloResponse{}
		if err := json.Unmarshal(response, helloResponse); err != nil {
			return nil, err
		}

		values := make(common.Values, len(peerEndpoints))
		for i, peerID := range peerEndpoints {
			if helloResponse.Name == peerID {
				values[i] = helloResponse
			}
		}

		logger.Infof("Returning values: %s", values)

		return values, nil
	}
}

type helloRequest struct {
	Name    string
	Message string
	Peers   []string
}

type helloResponse struct {
	Name    string
	Message string
}
