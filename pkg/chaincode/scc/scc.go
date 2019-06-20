/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package scc

import (
	"github.com/hyperledger/fabric/core/scc"
	"github.com/trustbloc/fabric-peer-ext/pkg/chaincode/scc/builder"
)

var sccBuilder = builder.New()

type creator interface{}

// Register registers a System Chaincode creator function. The system chaincode
// will be initialized during peer startup with all of its declared dependencies.
func Register(c creator) {
	sccBuilder.Add(c)
}

// Create returns a list of system chain codes, initialized with the given providers.
func Create(providers ...interface{}) []scc.SelfDescribingSysCC {
	descs, err := sccBuilder.Build(providers...)
	if err != nil {
		panic(err.Error())
	}
	return descs
}
