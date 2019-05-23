/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pvtdatastore

import (
	"testing"

	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/transientstore"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

const (
	channelID = "testchannel"

	ns1 = "ns1"
	ns2 = "ns2"

	coll1 = "coll1"
	coll2 = "coll2"
	coll3 = "coll3"

	policy1 = "OR('Org1MSP.member','Org2MSP.member')"
)

func TestStore_StorePvtData(t *testing.T) {
	collStore := mocks.NewDataStore()

	t.Run("No transient", func(t *testing.T) {
		tStore := &mockTransientStore{}

		c := New(channelID, tStore, collStore)
		require.NotNil(t, c)

		b := mocks.NewPvtReadWriteSetBuilder()
		b.Namespace(ns1).Collection(coll1).StaticConfig(policy1, 2, 5, 1000)
		pvtData := b.Build()

		err := c.StorePvtData("tx1", pvtData, 1000)
		assert.NoError(t, err)
		require.NotNil(t, tStore.persistedData)
		assert.Equal(t, pvtData.PvtRwset, tStore.persistedData.PvtRwset)
	})

	t.Run("Just transient", func(t *testing.T) {
		tStore := &mockTransientStore{}

		c := New(channelID, tStore, collStore)
		require.NotNil(t, c)

		b := mocks.NewPvtReadWriteSetBuilder()
		b.Namespace(ns1).Collection(coll2).DCASConfig(policy1, 2, 5, "10m")
		pvtData := b.Build()

		err := c.StorePvtData("tx1", pvtData, 1000)
		assert.NoError(t, err)
		assert.Nil(t, tStore.persistedData)
	})

	t.Run("Mixed types", func(t *testing.T) {
		tStore := &mockTransientStore{}

		c := New(channelID, tStore, collStore)
		require.NotNil(t, c)

		b := mocks.NewPvtReadWriteSetBuilder()
		nsb := b.Namespace(ns1)
		nsb.Collection(coll1).StaticConfig(policy1, 2, 5, 1000)
		nsb.Collection(coll2).TransientConfig(policy1, 2, 5, "10m")
		pvtData := b.Build()

		err := c.StorePvtData("tx1", pvtData, 1000)
		assert.NoError(t, err)
		require.NotNil(t, tStore.persistedData)
		require.NotNil(t, tStore.persistedData.PvtRwset)
		assert.NotEqual(t, pvtData.PvtRwset, tStore.persistedData.PvtRwset)

		// The original collections set should have two entries: DCAS and regular private data
		require.Equal(t, 1, len(pvtData.PvtRwset.NsPvtRwset))
		assert.Equal(t, 2, len(pvtData.PvtRwset.NsPvtRwset[0].CollectionPvtRwset))

		// The rewritten collections set should have only one entry: regular private data
		require.Equal(t, 1, len(tStore.persistedData.PvtRwset.NsPvtRwset))
		assert.Equal(t, 1, len(tStore.persistedData.PvtRwset.NsPvtRwset[0].CollectionPvtRwset))
	})
}

func TestStore_StorePvtData_error(t *testing.T) {
	t.Run("Marshal error", func(t *testing.T) {
		tStore := &mockTransientStore{}
		collStore := mocks.NewDataStore()

		c := New(channelID, tStore, collStore)
		require.NotNil(t, c)

		b := mocks.NewPvtReadWriteSetBuilder()
		b.Namespace(ns1).Collection(coll1).StaticConfig(policy1, 2, 5, 1000).WithMarshalError()
		pvtData := b.Build()

		err := c.StorePvtData("tx1", pvtData, 1000)
		assert.Error(t, err)
		assert.Nil(t, tStore.persistedData)
	})

	t.Run("CollStore error", func(t *testing.T) {
		tStore := &mockTransientStore{}
		expectedErr := errors.New("test coll error")
		collStore := mocks.NewDataStore().Error(expectedErr)

		c := New(channelID, tStore, collStore)
		require.NotNil(t, c)

		b := mocks.NewPvtReadWriteSetBuilder()
		b.Namespace(ns1).Collection(coll1).StaticConfig(policy1, 2, 5, 1000)
		pvtData := b.Build()

		err := c.StorePvtData("tx1", pvtData, 1000)
		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
	})
}

func TestIsPvtData(t *testing.T) {
	pvtColl := &common.CollectionConfig_StaticCollectionConfig{
		StaticCollectionConfig: &common.StaticCollectionConfig{
			Name: coll1,
		},
	}
	transientColl := &common.CollectionConfig_StaticCollectionConfig{
		StaticCollectionConfig: &common.StaticCollectionConfig{
			Name: coll2,
			Type: common.CollectionType_COL_TRANSIENT,
		},
	}

	collConfigs := make(map[string]*common.CollectionConfigPackage)
	collConfigs[ns1] = &common.CollectionConfigPackage{
		Config: []*common.CollectionConfig{
			{Payload: pvtColl},
			{Payload: transientColl},
		},
	}

	f := newPvtRWSetFilter(channelID, "tx1", 1000, &transientstore.TxPvtReadWriteSetWithConfigInfo{CollectionConfigs: collConfigs})

	ok, err := f.isPvtData(ns1, coll1)
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = f.isPvtData(ns1, coll2)
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, err = f.isPvtData(ns2, coll1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not find collection configs for namespace")
	assert.False(t, ok)

	ok, err = f.isPvtData(ns1, coll3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not find collection config for collection")
	assert.False(t, ok)
}

type mockTransientStore struct {
	persistedData *transientstore.TxPvtReadWriteSetWithConfigInfo
}

func (m *mockTransientStore) PersistWithConfig(txid string, blockHeight uint64, privateSimulationResultsWithConfig *transientstore.TxPvtReadWriteSetWithConfigInfo) error {
	m.persistedData = privateSimulationResultsWithConfig
	return nil
}
