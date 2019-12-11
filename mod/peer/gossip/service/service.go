/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"github.com/hyperledger/fabric-protos-go/gossip"
	"github.com/hyperledger/fabric/extensions/roles"
	"github.com/hyperledger/fabric/gossip/util"
)

var logger = util.GetLogger(util.ServiceLogger, "")

//HandleGossip can be used to extend GossipServiceAdapter.Gossip feature
func HandleGossip(handle func(msg *gossip.GossipMessage)) func(msg *gossip.GossipMessage) {
	if roles.IsEndorser() {
		return handle
	}
	return func(msg *gossip.GossipMessage) {
		logger.Debugf("Gossip from service adaptor skipped for non-endorsers")
	}
}

//IsPvtDataReconcilerEnabled can be used to override private data reconciller enable/disable
func IsPvtDataReconcilerEnabled(isEnabled bool) bool {
	return roles.IsCommitter() && isEnabled
}
