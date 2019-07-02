/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package storeprovider

import (
	"sync"

	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	cb "github.com/hyperledger/fabric/protos/common"
	olapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/dcas"
	olstoreprovider "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/storeprovider"
	tdapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/storeprovider"
)

// New returns a new store provider factory
func New() *StoreProvider {
	return &StoreProvider{
		transientDataProvider: newTransientDataProvider(),
		olProvider:            newOffLedgerProvider(),
		stores:                make(map[string]*store),
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

// newTransientDataProvider may be overridden in unit tests
var newTransientDataProvider = func() tdapi.StoreProvider {
	return storeprovider.New()
}

// newOffLedgerProvider may be overridden in unit tests
var newOffLedgerProvider = func() olapi.StoreProvider {
	return olstoreprovider.New(
		olstoreprovider.WithCollectionType(
			cb.CollectionType_COL_DCAS,
			olstoreprovider.WithDecorator(dcas.Decorator),
		),
	)
}
