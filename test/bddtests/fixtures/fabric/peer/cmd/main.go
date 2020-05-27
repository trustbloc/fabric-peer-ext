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
	viper "github.com/spf13/viper2015"

	"github.com/trustbloc/fabric-peer-ext/pkg/chaincode/ucc"
	extpeer "github.com/trustbloc/fabric-peer-ext/pkg/peer"
	"github.com/trustbloc/fabric-peer-ext/test/cc/examplecc"
	"github.com/trustbloc/fabric-peer-ext/test/cc/inprocucc"
	"github.com/trustbloc/fabric-peer-ext/test/cc/testcc"
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

	logger.Infof("Registering in-process user chaincodes for BDD tests...")

	ucc.Register(testcc.New)

	ucc.Register(inprocucc.NewV1)
	ucc.Register(inprocucc.NewV1_1)
	ucc.Register(inprocucc.NewV2)

	ucc.Register(func() ccapi.UserCC { return examplecc.New("ol_examplecc", offLedgerDBArtifacts) })
	ucc.Register(func() ccapi.UserCC { return examplecc.New("ol_examplecc_2", nil) })
	ucc.Register(func() ccapi.UserCC { return examplecc.New("tdata_examplecc", nil) })
	ucc.Register(func() ccapi.UserCC { return examplecc.New("tdata_examplecc_2", nil) })

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

	node.InitCmd(nil, nil)
}

func startPeer() error {
	return node.Start()
}
