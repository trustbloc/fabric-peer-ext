/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	xgossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
)

// BlockPublisherProvider returns a block publisher for a given channel
type BlockPublisherProvider interface {
	ForChannel(channelID string) xgossipapi.BlockPublisher
}
