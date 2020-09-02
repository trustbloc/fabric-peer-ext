/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package storeprovider

import (
	"sync"

	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/common/flogging"
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	collcommon "github.com/trustbloc/fabric-peer-ext/pkg/collections/common"
	olapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/dcas"
	olstoreprovider "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/storeprovider"
	tdapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/api"
)

var logger = flogging.MustGetLogger("ext_store")

// New returns a new store provider factory
func New() *StoreProvider {
	return &StoreProvider{
		stores: make(map[string]*store),
	}
}

// StoreProvider is a store provider that creates delegating stores.
// A delegating store delegates requests to collection-specific store.
// For example, transient data store, Off-ledger store, etc.
type StoreProvider struct {
	transientDataProvider tdapi.StoreProvider
	olProvider            olapi.StoreProvider
	stores                map[string]*store
	sync.RWMutex
}

// Initialize is called at startup by the resource manager
func (sp *StoreProvider) Initialize(tDataProvider tdapi.StoreProvider, olProvider olapi.StoreProvider) *StoreProvider {
	logger.Infof("Initializing collection store provider")
	sp.transientDataProvider = tDataProvider
	sp.olProvider = olProvider
	return sp
}

// StoreForChannel returns the store for the given channel
func (sp *StoreProvider) StoreForChannel(channelID string) storeapi.Store {
	sp.RLock()
	defer sp.RUnlock()
	return sp.stores[channelID]
}

// OpenStore opens the store for the given channel
func (sp *StoreProvider) OpenStore(channelID string) (storeapi.Store, error) {
	sp.Lock()
	defer sp.Unlock()

	store, ok := sp.stores[channelID]
	if !ok {
		tdataStore, err := sp.transientDataProvider.OpenStore(channelID)
		if err != nil {
			return nil, err
		}
		olStore, err := sp.olProvider.OpenStore(channelID)
		if err != nil {
			return nil, err
		}
		store = newDelegatingStore(channelID,
			targetStores{
				transientDataStore: tdataStore,
				offLedgerStore:     olStore,
			},
		)
		sp.stores[channelID] = store
	}
	return store, nil
}

// Close shuts down all of the stores
func (sp *StoreProvider) Close() {
	for _, s := range sp.stores {
		s.Close()
	}
}

// NewOffLedgerProvider creates a new off-ledger store provider that supports DCAS
func NewOffLedgerProvider(identifierProvider collcommon.IdentifierProvider, idDProvider collcommon.IdentityDeserializerProvider, collConfigProvider collcommon.CollectionConfigProvider) olapi.StoreProvider {
	logger.Infof("Creating off-ledger store provider with DCAS")

	return olstoreprovider.New(
		identifierProvider, idDProvider, collConfigProvider,
		olstoreprovider.WithCollectionType(
			pb.CollectionType_COL_DCAS,
			olstoreprovider.WithDecorator(dcas.Decorator),
			olstoreprovider.WithCacheEnabled(),
		),
	)
}
