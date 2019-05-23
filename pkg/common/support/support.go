/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package support

import (
	"github.com/bluele/gcache"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/core/ledger"
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	"github.com/hyperledger/fabric/protos/common"
)

var logger = flogging.MustGetLogger("ext_support")

type ledgerProvider func(channelID string) ledger.PeerLedger
type blockPublisherProvider func(channelID string) gossipapi.BlockPublisher

// Support holds the ledger provider and the cache
type Support struct {
	getLedger              ledgerProvider
	configRetrieverCache   gcache.Cache
	blockPublisherProvider blockPublisherProvider
}

// New creates a new Support using the ledger provider
func New(ledgerProvider ledgerProvider, blockPublisherProvider blockPublisherProvider) *Support {
	s := &Support{
		getLedger:              ledgerProvider,
		blockPublisherProvider: blockPublisherProvider,
	}
	s.configRetrieverCache = gcache.New(0).Simple().LoaderFunc(
		func(key interface{}) (interface{}, error) {
			channelID := key.(string)
			logger.Debugf("[%s] Creating collection config retriever", channelID)
			return NewCollectionConfigRetriever(channelID, s.getLedger(channelID), blockPublisherProvider(channelID)), nil
		}).Build()
	return s
}

// Config returns the configuration for the given collection
func (s *Support) Config(channelID, ns, coll string) (*common.StaticCollectionConfig, error) {
	ccRetriever, err := s.configRetrieverCache.Get(channelID)
	if err != nil {
		return nil, err
	}
	return ccRetriever.(*CollectionConfigRetriever).Config(ns, coll)
}

// Policy returns the collection access policy for the given collection
func (s *Support) Policy(channelID, ns, coll string) (privdata.CollectionAccessPolicy, error) {
	ccRetriever, err := s.configRetrieverCache.Get(channelID)
	if err != nil {
		return nil, err
	}
	return ccRetriever.(*CollectionConfigRetriever).Policy(ns, coll)
}

// BlockPublisher returns the block publisher for the given channel
func (s *Support) BlockPublisher(channelID string) gossipapi.BlockPublisher {
	return s.blockPublisherProvider(channelID)
}
