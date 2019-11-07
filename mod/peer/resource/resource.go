/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package resource

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/trustbloc/fabric-peer-ext/pkg/resource"
)

var logger = flogging.MustGetLogger("ext_resource")

// Initialize invokes all registered resource creator functions using the given set of providers.
func Initialize(providers ...interface{}) error {
	logger.Info("Initializing resources")
	return resource.Mgr.Initialize(providers...)
}

// ChannelJoined is called when the peer joins a channel. All resources that define the
// function, ChannelJoined(string) are invoked.
func ChannelJoined(channelID string) {
	logger.Infof("Notifying resources that channel [%s] was joined", channelID)
	resource.Mgr.ChannelJoined(channelID)
}

// Close is called when the peer is shut down. All resources that define the
// function, Close() are invoked.
func Close() {
	logger.Infof("Notifying resources that peer is shutting down")
	resource.Mgr.Close()
}
