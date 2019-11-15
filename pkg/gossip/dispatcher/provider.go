/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dispatcher

import (
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	collcommon "github.com/trustbloc/fabric-peer-ext/pkg/collections/common"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/discovery"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/requestmgr"
)

// Provider is a gossip dispatcher provider
type Provider struct {
	gossip     gossipAdapter
	ccProvider collcommon.CollectionConfigProvider
}

// NewProvider returns a new gossip message dispatcher provider
func NewProvider() *Provider {
	return &Provider{}
}

// Initialize is called at startup by the resource manager
func (p *Provider) Initialize(gossip gossipAdapter, ccProvider collcommon.CollectionConfigProvider) *Provider {
	logger.Infof("Initializing gossip dispatcher provider")
	p.gossip = gossip
	p.ccProvider = ccProvider
	return p
}

// ForChannel returns a new dispatcher for the given channel
func (p *Provider) ForChannel(channelID string, dataStore storeapi.Store) *Dispatcher {
	logger.Infof("Returning a new gossip dispatcher for channel [%s]", channelID)
	return &Dispatcher{
		ccProvider: p.ccProvider,
		channelID:  channelID,
		reqMgr:     requestmgr.Get(channelID),
		dataStore:  dataStore,
		discovery:  discovery.New(channelID, p.gossip),
	}
}
