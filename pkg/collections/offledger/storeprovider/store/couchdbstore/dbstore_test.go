/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package couchdbstore

import (
	"os"
	"testing"
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/extensions/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/storeprovider/store/api"
)

const (
	txID1 = "txid1"
	txID2 = "txid2"
	ns1   = "namespace1"
	coll2 = "coll2"
	coll1 = "coll1"
	coll3 = "coll3"
	key1  = "key1"
	key2  = "key2"
	key3  = "key3"
	key4  = "key4"
)

var (
	value1 = []byte("v111")
	value2 = []byte("v222")
)

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}

func testMain(m *testing.M) int {
	// CouchDB configuration
	_, _, stop := testutil.SetupExtTestEnv()

	//set the logging level to DEBUG to test debug only code
	flogging.ActivateSpec("couchdb=debug")

	viper.Set("coll.offledger.cleanupExpired.Interval", "500ms")

	//run the tests
	code := m.Run()

	//stop couchdb
	stop()

	return code
}

func TestGetKeysFromDB(t *testing.T) {
	provider := NewDBProvider()
	defer provider.Close()

	db1, err := provider.GetDB(ns1, coll1)
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

	db2, err := provider.GetDB(ns1, coll2)
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

func TestDeleteExpiredKeysFromDB(t *testing.T) {
	provider := NewDBProvider()
	defer provider.Close()

	db, err := provider.GetDB(ns1, coll3)
	require.NoError(t, err)

	err = db.Put(
		api.NewKeyValue(key3, value1, txID1, time.Now().UTC().Add(-1*time.Minute)),
		api.NewKeyValue(key4, value2, txID2, time.Now().UTC().Add(1*time.Minute)))
	require.NoError(t, err)

	// Wait for the periodic purge
	time.Sleep(1 * time.Second)

	// Check if key is deleted from db
	v, err := db.Get(key3)
	require.NoError(t, err)
	require.Nil(t, v)

	// Check if k2 is still exist in db
	v, err = db.Get(key4)
	require.NoError(t, err)
	require.NotNil(t, v)
}
