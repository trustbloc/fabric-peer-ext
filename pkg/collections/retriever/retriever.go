/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package retriever

import (
	"context"
	"sync"

	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/common/privdata"
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	collcommon "github.com/trustbloc/fabric-peer-ext/pkg/collections/common"
	olapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/api"
	olretriever "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/retriever"
	tdataapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/api"
)

var logger = flogging.MustGetLogger("ext_retriever")

// Provider is a transient data provider.
type Provider struct {
	transientDataProvider tdataapi.Provider
	offLedgerProvider     olapi.Provider
	retrievers            map[string]*retriever
	mutex                 sync.RWMutex
}

// NewProvider returns a new transient data Retriever provider
func NewProvider() *Provider {
	return &Provider{
		retrievers: make(map[string]*retriever),
	}
}

// Initialize is called at startup by the resource manager
func (p *Provider) Initialize(tdProvider tdataapi.Provider, olProvider olapi.Provider) *Provider {
	logger.Info("Initializing collection data retriever")
	p.transientDataProvider = tdProvider
	p.offLedgerProvider = olProvider
	p.retrievers = make(map[string]*retriever)
	return p
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

// Query executes the given rich query
func (r *retriever) Query(ctxt context.Context, key *storeapi.QueryKey) (storeapi.ResultsIterator, error) {
	return r.offLedgerRetriever.Query(ctxt, key)
}

// Support defines the supporting functions required by the transient data provider
type Support interface {
	Config(channelID, ns, coll string) (*pb.StaticCollectionConfig, error)
	Policy(channel, ns, collection string) (privdata.CollectionAccessPolicy, error)
	BlockPublisher(channelID string) gossipapi.BlockPublisher
}

// NewOffLedgerProvider returns a new off-ledger retriever provider that supports DCAS
func NewOffLedgerProvider(providers *collcommon.Providers) olapi.Provider {
	return olretriever.NewProvider(providers)
}
