/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package inprocucc

import (
	"fmt"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	ccapi "github.com/hyperledger/fabric/extensions/chaincode/api"
)

const (
	inProcCCName = "inproc_test_cc"
	v1           = "v1"

	funcGetVersion = "getversion"
)

// V1 chaincode for in-process UCC tests
type V1 struct {
}

// New returns a new instance of V1
func NewV1() *V1 {
	return &V1{}
}

// Name returns the name of this chaincode
func (cc *V1) Name() string { return inProcCCName }

// Version returns the version of this chaincode
func (cc *V1) Version() string { return v1 }

// Chaincode returns the chaincode implementation
func (cc *V1) Chaincode() shim.Chaincode { return cc }

// GetDBArtifacts returns DB artifacts. For this chaincode there are no artifacts.
func (cc *V1) GetDBArtifacts([]string) map[string]*ccapi.DBArtifacts { return nil }

// Init will be deprecated in a future Fabric release
func (cc *V1) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

// Invoke invokes the chaincode
func (cc *V1) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, _ := stub.GetFunctionAndParameters()
	if function == funcGetVersion {
		return shim.Success([]byte(cc.Version()))
	}

	return shim.Error(fmt.Sprintf("unknown function: [%s]", function))
}
