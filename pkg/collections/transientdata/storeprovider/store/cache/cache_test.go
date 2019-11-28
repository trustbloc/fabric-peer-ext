/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cache

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/storeprovider/store/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/storeprovider/store/dbstore"
	"github.com/trustbloc/fabric-peer-ext/pkg/config"
)

const (
	txID1 = "txid1"
	txID2 = "txid2"

	ns1 = "namespace1"
	ns2 = "namespace2"

	coll1 = "coll1"
	coll2 = "coll2"

	key1 = "key1"
	key2 = "key2"
	key3 = "key3"
)

var (
	k1 = api.Key{
		Namespace:  ns1,
		Collection: coll1,
		Key:        key1,
	}
	k2 = api.Key{
		Namespace:  ns2,
		Collection: coll2,
		Key:        key2,
	}
	k3 = api.Key{
		Namespace:  ns1,
		Collection: coll2,
		Key:        key2,
	}
	k4 = api.Key{
		Namespace:  ns1,
		Collection: coll2,
		Key:        key3,
	}
	v1 = []byte("v1")
	v2 = []byte("v2")
)

func TestCleanupExpiredTransientDataFromDB(t *testing.T) {
	defer removeDBPath(t)

	p, err := dbstore.NewDBProvider()
	require.NoError(t, err)
	require.NotNil(t, p)
	defer p.Close()

	db, err := p.OpenDBStore("testchannel")
	require.NoError(t, err)
	require.NotNil(t, db)

	cache := New(1, db)
	require.NotNil(t, cache)

	cache.PutWithExpire(k1, v1, txID1, 2*time.Second)
	cache.PutWithExpire(k2, v1, txID1, 1*time.Second)
	cache.Put(k3, v1, txID1)
	cache.Put(k4, v1, txID1)

	// wait one second and half to confirm cleanup remove k2
	time.Sleep(1500 * time.Millisecond)

	// check data exist
	v, err := cache.dbstore.GetKey(k1)
	require.Nil(t, err)
	require.NotNil(t, v)
	// check data not exist because it's expired
	v, err = cache.dbstore.GetKey(k2)
	require.Nil(t, err)
	require.Nil(t, v)
	// check data exist
	v, err = cache.dbstore.GetKey(k3)
	require.Nil(t, err)
	require.NotNil(t, v)

	// wait one more second and half to confirm cleanup remove k1
	time.Sleep(1500 * time.Millisecond)
	// check data not exist because it's expired
	v, err = cache.dbstore.GetKey(k1)
	require.Nil(t, err)
	require.Nil(t, v)
	// check data exist
	v, err = cache.dbstore.GetKey(k3)
	require.Nil(t, err)
	require.NotNil(t, v)

}

func TestRetrieveTransientDataFromDB(t *testing.T) {
	t.Run("getAfterPurgeFromCache", func(t *testing.T) {
		defer removeDBPath(t)
		p, err := dbstore.NewDBProvider()
		require.NoError(t, err)
		require.NotNil(t, p)
		defer p.Close()

		db, err := p.OpenDBStore("testchannel")
		require.NoError(t, err)
		require.NotNil(t, db)

		cache := New(1, db)
		require.NotNil(t, cache)
		v := cache.Get(k1)
		require.Nil(t, v)

		cache.Put(k1, v1, txID1)
		// Get k1 from cache
		v = cache.Get(k1)
		require.NotNil(t, v)
		require.Equal(t, txID1, v.TxID)
		require.Equal(t, v1, v.Value)

		// After put the k2 the k1 will be purged from cache
		cache.Put(k2, v2, txID2)

		// k1 will not be found in cache so will get it from db
		v = cache.Get(k1)
		require.NotNil(t, v)
		require.Equal(t, txID1, v.TxID)
		require.Equal(t, v1, v.Value)

	})

	t.Run("getAfterPurgeFromCacheForNotExpiredData", func(t *testing.T) {
		defer removeDBPath(t)
		p, err := dbstore.NewDBProvider()
		require.NoError(t, err)
		require.NotNil(t, p)
		defer p.Close()

		db, err := p.OpenDBStore("testchannel")
		require.NoError(t, err)
		require.NotNil(t, db)

		cache := New(1, db)
		require.NotNil(t, cache)
		v := cache.Get(k1)
		require.Nil(t, v)

		cache.PutWithExpire(k1, v1, txID1, 5*time.Second)
		// Get k1 from cache
		v = cache.Get(k1)
		require.NotNil(t, v)

		// After put the k2 the k1 will be purged from cache
		cache.PutWithExpire(k2, v2, txID2, 5*time.Second)

		// k1 will not be found in cache so will get it from db
		v = cache.Get(k1)
		require.NotNil(t, v)

	})

	t.Run("getAfterPurgeFromCacheForExpiredData", func(t *testing.T) {
		defer removeDBPath(t)
		p, err := dbstore.NewDBProvider()
		require.NoError(t, err)
		require.NotNil(t, p)
		defer p.Close()

		db, err := p.OpenDBStore("testchannel")
		require.NoError(t, err)
		require.NotNil(t, db)

		cache := New(1, db)
		require.NotNil(t, cache)
		v := cache.Get(k1)
		require.Nil(t, v)

		cache.PutWithExpire(k1, v1, txID1, 2*time.Second)
		// Get k1 from cache
		v = cache.Get(k1)
		require.NotNil(t, v)

		// After put the k2 the k1 will be purged from cache
		cache.PutWithExpire(k2, v2, txID2, 5*time.Second)

		// k1 will not be found in cache so will get it from db
		time.Sleep(1 * time.Second)
		v = cache.Get(k1)
		require.NotNil(t, v)

		// k1 will not be found in db because it's already expired
		time.Sleep(1 * time.Second)
		v = cache.Get(k1)
		require.Nil(t, v)

	})

}

