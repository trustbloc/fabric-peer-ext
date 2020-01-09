/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package storeprovider

import (
	"os"
	"reflect"
	"testing"

	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/common/flogging"
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	"github.com/hyperledger/fabric/extensions/testutil"
	viper "github.com/spf13/viper2015"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

func TestStoreProvider_OpenStore(t *testing.T) {
	channel1 := "channel1"
	channel2 := "channel2"

	f := New(&mocks.IdentifierProvider{}, &mocks.IdentityDeserializerProvider{})
	require.NotNil(t, f)

	s1, err := f.OpenStore(channel1)
	require.NotNil(t, s1)
	assert.NoError(t, err)
	assert.Equal(t, "*storeprovider.store", reflect.TypeOf(s1).String())
	assert.Equal(t, s1, f.StoreForChannel(channel1))

	s2, err := f.OpenStore(channel2)
	require.NotNil(t, s2)
	assert.NoError(t, err)
	assert.NotEqual(t, s1, s2)
	assert.Equal(t, s2, f.StoreForChannel(channel2))

	assert.NotPanics(t, func() {
		f.Close()
	})
}

func TestStoreProvider_WithDecorator(t *testing.T) {
	f := New(
		&mocks.IdentifierProvider{}, &mocks.IdentityDeserializerProvider{},
		WithCollectionType(
			pb.CollectionType_COL_DCAS,
			WithDecorator(&mockDecorator{}),
		))
	require.NotNil(t, f)
	config, ok := f.collConfigs[pb.CollectionType_COL_DCAS]
	assert.True(t, ok)
	assert.NotNil(t, config)
}

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}

type mockDecorator struct {
}

func (d *mockDecorator) BeforeSave(key *storeapi.Key, value *storeapi.ExpiringValue) (*storeapi.Key, *storeapi.ExpiringValue, error) {
	return key, value, nil
}

func (d *mockDecorator) BeforeLoad(key *storeapi.Key) (*storeapi.Key, error) {
	return key, nil
}

func testMain(m *testing.M) int {
	_, _, stop := testutil.SetupExtTestEnv()

	//set the logging level to DEBUG to test debug only code
	flogging.ActivateSpec("couchdb=debug")

	viper.Set("coll.offledger.cleanupExpired.Interval", "500ms")
	viper.Set("coll.offledger.cache.enable", "true")

	//run the tests
	code := m.Run()

	//stop couchdb
	stop()

	return code
}
