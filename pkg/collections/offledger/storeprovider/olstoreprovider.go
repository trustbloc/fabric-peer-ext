/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package storeprovider

import (
	"sync"

	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	"github.com/hyperledger/fabric/protos/common"
	olapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/storeprovider/store/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/storeprovider/store/couchdbstore"
)

// Option is a store provider option
type Option func(p *StoreProvider)

// CollOption is a collection option
type CollOption func(c *collTypeConfig)

// WithCollectionType adds a collection type to the set of collection types supported by the off-ledger store
// along with any options
func WithCollectionType(collType common.CollectionType, opts ...CollOption) Option {
	return func(p *StoreProvider) {
		c := &collTypeConfig{}
		p.collConfigs[collType] = c

		for _, opt := range opts {
			opt(c)
		}
	}
}

// Decorator allows the key/value to be modified/validated before being persisted
type Decorator func(key *storeapi.Key, value *storeapi.ExpiringValue) (*storeapi.Key, *storeapi.ExpiringValue, error)

// WithDecorator sets a decorator for a collection type allowing the key/value to be validated/modified before being persisted
func WithDecorator(decorator Decorator) CollOption {
	return func(c *collTypeConfig) {
		c.decorator = decorator
	}
}

// KeyDecorator allows the key to be modified/validated
type KeyDecorator func(key *storeapi.Key) (*storeapi.Key, error)

// WithKeyDecorator sets a key decorator for a collection type allowing the key to be validated/modified
func WithKeyDecorator(decorator KeyDecorator) CollOption {
	return func(c *collTypeConfig) {
		c.keyDecorator = decorator
	}
}

type collTypeConfig struct {
	decorator    Decorator
	keyDecorator KeyDecorator
}

// New returns a store provider factory
func New(opts ...Option) *StoreProvider {
	p := &StoreProvider{
		stores:      make(map[string]olapi.Store),
		dbProvider:  getDBProvider(),
		collConfigs: make(map[common.CollectionType]*collTypeConfig),
	}

	// OFF_LEDGER collection type supported by default
	opts = append(opts, WithCollectionType(common.CollectionType_COL_OFFLEDGER))

	// Apply options
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// StoreProvider is a store provider
type StoreProvider struct {
	stores map[string]olapi.Store
	sync.RWMutex
	dbProvider  api.DBProvider
	collConfigs map[common.CollectionType]*collTypeConfig
}

// StoreForChannel returns the store for the given channel
func (sp *StoreProvider) StoreForChannel(channelID string) olapi.Store {
	sp.RLock()
	defer sp.RUnlock()
	return sp.stores[channelID]
}

// OpenStore opens the store for the given channel
func (sp *StoreProvider) OpenStore(channelID string) (olapi.Store, error) {
	sp.Lock()
	defer sp.Unlock()

	store, ok := sp.stores[channelID]
	if !ok {
		store = newStore(channelID, sp.dbProvider, sp.collConfigs)
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

// getDBProvider returns the DB provider. This var may be overridden by unit tests
var getDBProvider = func() api.DBProvider {
	return couchdbstore.NewDBProvider()
}
