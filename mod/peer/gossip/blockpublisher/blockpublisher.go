/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockpublisher

import (
	"github.com/hyperledger/fabric/extensions/gossip/api"
	extblockpublisher "github.com/trustbloc/fabric-peer-ext/pkg/gossip/blockpublisher"
)

// ProviderInstance manages a block publisher for each channel
var ProviderInstance = extblockpublisher.NewProvider()

// ForChannel returns the block publisher for the given channel
func ForChannel(channelID string) api.BlockPublisher {
	return ProviderInstance.ForChannel(channelID)
}
