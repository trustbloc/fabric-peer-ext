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
	// The following should be uncommented after all resources are converted to use dependency injection
	//resource.Register(tdatastore.New, resource.PriorityHigh)
	//resource.Register(storeprovider.NewOffLedgerProvider, resource.PriorityHigh)
	//resource.Register(tretriever.NewProvider, resource.PriorityHigh)
	//resource.Register(extretriever.NewOffLedgerProvider, resource.PriorityHigh)
	//resource.Register(client.NewProvider, resource.PriorityNormal)
	//resource.Register(dcasclient.NewProvider, resource.PriorityNormal)
}

func registerSystemChaincodes() {
	scc.Register(configscc.New)
}
