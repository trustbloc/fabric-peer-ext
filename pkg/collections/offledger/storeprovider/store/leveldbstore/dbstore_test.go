/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package leveldbstore

import (
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/storeprovider/store/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/config"
)

const (
	txID1 = "txid1"
	txID2 = "txid2"
	ns1   = "namespace1"
	coll2 = "coll2"
	coll1 = "coll1"
	key1  = "key1"
	key2  = "key2"
)

var (
	value1 = []byte("v111")
	value2 = []byte("v222")
)

func TestDeleteExpiredKeysFromDB(t *testing.T) {
	removeDBPath(t)
	defer removeDBPath(t)

	provider := NewDBProvider()
	defer provider.Close()

	db, err := provider.GetDB(ns1, "", coll1)
	require.NoError(t, err)

	db2, err := provider.GetDB(ns1, "", coll1)
	require.NoError(t, err)
	require.Equal(t, db, db2)

	err = db.Put(
		api.NewKeyValue(key1, value1, txID1, time.Now().UTC()),
		api.NewKeyValue(key2, value2, txID2, time.Now().UTC().Add(1*time.Minute)))
	require.NoError(t, err)

	// Wait for the periodic purge
	time.Sleep(200 * time.Millisecond)

	// Check if k1 is delete from db
	v, err := db.Get(key1)
	require.NoError(t, err)
	require.Nil(t, v)

	// Check if k2 is still exist in db
	v, err = db.Get(key2)
	require.NoError(t, err)
	require.NotNil(t, v)
}

func TestGetKeysFromDB(t *testing.T) {
	defer removeDBPath(t)

	provider := NewDBProvider()
	defer provider.Close()

	db1, err := provider.GetDB(ns1, "", coll1)
	require.NoError(t, err)
	require.NotNil(t, db1)

	err = db1.Put(api.NewKeyValue(key1, value1, txID1, time.Now().UTC().Add(1*time.Minute)))
	require.NoError(t, err)

	err = db1.Put(api.NewKeyValue(key2, value2, txID1, time.Now().UTC().Add(1*time.Minute)))
	require.NoError(t, err)

	v, err := db1.Get(key1)
	require.NoError(t, err)
	require.NotNil(t, v)
	require.Equal(t, txID1, v.TxID)
	require.Equal(t, value1, v.Value)

	v, err = db1.Get(key2)
	require.NoError(t, err)
	require.NotNil(t, v)
	require.Equal(t, txID1, v.TxID)
	require.Equal(t, value2, v.Value)

	db2, err := provider.GetDB(ns1, "", coll2)
	require.NoError(t, err)
	require.NotNil(t, db2)

	err = db2.Put(api.NewKeyValue(key1, value2, txID2, time.Now().UTC().Add(1*time.Minute)))
	require.NoError(t, err)

	v, err = db2.Get(key1)
	require.NoError(t, err)
	require.NotNil(t, v)
	require.Equal(t, txID2, v.TxID)
	require.Equal(t, value2, v.Value)

	vals, err := db1.GetMultiple(key1, key2)
	require.NoError(t, err)
	require.Equal(t, 2, len(vals))
	require.Equal(t, value1, vals[0].Value)
	require.Equal(t, value2, vals[1].Value)

	// Delete
	err = db2.Put(&api.KeyValue{Key: key1})
	require.NoError(t, err)
	v, err = db2.Get(key1)
	require.NoError(t, err)
	require.Nil(t, v)

	// Delete again
	err = db2.Put(&api.KeyValue{Key: key1})
	require.NoError(t, err)
	v, err = db2.Get(key1)
	require.NoError(t, err)
	require.Nil(t, v)
}

func TestStore_Query(t *testing.T) {
	defer removeDBPath(t)

	provider := NewDBProvider()
	defer provider.Close()

	db, err := provider.GetDB(ns1, "", coll1)
	require.NoError(t, err)
	require.NotNil(t, db)

	require.PanicsWithValue(t, "not implemented", func() {
		_, err := db.Query("some query")
		require.NoError(t, err)
	})
}

func TestMain(m *testing.M) {
	removeDBPath(nil)
	viper.Set("peer.fileSystemPath", "/tmp/fabric/ledgertests/offledgerdb_89786")
	viper.Set("coll.offledger.cleanupExpired.Interval", "100ms")

	os.Exit(m.Run())
}

func removeDBPath(t testing.TB) {
	removePath(t, config.GetOLCollLevelDBPath())
}

func removePath(t testing.TB, path string) {
	if err := os.RemoveAll(path); err != nil {
		t.Fatalf("Err: %s", err)
	}
}
