/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package peer

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/trustbloc/fabric-peer-ext/cmd/chaincode/configcc"
	"github.com/trustbloc/fabric-peer-ext/pkg/chaincode/ucc"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/client"
	dcasclient "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/dcas/client"
	extretriever "github.com/trustbloc/fabric-peer-ext/pkg/collections/retriever"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/storeprovider"
	tretriever "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/retriever"
	tdatastore "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/storeprovider"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/support"
	cfgservice "github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/service"
	gossipstate "github.com/trustbloc/fabric-peer-ext/pkg/gossip/state"
	"github.com/trustbloc/fabric-peer-ext/pkg/resource"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn"
)

var logger = flogging.MustGetLogger("ext_peer")

// Initialize initializes the peer
func Initialize() {
	logger.Info("Initializing peer")

	registerResources()
	registerChaincodes()
}

func registerResources() {
	resource.Register(support.NewCollectionConfigRetrieverProvider)
	resource.Register(tdatastore.New)
	resource.Register(storeprovider.NewOffLedgerProvider)
	resource.Register(tretriever.NewProvider)
	resource.Register(extretriever.NewOffLedgerProvider)
	resource.Register(client.NewProvider)
	resource.Register(dcasclient.NewProvider)
	resource.Register(gossipstate.NewUpdateHandler)
	resource.Register(cfgservice.NewSvcMgr)
	resource.Register(txn.NewProvider)
	resource.Register(newConfig)
}

func registerChaincodes() {
	ucc.Register(configcc.New)
}
