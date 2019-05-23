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
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/storeprovider/store/api"
)

var logger = flogging.MustGetLogger("ext_offledger")

// Cache implements a cache for collection data
type Cache struct {
	channelID  string
	cache      gcache.Cache
	dbProvider api.DBProvider
}

type cacheKey struct {
	namespace  string
	collection string
	key        string
}

func (k cacheKey) String() string {
	return fmt.Sprintf("%s:%s:%s", k.namespace, k.collection, k.key)
}

// New returns a new collection data cache
func New(channelID string, dbProvider api.DBProvider, size int) *Cache {
	c := &Cache{
		channelID:  channelID,
		dbProvider: dbProvider,
	}
	c.cache = gcache.New(size).ARC().LoaderExpireFunc(
		func(k interface{}) (interface{}, *time.Duration, error) {
			key := k.(cacheKey)
			v, remainingTime, err := c.load(key)
			if err != nil {
				logger.Warningf("[%s] Error loading key [%s]: %s", c.channelID, key, err)
				return nil, nil, err
			}
			return v, remainingTime, nil
		}).Build()
	return c
}

// Put adds the value for the given key.
func (c *Cache) Put(ns, coll, key string, value *api.Value) {
	cKey := cacheKey{
		namespace:  ns,
		collection: coll,
		key:        key,
	}

	var err error
	if value.ExpiryTime.IsZero() {
		logger.Debugf("[%s] Putting key [%s]. Expires: NEVER", c.channelID, cKey)
		err = c.cache.Set(cKey, value)
	} else if value.ExpiryTime.Before(time.Now()) {
		logger.Debugf("[%s] Expiry time for key [%s] occurs in the past. Value will not be added", c.channelID, cKey)
	} else {
		logger.Debugf("[%s] Putting key [%s]. Expires: %s", c.channelID, cKey, value.ExpiryTime)
		err = c.cache.SetWithExpire(cKey, value, time.Until(value.ExpiryTime))
	}

	if err != nil {
		panic("Set must never return an error")
	}
}

// Get returns the values for the given keys
func (c *Cache) Get(ns, coll, key string) (*api.Value, error) {
	cKey := cacheKey{
		namespace:  ns,
		collection: coll,
		key:        key,
	}

	value, err := c.cache.Get(cKey)
	if err != nil {
		logger.Warningf("[%s] Error getting key [%s]: %s", c.channelID, cKey, err)
		return nil, err
	}

	if value == nil {
		logger.Debugf("[%s] Key not found [%s]", c.channelID, cKey)
		return nil, nil
	}

	v, ok := value.(*api.Value)
	if !ok {
		panic("Invalid value type!")
	}
	if v == nil {
		logger.Debugf("[%s] Key not found [%s]", c.channelID, cKey)
		return nil, nil
	}

	logger.Debugf("[%s] Got key [%s]. Expires: %s", c.channelID, cKey, v.ExpiryTime)
	return v, nil
}

// GetMultiple returns the values for the given keys
func (c *Cache) GetMultiple(ns, coll string, keys ...string) ([]*api.Value, error) {
	values := make([]*api.Value, len(keys))
	for i, key := range keys {
		value, err := c.Get(ns, coll, key)
		if err != nil {
			return nil, err
		}
		values[i] = value
	}
	return values, nil
}

func (c *Cache) load(key cacheKey) (*api.Value, *time.Duration, error) {
	db, err := c.dbProvider.GetDB(key.namespace, key.collection)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "error getting database")
	}

	logger.Debugf("Loading value for key %s from DB", key)
	v, err := db.Get(key.key)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "error loading value")
	}

	logger.Debugf("Loaded value %v for key %s from DB", v, key)
	if v == nil {
		logger.Debugf("[%s] Value not found for key [%s]", c.channelID, key)
		return nil, nil, nil
	}

	logger.Debugf("Checking expiry time for key %s", key)
	if v.ExpiryTime.IsZero() {
		logger.Debugf("[%s] Loaded key [%s]. Expires: NEVER", c.channelID, key)
		return v, nil, nil

	}
	remainingTime := time.Until(v.ExpiryTime)
	if remainingTime < 0 {
		// Already expired
		logger.Debugf("[%s] Loaded key [%s]. Expires: NOW", c.channelID, key)
		return nil, nil, nil
	}

	logger.Debugf("[%s] Loaded key [%s]. Expires: %s", c.channelID, key, v.ExpiryTime)
	return v, &remainingTime, nil
}
