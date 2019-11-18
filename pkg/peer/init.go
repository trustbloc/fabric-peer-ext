/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package peer

import (
	"github.com/trustbloc/fabric-peer-ext/cmd/chaincode/configscc"
	"github.com/trustbloc/fabric-peer-ext/pkg/chaincode/scc"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/client"
	dcasclient "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/dcas/client"
	extretriever "github.com/trustbloc/fabric-peer-ext/pkg/collections/retriever"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/storeprovider"
	tretriever "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/retriever"
	tdatastore "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/storeprovider"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/support"
	"github.com/trustbloc/fabric-peer-ext/pkg/resource"
)

// Initialize initializes the peer
func Initialize() {
	registerResources()
	registerSystemChaincodes()
}

func registerResources() {
	resource.Register(support.NewCollectionConfigRetrieverProvider)
	resource.Register(tdatastore.New)
	resource.Register(storeprovider.NewOffLedgerProvider)
	resource.Register(tretriever.NewProvider)
	resource.Register(extretriever.NewOffLedgerProvider)
	resource.Register(client.NewProvider)
	resource.Register(dcasclient.NewProvider)
}

func registerSystemChaincodes() {
	scc.Register(configscc.New)
}
