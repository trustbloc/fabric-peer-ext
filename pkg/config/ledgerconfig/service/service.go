/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"encoding/json"
	"sync"

	"github.com/bluele/gcache"
	"github.com/hyperledger/fabric-protos-go/ledger/rwset/kvrwset"
	"github.com/hyperledger/fabric/common/flogging"
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	"github.com/pkg/errors"
	cmnconfig "github.com/trustbloc/fabric-peer-ext/pkg/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/mgr"
	state "github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/state/api"
)

var logger = flogging.MustGetLogger("ledgerconfig")

const (
	// ConfigNS is the namespace (chaincode name) under which configuration data is stored.
	ConfigNS = "configscc"
)

// ErrConfigNotFound indicates that the config for the given key was not found
var ErrConfigNotFound = errors.New("config not found")

type configMgr interface {
	Query(criteria *config.Criteria) ([]*config.KeyValue, error)
}

// ConfigService manages configuration data for a given channel
type ConfigService struct {
	channelID  string
	configMgr  configMgr
	cache      gcache.Cache
	handlers   []config.UpdateHandler
	mutex      sync.RWMutex
	updateChan chan *config.KeyValue
}

type blockPublisher interface {
	AddWriteHandler(handler gossipapi.WriteHandler)
}

// New returns a new config service
func New(channelID string, retrieverProvider state.RetrieverProvider, publisher blockPublisher) *ConfigService {
	s := &ConfigService{
		channelID:  channelID,
		configMgr:  mgr.NewQueryManager(ConfigNS, retrieverProvider),
		updateChan: make(chan *config.KeyValue, cmnconfig.GetConfigUpdatePublisherBufferSize()),
	}

	// Set size to 0 so that all config is cached
	s.cache = gcache.New(0).
		LoaderFunc(func(key interface{}) (interface{}, error) {
			return s.load(key.(config.Key))
		}).
		Build()

	// Register for KV write events so we can invalidate our cache when config is updated/deleted
	publisher.AddWriteHandler(func(txMetadata gossipapi.TxMetadata, ns string, kvWrite *kvrwset.KVWrite) error {
		if ns != ConfigNS {
			// Only interested in config chaincode
			return nil
		}
		return s.handleKeyUpdate(kvWrite)
	})

	// Listen for and forward config updates to subscribers
	go s.listen()

	return s
}

// Get returns the config bytes for the given criteria.
// If the key is not found then ErrConfigNotFound error is returned
func (s *ConfigService) Get(key *config.Key) (*config.Value, error) {
	err := key.Validate()
	if err != nil {
		return nil, err
	}
	value, err := s.cache.Get(*key)
	if err != nil {
		return nil, err
	}
	return value.(*config.Value), nil
}

// Query retrieves configurations based on the provided criteria.
func (s *ConfigService) Query(criteria *config.Criteria) ([]*config.KeyValue, error) {
	err := criteria.Validate()
	if err != nil {
		return nil, err
	}

	return s.configMgr.Query(criteria)
}

// AddUpdateHandler adds a handler that is notified of config updates/deletes
func (s *ConfigService) AddUpdateHandler(handler config.UpdateHandler) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.handlers = append(s.handlers, handler)
}

func (s *ConfigService) load(key config.Key) (*config.Value, error) {
	logger.Debugf("[%s] Loading key [%s] from ledger...", s.channelID, key)
	results, err := s.configMgr.Query(config.CriteriaFromKey(&key))
	if err != nil {
		return nil, err
	}

	logger.Debugf("[%s] ... received results for key [%s]: %s", s.channelID, key, results)

	if len(results) > 1 {
		return nil, errors.Errorf("received more than one result for key [%s]", key)
	}
	if len(results) == 0 {
		return nil, ErrConfigNotFound
	}
	return results[0].Value, nil
}

func (s *ConfigService) handleKeyUpdate(kvWrite *kvrwset.KVWrite) error {
	logger.Debugf("[%s] Got KV write: [%s]", s.channelID, kvWrite.Key)
	key, err := mgr.UnmarshalKey(kvWrite.Key)
	if err != nil {
		// Not a Key - could be an index
		logger.Debugf("[%s] KV write [%s] is not a config key. Ignoring.", s.channelID, kvWrite.Key)
		return nil
	}

	if kvWrite.IsDelete {
		if s.cache.Remove(*key) {
			logger.Debugf("[%s] Removed deleted config key [%s] from cache", s.channelID, key)
		} else {
			logger.Debugf("[%s] Deleted config key [%s] not found in cache", s.channelID, key)
		}

		// Notify subscribers
		s.updateChan <- config.NewKeyValue(key, nil)
		return nil
	}

	value := &config.Value{}
	if err := json.Unmarshal(kvWrite.Value, value); err != nil {
		logger.Errorf("[%s] Error unmarshalling config value for key [%s]: %s", s.channelID, key, err)
		return err
	}

	logger.Debugf("[%s] Adding config key [%s] to cache", s.channelID, key)
	if err := s.cache.Set(*key, value); err != nil {
		logger.Errorf("[%s] Error caching config value for key [%s]: %s", s.channelID, key, err)
		return err
	}

	// Notify subscribers
	s.updateChan <- config.NewKeyValue(key, value)
	return nil
}

func (s *ConfigService) listen() {
	for kv := range s.updateChan {
		s.notify(kv)
	}
}

func (s *ConfigService) getHandlers() []config.UpdateHandler {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	handlers := make([]config.UpdateHandler, len(s.handlers))
	copy(handlers, s.handlers)
	return handlers
}

func (s *ConfigService) notify(kv *config.KeyValue) {
	logger.Debugf("[%s] Notifying subscribers of config update: [%s]", s.channelID, kv)
	for _, handleUpdate := range s.getHandlers() {
		handleUpdate(kv)
	}
}
