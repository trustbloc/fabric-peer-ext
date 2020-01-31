/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package inprocucc

import (
	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
)

const (
	v1_1 = "v1.1"
)

// V1_1 chaincode for in-process UCC tests
type V1_1 struct {
	*V1
}

// New returns a new instance of V1_1
func NewV1_1() *V1_1 {
	return &V1_1{V1: NewV1()}
}

// Version returns the version of this chaincode
func (cc *V1_1) Version() string { return v1_1 }

// Chaincode returns the chaincode implementation
func (cc *V1_1) Chaincode() shim.Chaincode { return cc }

// Invoke invokes the chaincode
func (cc *V1_1) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, _ := stub.GetFunctionAndParameters()
	if function == funcGetVersion {
		return shim.Success([]byte(cc.Version()))
	}

	return cc.V1.Invoke(stub)
}
