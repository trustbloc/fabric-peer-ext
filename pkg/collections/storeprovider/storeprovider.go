/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package storeprovider

import (
	"sync"

	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	"github.com/hyperledger/fabric/gossip/service"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/msp/mgmt"
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

	// Temporary code
	initTDProvider sync.Once
	initOLProvider sync.Once
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
		tdataStore, err := sp.getTransientDataProvider().OpenStore(channelID)
		if err != nil {
			return nil, err
		}
		olStore, err := sp.getOLProvider().OpenStore(channelID)
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
	return storeprovider.New(service.GetGossipService(), newMSPProvider())
}

// newOffLedgerProvider may be overridden in unit tests
var newOffLedgerProvider = func() olapi.StoreProvider {
	return olstoreprovider.New(
		newMSPProvider(), newMSPProvider(),
		olstoreprovider.WithCollectionType(
			cb.CollectionType_COL_DCAS,
			olstoreprovider.WithDecorator(dcas.Decorator),
		),
	)
}

// The code below is temporarily added to facilitate the migration to dependency injected resources.
// This code should be removed once all resources have been converted to use dependency injection.

// Temporarily add mspProvider.
type mspProvider struct {
	msp.MSP
}

func newMSPProvider() *mspProvider {
	return &mspProvider{
		MSP: mgmt.GetLocalMSP(),
	}
}

// GetIdentityDeserializer returns the identity deserializer for the given channel
func (m *mspProvider) GetIdentityDeserializer(channelID string) msp.IdentityDeserializer {
	return mgmt.GetIdentityDeserializer(channelID)
}

// Temporarily defer the retrieval of the transient data provider since the GossipService
// is not yet initialized when the store provider is initialized.
func (sp *StoreProvider) getTransientDataProvider() tdapi.StoreProvider {
	sp.initTDProvider.Do(func() {
		sp.transientDataProvider = newTransientDataProvider()
	})
	return sp.transientDataProvider
}

// Temporarily defer the retrieval of the off-ledger provider since the GossipService
// is not yet initialized when the store provider is initialized.
func (sp *StoreProvider) getOLProvider() olapi.StoreProvider {
	sp.initOLProvider.Do(func() {
		sp.olProvider = newOffLedgerProvider()
	})
	return sp.olProvider
}