func TestTransientDataCache(t *testing.T) {
	defer removeDBPath(t)
	p, err := dbstore.NewDBProvider()
	require.NoError(t, err)
	require.NotNil(t, p)
	defer p.Close()

	db, err := p.OpenDBStore("testchannel")
	require.NoError(t, err)
	require.NotNil(t, db)

	cache := New(1, db)
	require.NotNil(t, cache)
	defer cache.Close()

	t.Run("Key", func(t *testing.T) {
		require.Equal(t, "namespace1:coll1:key1", k1.String())
	})

	t.Run("GetAndPut", func(t *testing.T) {
		v := cache.Get(k1)
		require.Nil(t, v)

		cache.Put(k1, v1, txID1)
		cache.Put(k2, v2, txID2)

		v = cache.Get(k1)
		require.NotNil(t, v)
		require.Equal(t, txID1, v.TxID)
		require.Equal(t, v1, v.Value)

		v = cache.Get(k2)
		require.NotNil(t, v)
		require.Equal(t, txID2, v.TxID)
		require.Equal(t, v2, v.Value)
	})

	t.Run("Expire", func(t *testing.T) {
		expiration := 10 * time.Millisecond
		cache.PutWithExpire(k3, v1, txID1, expiration)

		v := cache.Get(k3)
		require.NotNil(t, v)

		time.Sleep(100 * time.Millisecond)
		v = cache.Get(k3)
		require.Nil(t, v)
	})
}

func TestTransientDataCacheConcurrency(t *testing.T) {
	defer removeDBPath(t)
	p, err := dbstore.NewDBProvider()
	require.NoError(t, err)
	require.NotNil(t, p)
	defer p.Close()

	db, err := p.OpenDBStore("testchannel")
	require.NoError(t, err)
	require.NotNil(t, db)

	cache := New(1, db)
	require.NotNil(t, cache)
	defer cache.Close()

	n := 100

	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		k := api.Key{
			Namespace:  ns1,
			Collection: coll1,
			Key:        fmt.Sprintf("k_%d", i),
		}
		v := []byte(fmt.Sprintf("v_%d", i))

		go func() {
			defer wg.Done()

			cache.Put(k, v, txID1)
			val := cache.Get(k)
			if val == nil {
				panic("Unable to get value for key")
			}
			if string(val.Value) != string(v) {
				panic("Unable to get value for key")
			}
		}()
	}

	wg.Wait()
}

func Test_Error(t *testing.T) {
	errExpected := errors.New("db error")
	db := newMockDB().WithError(errExpected)
	c := New(100, db)

	v := c.Get(k1)
	require.Nil(t, v)

	// Let periodic purge run with DB error to ensure it doesn't panic
	time.Sleep(200 * time.Millisecond)
}

func TestMain(m *testing.M) {
	removeDBPath(nil)
	viper.Set("peer.fileSystemPath", "/tmp/fabric/ledgertests/transientdatadb")
	viper.Set("coll.transientdata.cleanupExpired.Interval", "50ms")

	os.Exit(m.Run())
}

func removeDBPath(t testing.TB) {
	removePath(t, config.GetTransientDataLevelDBPath())
}

func removePath(t testing.TB, path string) {
	if err := os.RemoveAll(path); err != nil {
		t.Fatalf("Err: %s", err)
	}
}

type mockDB struct {
	err error
}

func newMockDB() *mockDB {
	return &mockDB{}
}

func (m *mockDB) WithError(err error) *mockDB {
	m.err = err
	return m
}

func (m *mockDB) AddKey(api.Key, *api.Value) error {
	if m.err != nil {
		return m.err
	}
	panic("not implemented")
}

func (m *mockDB) DeleteExpiredKeys() error {
	if m.err != nil {
		return m.err
	}
	panic("not implemented")
}

func (m *mockDB) GetKey(key api.Key) (*api.Value, error) {
	if m.err != nil {
		return nil, m.err
	}
	panic("not implemented")
}
