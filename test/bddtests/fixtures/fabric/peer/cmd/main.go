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
	"github.com/trustbloc/fabric-peer-ext/cmd/chaincode/configscc"
	extscc "github.com/trustbloc/fabric-peer-ext/pkg/chaincode/scc"
)

var logger = flogging.MustGetLogger("peer-ext-test")

func main() {
	setup()

	registerSystemChaincodes()

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

func registerSystemChaincodes() {
	logger.Infof("Registering configscc...")
	extscc.Register(configscc.New)
}

func startPeer() error {
	return node.Start()
}
