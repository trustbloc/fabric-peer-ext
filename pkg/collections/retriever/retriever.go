/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package retriever

import (
	"context"
	"sync"

	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/core/ledger"
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	supportapi "github.com/hyperledger/fabric/extensions/collections/api/support"
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	cb "github.com/hyperledger/fabric/protos/common"
	olapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/dcas"
	olretriever "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/retriever"
	tdataapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/api"
	tretriever "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/retriever"
	supp "github.com/trustbloc/fabric-peer-ext/pkg/common/support"
)

// Provider is a transient data provider.
type Provider struct {
	transientDataProvider tdataapi.Provider
	offLedgerProvider     olapi.Provider
	retrievers            map[string]*retriever
	mutex                 sync.RWMutex
}

// NewProvider returns a new transient data Retriever provider
func NewProvider(
	storeProvider func(channelID string) storeapi.Store,
	ledgerProvider func(channelID string) ledger.PeerLedger,
	gossipProvider func() supportapi.GossipAdapter,
	blockPublisherProvider func(channelID string) gossipapi.BlockPublisher) storeapi.Provider {

	support := supp.New(ledgerProvider, blockPublisherProvider)

	tdataStoreProvider := func(channelID string) tdataapi.Store { return storeProvider(channelID) }
	offLedgerStoreProvider := func(channelID string) olapi.Store { return storeProvider(channelID) }

	return &Provider{
		transientDataProvider: getTransientDataProvider(tdataStoreProvider, support, gossipProvider),
		offLedgerProvider:     getOffLedgerProvider(offLedgerStoreProvider, support, gossipProvider),
		retrievers:            make(map[string]*retriever),
	}
}

// RetrieverForChannel returns the collection retriever for the given channel
func (p *Provider) RetrieverForChannel(channelID string) storeapi.Retriever {
	p.mutex.RLock()
	r, ok := p.retrievers[channelID]
	p.mutex.RUnlock()

	if ok {
		return r
	}

	return p.getOrCreateRetriever(channelID)
}

func (p *Provider) getOrCreateRetriever(channelID string) storeapi.Retriever {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	r, ok := p.retrievers[channelID]
	if !ok {
		r = &retriever{
			transientDataRetriever: p.transientDataProvider.RetrieverForChannel(channelID),
			offLedgerRetriever:     p.offLedgerProvider.RetrieverForChannel(channelID),
		}
		p.retrievers[channelID] = r
	}

	return r
}

type retriever struct {
	transientDataRetriever tdataapi.Retriever
	offLedgerRetriever     olapi.Retriever
}

// GetTransientData returns the transient data for the given key
func (r *retriever) GetTransientData(ctxt context.Context, key *storeapi.Key) (*storeapi.ExpiringValue, error) {
	return r.transientDataRetriever.GetTransientData(ctxt, key)
}

// GetTransientDataMultipleKeys gets the values for the multiple transient data items in a single call
func (r *retriever) GetTransientDataMultipleKeys(ctxt context.Context, key *storeapi.MultiKey) (storeapi.ExpiringValues, error) {
	return r.transientDataRetriever.GetTransientDataMultipleKeys(ctxt, key)
}

// GetData gets the value for the given data item
func (r *retriever) GetData(ctxt context.Context, key *storeapi.Key) (*storeapi.ExpiringValue, error) {
	return r.offLedgerRetriever.GetData(ctxt, key)
}

// GetDataMultipleKeys gets the values for the multiple data items in a single call
func (r *retriever) GetDataMultipleKeys(ctxt context.Context, key *storeapi.MultiKey) (storeapi.ExpiringValues, error) {
	return r.offLedgerRetriever.GetDataMultipleKeys(ctxt, key)
}

// Support defines the supporting functions required by the transient data provider
type Support interface {
	Config(channelID, ns, coll string) (*cb.StaticCollectionConfig, error)
	Policy(channel, ns, collection string) (privdata.CollectionAccessPolicy, error)
	BlockPublisher(channelID string) gossipapi.BlockPublisher
}

var getTransientDataProvider = func(storeProvider func(channelID string) tdataapi.Store, support Support, gossipProvider func() supportapi.GossipAdapter) tdataapi.Provider {
	return tretriever.NewProvider(storeProvider, support, gossipProvider)
}

var getOffLedgerProvider = func(storeProvider func(channelID string) olapi.Store, support Support, gossipProvider func() supportapi.GossipAdapter) olapi.Provider {
	return olretriever.NewProvider(storeProvider, support, gossipProvider,
		olretriever.WithValidator(cb.CollectionType_COL_DCAS, dcas.Validator),
	)
}
