/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package dbstore

import (
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/storeprovider/store/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/config"
)

const (
	txID1 = "txid1"
	txID2 = "txid2"
	ns1   = "namespace1"
	ns2   = "namespace2"
	coll2 = "coll2"
	coll1 = "coll1"
	key1  = "key1"
	key2  = "key2"
)

var (
	k1 = api.Key{
		Namespace:  ns1,
		Collection: coll1,
		Key:        key1,
	}
	v1 = &api.Value{TxID: txID1, Value: value1, ExpiryTime: time.Now().UTC()}

	k2 = api.Key{
		Namespace:  ns2,
		Collection: coll2,
		Key:        key2,
	}
	v2 = &api.Value{TxID: txID2, Value: value2, ExpiryTime: time.Now().UTC().Add(1 * time.Minute)}

	value1 = []byte("v1")
	value2 = []byte("v2")
)

func TestDeleteExpiredKeysFromDB(t *testing.T) {
	removeDBPath(t)
	defer removeDBPath(t)

	p := NewDBProvider()
	db, err := p.OpenDBStore("testchannel")
	require.NoError(t, err)
	defer p.Close()

	err = db.AddKey(k1, v1)
	require.Nil(t, err)

	err = db.AddKey(k2, v2)
	require.Nil(t, err)

	// delete expired keys
	require.NoError(t, db.DeleteExpiredKeys())

	// Check if k1 is delete from db
	v, err := db.GetKey(k1)
	require.Nil(t, err)
	require.Nil(t, v)

	// Check if k2 is still exist in db
	v, err = db.GetKey(k2)
	require.Nil(t, err)
	require.NotNil(t, v)

}

func TestAddRetrieveKeysFromDB(t *testing.T) {
	defer removeDBPath(t)
	p := NewDBProvider()
	db, err := p.OpenDBStore("testchannel")
	require.NoError(t, err)
	defer p.Close()

	err = db.AddKey(k1, v1)
	require.Nil(t, err)

	v, err := db.GetKey(k1)
	require.Nil(t, err)
	require.NotNil(t, v)
	require.Equal(t, txID1, v.TxID)
	require.Equal(t, value1, v.Value)

}

func TestMain(m *testing.M) {
	removeDBPath(nil)
	viper.Set("peer.fileSystemPath", "/tmp/fabric/ledgertests/transientdatadb")
	viper.Set("ledger.transientdata.cleanupExpired.Interval", "100ms")

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
