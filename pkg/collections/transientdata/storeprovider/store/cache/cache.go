/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cache

import (
	"fmt"
	"time"

	"github.com/bluele/gcache"
	"github.com/hyperledger/fabric/common/flogging"

	"github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/storeprovider/store/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/config"
)

var logger = flogging.MustGetLogger("transientdata")

// Cache is an in-memory key-value cache
type Cache struct {
	channelID     string
	cache         gcache.Cache
	ticker        *time.Ticker
	dbstore       transientDB
	alwaysPersist bool
}

// transientDB - an interface for persisting and retrieving keys
type transientDB interface {
	AddKey(api.Key, *api.Value) error
	DeleteExpiredKeys() error
	GetKey(key api.Key) (*api.Value, error)
}

// New return a new in-memory key-value cache
func New(channelID string, size int, alwaysPersist bool, dbstore transientDB) *Cache {
	c := &Cache{
		channelID:     channelID,
		ticker:        time.NewTicker(config.GetTransientDataExpiredIntervalTime()),
		dbstore:       dbstore,
		alwaysPersist: alwaysPersist,
	}

	cb := gcache.New(size).
		LoaderExpireFunc(c.loadFromDB).
		ARC()

	if !alwaysPersist {
		cb.EvictedFunc(c.evict)
	}

	c.cache = cb.Build()

	// cleanup expired data in db
	go c.periodicPurge()

	return c
}

// Close closes the cache
func (c *Cache) Close() {
	c.cache.Purge()
	c.ticker.Stop()
}

// Put adds the transient value for the given key.
func (c *Cache) Put(key api.Key, value []byte, txID string) {
	v := &api.Value{
		Value: value,
		TxID:  txID,
	}

	if err := c.cache.Set(key, v); err != nil {
		panic("Set must never return an error")
	}

	if c.alwaysPersist {
		c.persist(key, v)
	}
}

// PutWithExpire adds the transient value for the given key.
func (c *Cache) PutWithExpire(key api.Key, value []byte, txID string, expiry time.Duration) {
	v := &api.Value{
		Value:      value,
		TxID:       txID,
		ExpiryTime: time.Now().UTC().Add(expiry),
	}

	if err := c.cache.SetWithExpire(key, v, expiry); err != nil {
		panic("Set must never return an error")
	}

	if c.alwaysPersist {
		c.persist(key, v)
	}
}

// Get returns the transient value for the given key
func (c *Cache) Get(key api.Key) *api.Value {
	value, err := c.cache.Get(key)
	if err != nil {
		if err != gcache.KeyNotFoundError {
			panic(fmt.Sprintf("Get must never return an error other than KeyNotFoundError err:%s", err))
		}
		return nil
	}

	return value.(*api.Value)
}

func (c *Cache) loadFromDB(key interface{}) (interface{}, *time.Duration, error) {
	logger.Debugf("[%s] Loading key from database: %s", c.channelID, key)

	value, err := c.dbstore.GetKey(key.(api.Key))
	if value == nil || err != nil {
		if err != nil {
			logger.Error(err.Error())
		}
		logger.Debugf("[%s] Key [%s] not found in DB", c.channelID, key)
		return nil, nil, gcache.KeyNotFoundError
	}
	isExpired, diff := c.checkExpiryTime(value.ExpiryTime)
	if isExpired {
		logger.Debugf("[%s] Key [%s] from DB has expired", c.channelID, key)
		return nil, nil, gcache.KeyNotFoundError
	}
	logger.Debugf("[%s] Loaded key [%s] from DB", c.channelID, key)
	return value, &diff, nil
}

func (c *Cache) evict(key, value interface{}) {
	if value != nil {
		logger.Debugf("[%s] Evicting key to database: %s", c.channelID, key)

		c.persist(key.(api.Key), value.(*api.Value))
	}
}

func (c *Cache) persist(key api.Key, value *api.Value) {
	isExpired, _ := c.checkExpiryTime(value.ExpiryTime)
	if isExpired {
		logger.Debugf("[%s] Not persisting key [%s] since the value expired at [%s]", c.channelID, key, value.ExpiryTime)

		return
	}

	logger.Debugf("[%s] Persisting key [%s]", c.channelID, key)

	dbstoreErr := c.dbstore.AddKey(key, value)
	if dbstoreErr != nil {
		logger.Errorf("[%s] Key [%s] could not be offloaded to DB: %s", c.channelID, key, dbstoreErr.Error())
	} else {
		logger.Debugf("[%s] Key [%s] offloaded to DB", c.channelID, key)
	}
}

func (c *Cache) periodicPurge() {
	for range c.ticker.C {
		dbstoreErr := c.dbstore.DeleteExpiredKeys()
		if dbstoreErr != nil {
			logger.Error(dbstoreErr.Error())
		}
	}
}

func (c *Cache) checkExpiryTime(expiryTime time.Time) (bool, time.Duration) {
	if expiryTime.IsZero() {
		return false, 0
	}

	timeNow := time.Now().UTC()
	logger.Debugf("[%s] Checking expiration - Current time: %s, Expiry time: %s", c.channelID, timeNow, expiryTime)

	diff := expiryTime.Sub(timeNow)
	return diff <= 0, diff
}
