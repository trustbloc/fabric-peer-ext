/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dcasclient

import (
	"errors"
	"testing"

	"github.com/btcsuite/btcutil/base58"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	gmocks "github.com/hyperledger/fabric/extensions/gossip/mocks"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/transientstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	olclient "github.com/trustbloc/fabric-peer-ext/pkg/collections/client"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/dcas"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

const (
	channelID   = "testchannel"
	ns1         = "ns1"
	coll1       = "coll1"
	blockHeight = uint64(1000)
)

func TestDCASClient_Put(t *testing.T) {
	ledger := &mocks.Ledger{
		TxSimulator: &mocks.TxSimulator{
			SimulationResults: &ledger.TxSimulationResults{},
		},
		BlockchainInfo: &cb.BlockchainInfo{
			Height: blockHeight,
		},
	}
	gossip := &mockGossipAdapter{}
	configRetriever := &mockCollectionConfigRetriever{}
	var creatorError error

	// Mock out all of the dependencies
	olclient.SetLedgerProvider(func(channelID string) olclient.PeerLedger { return ledger })
	olclient.SetGossipProvider(func() olclient.GossipAdapter { return gossip })
	olclient.SetBlockPublisherProvider(func(channelID string) gossipapi.BlockPublisher { return gmocks.NewBlockPublisher() })
	olclient.SetCollConfigRetrieverProvider(func(_ string, _ olclient.PeerLedger, _ gossipapi.BlockPublisher) olclient.CollectionConfigRetriever {
		return configRetriever
	})
	olclient.SetCreatorProvider(func() ([]byte, error) { return []byte("creator"), creatorError })

	c := New(channelID)
	require.NotNil(t, c)

	value1 := []byte("value1")

	jsonValue := []byte(`{"fieldx":"valuex","field1":"value1","field2":"value2"}`)

	t.Run("Non-JSON -> Success", func(t *testing.T) {
		key, err := c.Put(ns1, coll1, value1)
		require.NoError(t, err)
		casKey, _, err := dcas.GetCASKeyAndValue(value1)
		require.NoError(t, err)
		assert.Equal(t, casKey, key)
	})

	t.Run("JSON -> Success", func(t *testing.T) {
		key, err := c.Put(ns1, coll1, jsonValue)
		require.NoError(t, err)
		casKey, _, err := dcas.GetCASKeyAndValue(jsonValue)
		require.NoError(t, err)
		assert.Equal(t, casKey, key)
	})

	t.Run("JSON marshal error -> error", func(t *testing.T) {
		reset := dcas.SetJSONMarshaller(func(m map[string]interface{}) ([]byte, error) {
			return nil, errors.New("injected marshal error")
		})
		defer reset()

		_, err := c.Put(ns1, coll1, jsonValue)
		require.Error(t, err)
	})

	t.Run("Delete", func(t *testing.T) {
		casKey, _, err := dcas.GetCASKeyAndValue(value1)
		require.NoError(t, err)
		require.NoError(t, c.Delete(ns1, coll1, casKey))
	})
}

func TestDCASClient_Get(t *testing.T) {
	key1, value1, err := dcas.GetCASKeyAndValue([]byte("value1"))
	require.NoError(t, err)
	key2, value2, err := dcas.GetCASKeyAndValue([]byte("value2"))
	require.NoError(t, err)

	ledger := &mocks.Ledger{
		QueryExecutor: mocks.NewQueryExecutor().
			WithPrivateState(ns1, coll1, base58.Encode([]byte(key1)), value1).
			WithPrivateState(ns1, coll1, base58.Encode([]byte(key2)), value2),
	}

	gossip := &mockGossipAdapter{}
	configRetriever := &mockCollectionConfigRetriever{}
	var creatorError error

	// Mock out all of the dependencies
	olclient.SetLedgerProvider(func(channelID string) olclient.PeerLedger { return ledger })
	olclient.SetGossipProvider(func() olclient.GossipAdapter { return gossip })
	olclient.SetBlockPublisherProvider(func(channelID string) gossipapi.BlockPublisher { return gmocks.NewBlockPublisher() })
	olclient.SetCollConfigRetrieverProvider(func(_ string, _ olclient.PeerLedger, _ gossipapi.BlockPublisher) olclient.CollectionConfigRetriever {
		return configRetriever
	})
	olclient.SetCreatorProvider(func() ([]byte, error) { return []byte("creator"), creatorError })

	c := New(channelID)
	require.NotNil(t, c)

	t.Run("Get - success", func(t *testing.T) {
		value, err := c.Get(ns1, coll1, key1)
		require.NoError(t, err)
		assert.Equal(t, value1, value)
	})

	t.Run("GetMultipleKeys - success", func(t *testing.T) {
		values, err := c.GetMultipleKeys(ns1, coll1, key1, key2)
		require.NoError(t, err)
		require.Equal(t, 2, len(values))
		assert.Equal(t, value1, values[0])
		assert.Equal(t, value2, values[1])
	})
}

func TestClient_Query(t *testing.T) {
	key1, value1, err := dcas.GetCASKeyAndValue([]byte(`{"id":"id1","value":"value1"}`))
	require.NoError(t, err)
	key2, value2, err := dcas.GetCASKeyAndValue([]byte(`{"id":"id2","value":"value2"}`))
	require.NoError(t, err)

	query1 := "query1"

	vk1 := &statedb.VersionedKV{
		CompositeKey: statedb.CompositeKey{
			Namespace: ns1 + "~" + coll1,
			Key:       base58.Encode([]byte(key1)),
		},
		VersionedValue: statedb.VersionedValue{
			Value: value1,
		},
	}
	vk2 := &statedb.VersionedKV{
		CompositeKey: statedb.CompositeKey{
			Namespace: ns1 + "~" + coll1,
			Key:       base58.Encode([]byte(key2)),
		},
		VersionedValue: statedb.VersionedValue{
			Value: value2,
		},
	}

	mockLedger := &mocks.Ledger{
		QueryExecutor: mocks.NewQueryExecutor().
			WithPrivateQueryResults(ns1, coll1, query1, []*statedb.VersionedKV{vk1, vk2}),
	}

	gossip := &mockGossipAdapter{}
	configRetriever := &mockCollectionConfigRetriever{}
	var creatorError error

	// Mock out all of the dependencies
	olclient.SetLedgerProvider(func(channelID string) olclient.PeerLedger { return mockLedger })
	olclient.SetGossipProvider(func() olclient.GossipAdapter { return gossip })
	olclient.SetBlockPublisherProvider(func(channelID string) gossipapi.BlockPublisher { return gmocks.NewBlockPublisher() })
	olclient.SetCollConfigRetrieverProvider(func(_ string, _ olclient.PeerLedger, _ gossipapi.BlockPublisher) olclient.CollectionConfigRetriever {
		return configRetriever
	})
	olclient.SetCreatorProvider(func() ([]byte, error) { return []byte("creator"), creatorError })

	c := New(channelID)
	require.NotNil(t, c)

	t.Run("Query - success", func(t *testing.T) {
		it, err := c.Query(ns1, coll1, query1)
		require.NoError(t, err)
		require.NotNil(t, it)

		next, err := it.Next()
		require.NoError(t, err)
		kv, ok := next.(*statedb.VersionedKV)
		require.True(t, ok)
		require.Equal(t, vk1.Namespace, kv.Namespace)
		require.Equal(t, key1, kv.Key)

		next, err = it.Next()
		require.NoError(t, err)
		kv, ok = next.(*statedb.VersionedKV)
		require.True(t, ok)
		require.Equal(t, vk2.Namespace, kv.Namespace)
		require.Equal(t, key2, kv.Key)

		next, err = it.Next()
		require.NoError(t, err)
		require.Nil(t, next)

		require.NotPanics(t, func() {
			it.Close()
		})
	})

	t.Run("Query error", func(t *testing.T) {
		mockLedger.QueryExecutor.WithError(errors.New("injected query error"))
		defer func() { mockLedger.QueryExecutor.WithError(nil) }()

		it, err := c.Query(ns1, coll1, query1)
		require.Error(t, err)
		require.Nil(t, it)
	})

	t.Run("Iterator error", func(t *testing.T) {
		mockLedger.QueryExecutor.WithIteratorProvider(func() *mocks.ResultsIterator {
			return mocks.NewResultsIterator().WithError(errors.New("injected iterator error"))
		})
		defer func() {
			// Reset the iterator
			mockLedger.QueryExecutor.WithIteratorProvider(func() *mocks.ResultsIterator {
				return mocks.NewResultsIterator()
			})
		}()

		it, err := c.Query(ns1, coll1, query1)
		require.NoError(t, err)
		require.NotNil(t, it)

		next, err := it.Next()
		require.Error(t, err)
		require.Nil(t, next)
	})
}

type mockCollectionConfigRetriever struct {
	Error error
}

func (m *mockCollectionConfigRetriever) Config(ns, coll string) (*cb.StaticCollectionConfig, error) {
	return &cb.StaticCollectionConfig{}, m.Error
}

type mockGossipAdapter struct {
	Error error
}

func (m *mockGossipAdapter) DistributePrivateData(chainID string, txID string, privateData *transientstore.TxPvtReadWriteSetWithConfigInfo, blkHt uint64) error {
	return m.Error
}
