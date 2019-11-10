/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"strings"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/peer/node"
	"github.com/spf13/viper"
	extscc "github.com/trustbloc/fabric-peer-ext/pkg/chaincode/scc"
	extpeer "github.com/trustbloc/fabric-peer-ext/pkg/peer"
	"github.com/trustbloc/fabric-peer-ext/test/scc/testscc"
)

var logger = flogging.MustGetLogger("peer-ext-test")

func main() {
	setup()

	extpeer.Initialize()

	logger.Infof("Registering testscc...")
	extscc.Register(testscc.New)

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
