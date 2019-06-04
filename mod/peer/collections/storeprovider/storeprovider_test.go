/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package storeprovider

import (
	"testing"

	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	spmocks "github.com/hyperledger/fabric/extensions/collections/storeprovider/mocks"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/mod/peer/testutil"
	extstoreprovider "github.com/trustbloc/fabric-peer-ext/pkg/collections/storeprovider"
	tdapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

func TestStoreProvider(t *testing.T) {
	tdataProvider := spmocks.NewTransientDataStoreProvider()

	extstoreprovider.SetNewTransientDataProvider(func() tdapi.StoreProvider {
		return tdataProvider
	})

	t.Run("OpenStore - success", func(t *testing.T) {
		p := NewProviderFactory(testutil.TestLedgerConf())
		require.NotNil(t, p)

		s, err := p.OpenStore("testchannel")
		require.NoError(t, err)
		require.NotNil(t, s)

		s2 := p.StoreForChannel("testchannel")
		require.Equal(t, s, s2)
	})

	t.Run("OpenStore - transient data error", func(t *testing.T) {
		p := NewProviderFactory(testutil.TestLedgerConf())
		require.NotNil(t, p)

		expectedErr := errors.New("transientdata error")
		tdataProvider.Error(expectedErr)
		defer tdataProvider.Error(nil)

		s, err := p.OpenStore("testchannel")
		assert.EqualError(t, err, expectedErr.Error())
		require.Nil(t, s)
	})
}

func TestStore_PutAndGetData(t *testing.T) {
	const (
		tx1   = "tx1"
		ns1   = "ns1"
		coll1 = "coll1"
		key1  = "key1"
		key2  = "key2"
	)

	k1 := storeapi.NewKey(tx1, ns1, coll1, key1)
	k2 := storeapi.NewKey(tx1, ns1, coll1, key2)

	v1 := &storeapi.ExpiringValue{Value: []byte("value1")}
	v2 := &storeapi.ExpiringValue{Value: []byte("value1")}

	tdataProvider := spmocks.NewTransientDataStoreProvider()

	extstoreprovider.SetNewTransientDataProvider(func() tdapi.StoreProvider {
		return tdataProvider.Data(k1, v1).Data(k2, v2)
	})

	p := NewProviderFactory(testutil.TestLedgerConf())
	require.NotNil(t, p)

	s, err := p.OpenStore("testchannel")
	require.NoError(t, err)
	require.NotNil(t, s)

	t.Run("GetTransientData", func(t *testing.T) {
		value, err := s.GetTransientData(k1)
		require.NoError(t, err)
		require.NotNil(t, value)

		values, err := s.GetTransientDataMultipleKeys(storeapi.NewMultiKey(tx1, ns1, coll1, key1, key2))
		require.NoError(t, err)
		assert.Equal(t, 2, len(values))
	})

	t.Run("Persist", func(t *testing.T) {
		err := s.Persist(tx1, mocks.NewPvtReadWriteSetBuilder().Build())
		assert.NoError(t, err)

		expectedErr := errors.New("transient data error")
		tdataProvider.StoreError(expectedErr)
		err = s.Persist(tx1, mocks.NewPvtReadWriteSetBuilder().Build())
		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
		tdataProvider.StoreError(nil)
	})
}
