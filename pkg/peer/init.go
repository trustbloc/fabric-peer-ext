/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package peer

import (
	"github.com/trustbloc/fabric-peer-ext/cmd/chaincode/configcc"
	"github.com/trustbloc/fabric-peer-ext/pkg/chaincode/ucc"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/client"
	dcasclient "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/dcas/client"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/dissemination"
	extretriever "github.com/trustbloc/fabric-peer-ext/pkg/collections/retriever"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/storeprovider"
	tretriever "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/retriever"
	tdatastore "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/storeprovider"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/dbname"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/support"
	cfgservice "github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/service"
	configvalidator "github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/validator"
	"github.com/trustbloc/fabric-peer-ext/pkg/gossip/appdata"
	gossipstate "github.com/trustbloc/fabric-peer-ext/pkg/gossip/state"
	"github.com/trustbloc/fabric-peer-ext/pkg/resource"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/proprespvalidator"
)

// Initialize initializes the peer
func Initialize() {
	registerResources()
	registerChaincodes()

	// The DB name resolver needs to be initialized explicitly since the
	// databases are created before the resources are initialized.
	dbname.ResolverInstance.Initialize(newConfig())
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
	resource.Register(configvalidator.NewRegistry)
	resource.Register(newConfig)
	resource.Register(txn.NewProvider)
	resource.Register(dissemination.LocalMSPProvider.Initialize)
	resource.Register(appdata.NewHandlerRegistry)
	resource.Register(proprespvalidator.New)
}

func registerChaincodes() {
	ucc.Register(configcc.New)
}
