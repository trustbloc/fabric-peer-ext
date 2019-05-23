/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cache

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/storeprovider/store/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/storeprovider/store/mocks"
)

const (
	channelID = "testchannel"

	ns1 = "ns1"

	coll1 = "coll1"
	coll2 = "coll2"

	key1 = "key1"
	key2 = "key2"
	key3 = "key3"
	key4 = "key4"
	key5 = "key5"

	txID1 = "tx1"
)

var (
	value1 = []byte("value1")
	value2 = []byte("value2")
)

func TestCache_PutAndGet(t *testing.T) {
	now := time.Now()
	v1 := &api.Value{Value: value1, TxID: txID1}
	v2 := &api.Value{Value: value2, TxID: txID1}
	v4 := &api.Value{Value: value2, TxID: txID1, ExpiryTime: now.Add(50 * time.Millisecond)}
	v5 := &api.Value{Value: value2, TxID: txID1, ExpiryTime: now.Add(-1 * time.Minute)}

	dbProvider := mocks.NewDBProvider()

	c := New(channelID, dbProvider, 100)
	require.NotNil(t, c)

	c.Put(ns1, coll1, key1, v1)
	c.Put(ns1, coll1, key2, v2)
	c.Put(ns1, coll1, key4, v4)
	c.Put(ns1, coll1, key5, v5)

	v, err := c.Get(ns1, coll1, key1)
	assert.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, v1, v)

	values, err := c.GetMultiple(ns1, coll1, key1, key2, key3, key4, key5)
	assert.NoError(t, err)
	require.Equal(t, 5, len(values))
	assert.Equal(t, v1, values[0])
	assert.Equal(t, v2, values[1])
	assert.Nil(t, values[2]) // Not added
	assert.Equal(t, v4, values[3])
	assert.Nil(t, values[4]) // Should not have been added

	time.Sleep(100 * time.Millisecond)

	v, err = c.Get(ns1, coll1, key4)
	assert.NoError(t, err)
	require.Nil(t, v) // Should have expired
}

func TestCache_LoadFromDB(t *testing.T) {
	valueWithExpiry := &api.Value{
		Value:      []byte("value1"),
		ExpiryTime: time.Now().Add(time.Minute),
	}
	valueWithNoExpiry := &api.Value{
		Value: []byte("value1"),
	}
	expiredValue := &api.Value{
		Value:      []byte("value1"),
		ExpiryTime: time.Now().Add(-1 * time.Minute),
	}

	dbProvider := mocks.NewDBProvider().
		WithValue(ns1, coll1, key1, valueWithExpiry).
		WithValue(ns1, coll1, key2, valueWithNoExpiry).
		WithValue(ns1, coll1, key3, expiredValue)

	c := New(channelID, dbProvider, 100)
	require.NotNil(t, c)

	// Not found
	v, err := c.Get(ns1, coll2, key1)
	assert.NoError(t, err)
	assert.Nil(t, v)

	// Found
	v, err = c.Get(ns1, coll1, key1)
	assert.NoError(t, err)
	assert.Equal(t, valueWithExpiry, v)
	v, err = c.Get(ns1, coll1, key2)
	assert.NoError(t, err)
	assert.Equal(t, valueWithNoExpiry, v)

	// Expired
	v, err = c.Get(ns1, coll1, key3)
	assert.NoError(t, err)
	assert.Nil(t, v)
}

func TestCache_LoadFromDBError(t *testing.T) {
	t.Run("Error getting DB", func(t *testing.T) {
		expectedErr := fmt.Errorf("get DB error")
		dbProvider := mocks.NewDBProvider().
			WithError(expectedErr)

		c := New(channelID, dbProvider, 100)
		require.NotNil(t, c)

		v, err := c.Get(ns1, coll1, key1)
		require.Error(t, err)
		assert.Nil(t, v)
		assert.Contains(t, err.Error(), expectedErr.Error())
	})

	t.Run("Error loading key from DB", func(t *testing.T) {
		expectedErr := fmt.Errorf("load key from DB error")
		dbProvider := mocks.NewDBProvider()
		dbProvider.MockDB(ns1, coll1).WithError(expectedErr)

		c := New(channelID, dbProvider, 100)
		require.NotNil(t, c)

		v, err := c.Get(ns1, coll1, key1)
		require.Error(t, err)
		assert.Nil(t, v)
		assert.Contains(t, err.Error(), expectedErr.Error())

		values, err := c.GetMultiple(ns1, coll1, key1, key2)
		require.Error(t, err)
		assert.Nil(t, values)
		assert.Contains(t, err.Error(), expectedErr.Error())
	})
}

