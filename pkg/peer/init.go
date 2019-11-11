/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package peer

import (
	"github.com/trustbloc/fabric-peer-ext/cmd/chaincode/configscc"
	"github.com/trustbloc/fabric-peer-ext/pkg/chaincode/scc"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/support"
	"github.com/trustbloc/fabric-peer-ext/pkg/resource"
)

// Initialize initializes the peer
func Initialize() {
	registerResources()
	registerSystemChaincodes()
}

func registerResources() {
	resource.Register(support.NewCollectionConfigRetrieverProvider, resource.PriorityHighest)
}

func registerSystemChaincodes() {
	scc.Register(configscc.New)
}
