/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package statedb

import (
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	"github.com/hyperledger/fabric/extensions/gossip/blockpublisher"
	"github.com/hyperledger/fabric/extensions/roles"
)

//AddCCUpgradeHandler adds chaincode upgrade handler to blockpublisher
func AddCCUpgradeHandler(chainName string, handler gossipapi.ChaincodeUpgradeHandler) {
	if !roles.IsCommitter() {
		blockpublisher.GetProvider().ForChannel(chainName).AddCCUpgradeHandler(handler)
	}
}
