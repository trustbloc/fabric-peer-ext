/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/core/common/ccprovider"
)

// ChaincodeDataBuilder builds mock ChaincodeData
type ChaincodeDataBuilder struct {
	name    string
	version string
	vscc    string
	policy  string
}

// NewChaincodeDataBuilder returns a mock ChaincodeData builder
func NewChaincodeDataBuilder() *ChaincodeDataBuilder {
	return &ChaincodeDataBuilder{}
}

// Name sets the chaincode name
func (b *ChaincodeDataBuilder) Name(value string) *ChaincodeDataBuilder {
	b.name = value
	return b
}

// Version sets the chaincode version
func (b *ChaincodeDataBuilder) Version(value string) *ChaincodeDataBuilder {
	b.version = value
	return b
}

// VSCC sets the VSCC
func (b *ChaincodeDataBuilder) VSCC(value string) *ChaincodeDataBuilder {
	b.vscc = value
	return b
}

// Policy sets the chaincode policy
func (b *ChaincodeDataBuilder) Policy(value string) *ChaincodeDataBuilder {
	b.policy = value
	return b
}

// Build returns the ChaincodeData
func (b *ChaincodeDataBuilder) Build() *ccprovider.ChaincodeData {
	cd := &ccprovider.ChaincodeData{
		Name:    b.name,
		Version: b.version,
		Vscc:    b.vscc,
	}

	if b.policy != "" {
		policyEnv, err := cauthdsl.FromString(b.policy)
		if err != nil {
			panic(err.Error())
		}
		policyBytes, err := proto.Marshal(policyEnv)
		if err != nil {
			panic(err.Error())
		}
		cd.Policy = policyBytes
	}
	return cd
}

// BuildBytes returns the ChaincodeData bytes
func (b *ChaincodeDataBuilder) BuildBytes() []byte {
	bytes, err := proto.Marshal(b.Build())
	if err != nil {
		panic(err.Error())
	}
	return bytes
}