func TestCache_ReloadFromDB(t *testing.T) {
	v1 := &api.Value{
		Value:      []byte("value1"),
		ExpiryTime: time.Now().Add(50 * time.Millisecond),
	}
	v2 := &api.Value{
		Value:      []byte("value2"),
		ExpiryTime: time.Now().Add(time.Minute),
	}
	v3 := &api.Value{
		Value:      []byte("value3"),
		ExpiryTime: time.Now().Add(time.Minute),
	}
	v4 := &api.Value{
		Value:      []byte("value4"),
		ExpiryTime: time.Now().Add(time.Minute),
	}

	dbProvider := mocks.NewDBProvider().
		WithValue(ns1, coll1, key1, v1)

	c := New(channelID, dbProvider, 1)
	require.NotNil(t, c)

	// v1 should be retrieved from the DB and cached
	v, err := c.Get(ns1, coll1, key1)
	assert.NoError(t, err)
	assert.Equal(t, v1, v)

	// Update the value in the DB
	dbProvider.WithValue(ns1, coll1, key1, v2)

	// v1 should still be retrieved from cache
	v, err = c.Get(ns1, coll1, key1)
	assert.NoError(t, err)
	assert.Equal(t, v1, v)

	time.Sleep(100 * time.Millisecond)

	// key1 should have expired and v2 will be retrieved from the DB
	v, err = c.Get(ns1, coll1, key1)
	assert.NoError(t, err)
	assert.Equal(t, v2, v)

	// Update the value in the DB
	dbProvider.WithValue(ns1, coll1, key1, v3)

	// The following call should evict key1 from the cache
	c.Put(ns1, coll1, key2, v4)

	// key1 should have been evicted and v3 will be retrieved from the DB
	v, err = c.Get(ns1, coll1, key1)
	assert.NoError(t, err)
	assert.Equal(t, v3, v)
}

func Test_Concurrency(t *testing.T) {
	const (
		nReaders = 10
		nWriters = 3
		nItems   = 1000
		nReads   = 1000
		txID1    = "tx1"
	)

	dbProvider := mocks.NewDBProvider()

	c := New(channelID, dbProvider, 100)

	var wWg sync.WaitGroup
	wWg.Add(nWriters)

	var rWg sync.WaitGroup
	rWg.Add(nReaders)

	for i := 0; i < nWriters; i++ {
		go func() {
			defer wWg.Done()

			for j := 0; j < nItems; j++ {
				key := fmt.Sprintf("k_%d", j)
				expiry := time.Now().Add(getRandomDuration(100) * time.Millisecond)
				c.Put(ns1, coll1, key, &api.Value{Value: []byte(key), TxID: txID1, ExpiryTime: expiry})
				time.Sleep(time.Millisecond)
			}
		}()
	}

	var mutex sync.Mutex
	var errors []error
	for i := 0; i < nReaders; i++ {
		go func() {
			defer rWg.Done()

			for j := 0; j < nReads; j++ {
				_, err := c.Get(ns1, coll1, getRandomKey(nItems))
				if err != nil {
					mutex.Lock()
					errors = append(errors, err)
					mutex.Unlock()
				}
			}
		}()
	}

	wWg.Wait()
	rWg.Wait()

	if len(errors) > 0 {
		t.Errorf("Got errors: %s", errors)
	}
}

func getRandomKey(max int32) string {
	return fmt.Sprintf("k_%d", rand.Int31n(max))
}

func getRandomDuration(max int32) time.Duration {
	return time.Duration(rand.Int31n(max))
}
