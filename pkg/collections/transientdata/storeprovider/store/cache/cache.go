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

var logger = flogging.MustGetLogger("memtransientdatastore")

// Cache is an in-memory key-value cache
type Cache struct {
	cache   gcache.Cache
	ticker  *time.Ticker
	dbstore transientDB
}

// transientDB - an interface for persisting and retrieving keys
type transientDB interface {
	AddKey(api.Key, *api.Value) error
	DeleteExpiredKeys() error
	GetKey(key api.Key) (*api.Value, error)
}

// New return a new in-memory key-value cache
func New(size int, dbstore transientDB) *Cache {
	c := &Cache{
		ticker:  time.NewTicker(config.GetTransientDataExpiredIntervalTime()),
		dbstore: dbstore,
	}

	c.cache = gcache.New(size).
		LoaderExpireFunc(c.loadFromDB).
		EvictedFunc(c.storeToDB).
		ARC().Build()

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
	if err := c.cache.Set(key,
		&api.Value{
			Value: value,
			TxID:  txID,
		}); err != nil {
		panic("Set must never return an error")
	}
}

// PutWithExpire adds the transient value for the given key.
func (c *Cache) PutWithExpire(key api.Key, value []byte, txID string, expiry time.Duration) {
	if err := c.cache.SetWithExpire(key,
		&api.Value{
			Value:      value,
			TxID:       txID,
			ExpiryTime: time.Now().UTC().Add(expiry),
		}, expiry); err != nil {
		panic("Set must never return an error")
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
	logger.Debugf("LoaderExpireFunc for key %s", key)
	value, err := c.dbstore.GetKey(key.(api.Key))
	if value == nil || err != nil {
		if err != nil {
			logger.Error(err.Error())
		}
		logger.Debugf("Key [%s] not found in DB", key)
		return nil, nil, gcache.KeyNotFoundError
	}
	isExpired, diff := checkExpiryTime(value.ExpiryTime)
	if isExpired {
		logger.Debugf("Key [%s] from DB has expired", key)
		return nil, nil, gcache.KeyNotFoundError
	}
	logger.Debugf("Loaded key [%s] from DB", key)
	return value, &diff, nil
}

func (c *Cache) storeToDB(key, value interface{}) {
	logger.Debugf("EvictedFunc for key %s", key)
	if value != nil {
		k := key.(api.Key)
		v := value.(*api.Value)
		isExpired, _ := checkExpiryTime(v.ExpiryTime)
		if !isExpired {
			dbstoreErr := c.dbstore.AddKey(k, v)
			if dbstoreErr != nil {
				logger.Error(dbstoreErr.Error())
			} else {
				logger.Debugf("Key [%s] offloaded to DB", key)
			}
		}
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

func checkExpiryTime(expiryTime time.Time) (bool, time.Duration) {
	if expiryTime.IsZero() {
		return false, 0
	}

	timeNow := time.Now().UTC()
	logger.Debugf("Checking expiration - Current time: %s, Expiry time: %s", timeNow, expiryTime)

	diff := expiryTime.Sub(timeNow)
	return diff <= 0, diff
}
