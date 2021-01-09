/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package peer

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/extensions/gossip/state"
	storagecouchdb "github.com/hyperledger/fabric/extensions/storage/couchdb"

	"github.com/trustbloc/fabric-peer-ext/cmd/chaincode/configcc"
	ccnotifier "github.com/trustbloc/fabric-peer-ext/pkg/chaincode/notifier"
	"github.com/trustbloc/fabric-peer-ext/pkg/chaincode/ucc"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/client"
	dcasclient "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/dcas/client"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/dissemination"
	extretriever "github.com/trustbloc/fabric-peer-ext/pkg/collections/retriever"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/storeprovider"
	tretriever "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/retriever"
	tdatastore "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/storeprovider"
	extcouchdb "github.com/trustbloc/fabric-peer-ext/pkg/common/couchdb"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/dbname"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/support"
	cfgservice "github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/service"
	configvalidator "github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/validator"
	"github.com/trustbloc/fabric-peer-ext/pkg/gossip/appdata"
	gossipstate "github.com/trustbloc/fabric-peer-ext/pkg/gossip/state"
	"github.com/trustbloc/fabric-peer-ext/pkg/resource"
	extstatedb "github.com/trustbloc/fabric-peer-ext/pkg/statedb"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/proprespvalidator"
	"github.com/trustbloc/fabric-peer-ext/pkg/validation/validationctx"
	"github.com/trustbloc/fabric-peer-ext/pkg/validation/validationhandler"
	"github.com/trustbloc/fabric-peer-ext/pkg/validation/validator"
)

var logger = flogging.MustGetLogger("ext_peer")

// Initialize initializes the peer
func Initialize() {
	logger.Infof("Initializing resources")

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
	resource.Register(extcouchdb.NewReadOnlyProvider)
	resource.Register(storagecouchdb.NewHandler)
	resource.Register(ccnotifier.New)
	resource.Register(extstatedb.GetProvider)
	resource.Register(newDCASConfig)
	resource.Register(validator.NewProvider)
	resource.Register(validationhandler.NewProvider)
	resource.Register(state.InitValidationMgr)
	resource.Register(validationctx.NewProvider)
}

func registerChaincodes() {
	ucc.Register(configcc.New)
}
