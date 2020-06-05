/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package scc

import (
	"github.com/hyperledger/fabric/core/scc"

	"github.com/trustbloc/fabric-peer-ext/pkg/common/injectinvoker"
	"github.com/trustbloc/fabric-peer-ext/pkg/resource"
)

var sccBuilder = injectinvoker.NewBuilder()

type creator interface{}

// Register registers a System Chaincode creator function. The system chaincode
// will be initialized during peer startup with all of its declared dependencies.
func Register(c creator) {
	sccBuilder.Add(c)
}

// Create returns a list of system chain codes, initialized with the given providers.
func Create(providers ...interface{}) []scc.SelfDescribingSysCC {
	// Merge the given providers with all of the registered resources
	resources, err := sccBuilder.Build(append(providers, resource.Mgr.Resources()...)...)
	if err != nil {
		panic(err.Error())
	}

	descs := make([]scc.SelfDescribingSysCC, len(resources))

	for i, r := range resources {
		descs[i] = r.(scc.SelfDescribingSysCC)
	}

	return descs
}
