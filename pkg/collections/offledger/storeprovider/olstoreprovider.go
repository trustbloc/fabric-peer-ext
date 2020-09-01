/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package storeprovider

import (
	"sync"

	pb "github.com/hyperledger/fabric-protos-go/peer"
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"

	collcommon "github.com/trustbloc/fabric-peer-ext/pkg/collections/common"
	olapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/storeprovider/store/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/storeprovider/store/couchdbstore"
	"github.com/trustbloc/fabric-peer-ext/pkg/config"
)

// Option is a store provider option
type Option func(p *StoreProvider)

// CollOption is a collection option
type CollOption func(c *collTypeConfig)

// WithCollectionType adds a collection type to the set of collection types supported by the off-ledger store
// along with any options
func WithCollectionType(collType pb.CollectionType, opts ...CollOption) Option {
	return func(p *StoreProvider) {
		c := &collTypeConfig{}
		p.collConfigs[collType] = c

		for _, opt := range opts {
			opt(c)
		}
	}
}

// Decorator allows the key/value to be modified/validated before being persisted.
type Decorator interface {
	// BeforeSave has the opportunity to decorate the key and/or value before the key-value is saved.
	BeforeSave(key *storeapi.Key, value *storeapi.ExpiringValue) (*storeapi.Key, *storeapi.ExpiringValue, error)

	// BeforeLoad has the opportunity to decorate the key before it is loaded/deleted.
	BeforeLoad(key *storeapi.Key) (*storeapi.Key, error)
}

// WithDecorator sets a decorator for a collection type allowing the key/value to be validated/modified before being persisted
func WithDecorator(decorator Decorator) CollOption {
	return func(c *collTypeConfig) {
		c.decorator = decorator
	}
}

// WithCacheEnabled enables caching for the given collection type
func WithCacheEnabled() CollOption {
	return func(c *collTypeConfig) {
		c.enableCache = true
	}
}

type collTypeConfig struct {
	decorator   Decorator
	enableCache bool
}

// New returns a store provider factory
func New(
	identifierProvider collcommon.IdentifierProvider,
	identityDeserializerProvider collcommon.IdentityDeserializerProvider,
	collConfigProvider collcommon.CollectionConfigProvider,
	opts ...Option) *StoreProvider {
	p := &StoreProvider{
		stores:                       make(map[string]olapi.Store),
		dbProvider:                   getDBProvider(),
		identifierProvider:           identifierProvider,
		identityDeserializerProvider: identityDeserializerProvider,
		collConfigs:                  make(map[pb.CollectionType]*collTypeConfig),
		collConfigProvider:           collConfigProvider,
	}

	// OFF_LEDGER collection type supported by default
	opts = append(opts, WithCollectionType(pb.CollectionType_COL_OFFLEDGER))

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
	dbProvider                   api.DBProvider
	identifierProvider           collcommon.IdentifierProvider
	identityDeserializerProvider collcommon.IdentityDeserializerProvider
	collConfigs                  map[pb.CollectionType]*collTypeConfig
	collConfigProvider           collcommon.CollectionConfigProvider
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
		store = newStore(
			channelID,
			&olConfig{cacheSize: config.GetOLCollCacheSize()},
			sp.collConfigs,
			&providers{
				dbProvider:           sp.dbProvider,
				identifierProvider:   sp.identifierProvider,
				identityDeserializer: sp.identityDeserializerProvider.GetIdentityDeserializer(channelID),
				collConfigRetriever:  sp.collConfigProvider.ForChannel(channelID),
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

// getDBProvider returns the DB provider. This var may be overridden by unit tests
var getDBProvider = func() api.DBProvider {
	return couchdbstore.NewDBProvider()
}
