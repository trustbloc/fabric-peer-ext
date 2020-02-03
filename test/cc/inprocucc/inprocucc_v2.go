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
	v2 = "v2.0"
)

// V2 chaincode for in-process UCC tests
type V2 struct {
	*V1_1
}

// New returns a new instance of V2
func NewV2() *V2 {
	return &V2{V1_1: NewV1_1()}
}

// Version returns the version of this chaincode
func (cc *V2) Version() string { return v2 }

// Chaincode returns the chaincode implementation
func (cc *V2) Chaincode() shim.Chaincode { return cc }

// Invoke invokes the chaincode
func (cc *V2) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, _ := stub.GetFunctionAndParameters()
	if function == funcGetVersion {
		return shim.Success([]byte(cc.Version()))
	}

	return cc.V1_1.Invoke(stub)
}
