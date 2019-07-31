/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package support

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/common/privdata"
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	"github.com/hyperledger/fabric/protos/common"
)

var logger = flogging.MustGetLogger("ext_support")

type blockPublisherProvider func(channelID string) gossipapi.BlockPublisher

// Support holds the ledger provider and the cache
type Support struct {
	blockPublisherProvider blockPublisherProvider
}

// New creates a new Support using the ledger provider
func New(blockPublisherProvider blockPublisherProvider) *Support {
	return &Support{
		blockPublisherProvider: blockPublisherProvider,
	}
}

// Config returns the configuration for the given collection
func (s *Support) Config(channelID, ns, coll string) (*common.StaticCollectionConfig, error) {
	return CollectionConfigRetrieverForChannel(channelID).Config(ns, coll)
}

// Policy returns the collection access policy for the given collection
func (s *Support) Policy(channelID, ns, coll string) (privdata.CollectionAccessPolicy, error) {
	return CollectionConfigRetrieverForChannel(channelID).Policy(ns, coll)
}

// BlockPublisher returns the block publisher for the given channel
func (s *Support) BlockPublisher(channelID string) gossipapi.BlockPublisher {
	return s.blockPublisherProvider(channelID)
}
