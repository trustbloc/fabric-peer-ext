/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package couchdbstore

import (
	"encoding/json"
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

func TestCreateCouchInstance(t *testing.T) {
	provider := NewDBProvider()
	defer provider.Close()

	ci, err := provider.createCouchInstance()
	require.NoError(t, err)
	require.NotNil(t, ci)

	ci2, err := provider.createCouchInstance()
	require.NoError(t, err)
	require.True(t, ci == ci2)
}

func TestGetKeysFromDB(t *testing.T) {
	provider := NewDBProvider()
	defer provider.Close()

	db1, err := provider.GetDB("testchannel", ns1, coll1)
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

	db2, err := provider.GetDB("testchannel", coll2, ns1)
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

type testValue struct {
	Field1 string
	Field2 int
}

type invalidTestValue1 struct {
	ID string `json:"_id"`
}

type invalidTestValue2 struct {
	TxID string `json:"~txnID"`
}

func TestJSONValues(t *testing.T) {
	provider := NewDBProvider()
	defer provider.Close()

	db1, err := provider.GetDB("testchannel", ns1, coll1)
	require.NoError(t, err)
	require.NotNil(t, db1)

	t.Run("Valid JSON", func(t *testing.T) {
		v1 := &testValue{
			Field1: "value1",
			Field2: 12345,
		}

		v1Bytes, err := json.Marshal(v1)
		require.NoError(t, err)

		err = db1.Put(api.NewKeyValue(key1, v1Bytes, txID1, time.Now().UTC().Add(1*time.Minute)))
		require.NoError(t, err)

		val1, err := db1.Get(key1)
		require.NoError(t, err)
		require.NotNil(t, val1)
		require.Equal(t, txID1, val1.TxID)

		rv1 := &testValue{}
		err = json.Unmarshal(val1.Value, rv1)
		require.NoError(t, err)
		require.Equal(t, v1, rv1)
		require.NotEmpty(t, val1.Revision)

		// Update the value for key1
		v2 := *v1
		v2.Field1 = "new value"
		v2Bytes, err := json.Marshal(v2)
		require.NoError(t, err)

		val1.Value = v2Bytes
		err = db1.Put(&api.KeyValue{Key: key1, Value: val1})
		require.NoError(t, err)

		val2, err := db1.Get(key1)
		require.NoError(t, err)
		require.NotNil(t, val2)
		require.Equal(t, txID1, val2.TxID)
		require.NotEmpty(t, val2.Revision)
		require.NotEqual(t, val1.Revision, val2.Revision)

		rv2 := &testValue{}
		err = json.Unmarshal(val2.Value, rv2)
		require.NoError(t, err)
		require.Equal(t, &v2, rv2)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		v1 := &invalidTestValue1{
			ID: "some_id",
		}

		v1Bytes, err := json.Marshal(v1)
		require.NoError(t, err)

		err = db1.Put(api.NewKeyValue(key1, v1Bytes, txID1, time.Now().UTC().Add(1*time.Minute)))
		require.EqualError(t, err, "field [_id] is not valid for the CouchDB state database")

		v2 := &invalidTestValue2{
			TxID: "some_tx_id",
		}

		v2Bytes, err := json.Marshal(v2)
		require.NoError(t, err)

		err = db1.Put(api.NewKeyValue(key1, v2Bytes, txID1, time.Now().UTC().Add(1*time.Minute)))
		require.EqualError(t, err, "field [~txnID] is not valid for the CouchDB state database")
	})
}

func TestDeleteExpiredKeysFromDB(t *testing.T) {
	provider := NewDBProvider()
	defer provider.Close()

	db, err := provider.GetDB("testchannel", coll3, ns1)
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
