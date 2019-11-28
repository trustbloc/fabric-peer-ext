/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package storeprovider

import (
	"os"
	"testing"

	"github.com/hyperledger/fabric/common/ledger/util/leveldbhelper"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/storeprovider/store/dbstore"
	"github.com/trustbloc/fabric-peer-ext/pkg/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

func TestNewError(t *testing.T) {
	errExpected := errors.New("injected error")
	restoreDBCreator := dbstore.SetLevelDBCreator(func(dbPath string) (provider *leveldbhelper.Provider, e error) {
		return nil, errExpected
	})
	defer restoreDBCreator()

	require.PanicsWithValue(t, errExpected, func() {
		New(nil, nil)
	})
}

func TestStoreProvider(t *testing.T) {
	defer removeDBPath(t)
	channel1 := "channel1"
	channel2 := "channel2"

	gossipProvider := &mocks.GossipProvider{}
	gossipProvider.GetGossipServiceReturns(mocks.NewMockGossipAdapter())

	p := New(gossipProvider, &mocks.IdentityDeserializerProvider{})
	require.NotNil(t, p)

	s1, err := p.OpenStore(channel1)
	require.NotNil(t, s1)
	assert.NoError(t, err)

	_, err = p.OpenStore(channel1)
	assert.Error(t, err)
	assert.Equal(t, s1, p.StoreForChannel(channel1))
	removeDBPath(t)

	s2, err := p.OpenStore(channel2)
	require.NotNil(t, s2)
	assert.NoError(t, err)
	assert.NotEqual(t, s1, s2)
	assert.Equal(t, s2, p.StoreForChannel(channel2))

	assert.NotPanics(t, func() {
		p.Close()
	})
}

func TestMain(m *testing.M) {
	removeDBPath(nil)
	viper.Set("peer.fileSystemPath", "/tmp/fabric/ledgertests/transientdatadb")
	os.Exit(m.Run())
}

func removeDBPath(t testing.TB) {
	removePath(t, config.GetTransientDataLevelDBPath())
}

func removePath(t testing.TB, path string) {
	if err := os.RemoveAll(path); err != nil {
		t.Fatalf("Err: %s", err)
		t.FailNow()
	}
}
