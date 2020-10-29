/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"strings"

	"github.com/hyperledger/fabric/common/flogging"
	ccapi "github.com/hyperledger/fabric/extensions/chaincode/api"
	"github.com/hyperledger/fabric/peer/node"
	"github.com/spf13/cobra"
	viper "github.com/spf13/viper2015"

	"github.com/trustbloc/fabric-peer-ext/pkg/chaincode/scc"
	"github.com/trustbloc/fabric-peer-ext/pkg/chaincode/ucc"
	"github.com/trustbloc/fabric-peer-ext/pkg/handler/authfilter"
	extpeer "github.com/trustbloc/fabric-peer-ext/pkg/peer"
	"github.com/trustbloc/fabric-peer-ext/test/cc/examplecc"
	"github.com/trustbloc/fabric-peer-ext/test/cc/hellocc"
	"github.com/trustbloc/fabric-peer-ext/test/cc/inprocucc"
	"github.com/trustbloc/fabric-peer-ext/test/cc/testscc"
	"github.com/trustbloc/fabric-peer-ext/test/handler/exampleauthfilter"
)

var logger = flogging.MustGetLogger("peer-ext-test")

var (
	offLedgerDBArtifacts = map[string]*ccapi.DBArtifacts{
		"couchdb": {
			CollectionIndexes: map[string][]string{
				"accounts": {`{"index": {"fields": ["id"]}, "ddoc": "indexIDDoc", "name": "indexID", "type": "json"}`},
			},
		},
	}
)

func main() {
	setup()

	extpeer.Initialize()

	logger.Infof("Registering auth filters for BDD tests...")

	authfilter.Register(exampleauthfilter.New)

	logger.Infof("Registering in-process system chaincodes for BDD tests...")

	scc.Register(testscc.New)

	logger.Infof("Registering in-process user chaincodes for BDD tests...")

	ucc.Register(inprocucc.NewV1)
	ucc.Register(inprocucc.NewV1_1)
	ucc.Register(inprocucc.NewV2)

	ucc.Register(func(p examplecc.DCASStubWrapperFactory) ccapi.UserCC {
		return examplecc.New("ol_examplecc", offLedgerDBArtifacts, p)
	})
	ucc.Register(func(p examplecc.DCASStubWrapperFactory) ccapi.UserCC {
		return examplecc.New("ol_examplecc_2", nil, p)
	})
	ucc.Register(func(p examplecc.DCASStubWrapperFactory) ccapi.UserCC {
		return examplecc.New("tdata_examplecc", nil, p)
	})
	ucc.Register(func(p examplecc.DCASStubWrapperFactory) ccapi.UserCC {
		return examplecc.New("tdata_examplecc_2", nil, p)
	})

	ucc.Register(func(handlerRegistry hellocc.AppDataHandlerRegistry, gossipProvider hellocc.GossipProvider, stateDBProvider hellocc.StateDBProvider) ccapi.UserCC {
		return hellocc.New("hellocc", handlerRegistry, gossipProvider, stateDBProvider)
	})

	if err := startPeer(); err != nil {
		panic(err)
	}
}

func setup() {
	// For environment variables.
	viper.SetEnvPrefix(node.CmdRoot)
	viper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)

	node.InitCmd(&cobra.Command{}, nil)
}

func startPeer() error {
	return node.Start()
}
