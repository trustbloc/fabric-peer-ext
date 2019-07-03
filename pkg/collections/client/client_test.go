/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"testing"

	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/transientstore"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

const (
	channelID   = "testchannel"
	ns1         = "ns1"
	coll1       = "coll1"
	key1        = "key1"
	key2        = "key2"
	blockHeight = uint64(1000)
)

func TestClient_Put(t *testing.T) {
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
	getLedger = func(channelID string) PeerLedger { return ledger }
	getGossipAdapter = func() GossipAdapter { return gossip }
	getBlockPublisher = func(channelID string) gossipapi.BlockPublisher { return mocks.NewBlockPublisher() }
	getCollConfigRetriever = func(_ string, _ PeerLedger, _ gossipapi.BlockPublisher) CollectionConfigRetriever {
		return configRetriever
	}
	newCreator = func() ([]byte, error) { return []byte("creator"), creatorError }

	c := New(channelID)
	require.NotNil(t, c)

	value1 := []byte("value1")

	t.Run("TxID error", func(t *testing.T) {
		creatorError = errors.New("mock creator error")
		defer func() { creatorError = nil }()

		err := c.Put(ns1, coll1, key1, value1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error generating transaction ID")
	})

	t.Run("Success", func(t *testing.T) {
		err := c.Put(ns1, coll1, key1, value1)
		require.NoError(t, err)
	})

	t.Run("Simulation results error", func(t *testing.T) {
		ledger.TxSimulator.SimError = errors.New("mock TxSimulator error")
		defer func() { ledger.TxSimulator.SimError = nil }()

		err := c.Put(ns1, coll1, key1, value1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error generating simulation results")
	})

	t.Run("GetTxSimulator error", func(t *testing.T) {
		ledger.Error = errors.New("mock TxSimulator error")
		defer func() { ledger.Error = nil }()

		err := c.Put(ns1, coll1, key1, value1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error getting TxSimulator")
	})

	t.Run("TxSimulator - Put error", func(t *testing.T) {
		ledger.TxSimulator.Error = errors.New("mock TxSimulator error")
		defer func() { ledger.TxSimulator.Error = nil }()

		err := c.Put(ns1, coll1, key1, value1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error setting keys")
	})

	t.Run("Gossip error", func(t *testing.T) {
		gossip.Error = errors.New("mock gossip error")
		defer func() { gossip.Error = nil }()

		err := c.Put(ns1, coll1, key1, value1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error distributing private data")
	})

	t.Run("GetBlockchainInfo error", func(t *testing.T) {
		ledger.BcInfoError = errors.New("mock ledger error")
		defer func() { ledger.BcInfoError = nil }()

		err := c.Put(ns1, coll1, key1, value1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error getting blockchain info")
	})

	t.Run("CollectionConfig error", func(t *testing.T) {
		configRetriever.Error = errors.New("mock config error")
		defer func() { configRetriever.Error = nil }()

		err := c.Put(ns1, coll1, key1, value1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error getting collection config")
	})

	t.Run("Delete", func(t *testing.T) {
		err := c.Delete(ns1, coll1, key1, key2)
		require.NoError(t, err)
	})
}

func TestClient_Get(t *testing.T) {
	value1 := []byte("value1")
	value2 := []byte("value2")

	ledger := &mocks.Ledger{
		QueryExecutor: mocks.NewQueryExecutor().
			WithPrivateState(ns1, coll1, key1, value1).
			WithPrivateState(ns1, coll1, key2, value2),
	}

	gossip := &mockGossipAdapter{}
	configRetriever := &mockCollectionConfigRetriever{}
	var creatorError error

	// Mock out all of the dependencies
	getLedger = func(channelID string) PeerLedger { return ledger }
	getGossipAdapter = func() GossipAdapter { return gossip }
	getBlockPublisher = func(channelID string) gossipapi.BlockPublisher { return mocks.NewBlockPublisher() }
	getCollConfigRetriever = func(_ string, _ PeerLedger, _ gossipapi.BlockPublisher) CollectionConfigRetriever {
		return configRetriever
	}
	newCreator = func() ([]byte, error) { return []byte("creator"), creatorError }

	c := New(channelID)
	require.NotNil(t, c)

	t.Run("Get - success", func(t *testing.T) {
		value, err := c.Get(ns1, coll1, key1)
		require.NoError(t, err)
		assert.Equal(t, value1, value)
	})

	t.Run("Get - error", func(t *testing.T) {
		ledger.Error = errors.New("mock QueryExecutor error")
		defer func() { ledger.Error = nil }()

		_, err := c.Get(ns1, coll1, key1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error getting QueryExecutor")
	})

	t.Run("GetMultipleKeys - success", func(t *testing.T) {
		values, err := c.GetMultipleKeys(ns1, coll1, key1, key2)
		require.NoError(t, err)
		require.Equal(t, 2, len(values))
		assert.Equal(t, value1, values[0])
		assert.Equal(t, value2, values[1])
	})

	t.Run("GetMultipleKeys - error", func(t *testing.T) {
		ledger.Error = errors.New("mock QueryExecutor error")
		defer func() { ledger.Error = nil }()

		_, err := c.GetMultipleKeys(ns1, coll1, key1, key2)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error getting QueryExecutor")
	})
}

func TestClient_Query(t *testing.T) {
	query1 := "query1"

	vk1 := &statedb.VersionedKV{
		CompositeKey: statedb.CompositeKey{
			Namespace: ns1 + "~" + coll1,
			Key:       key1,
		},
		VersionedValue: statedb.VersionedValue{
			Value: []byte("v1_1"),
		},
	}
	vk2 := &statedb.VersionedKV{
		CompositeKey: statedb.CompositeKey{
			Namespace: ns1 + "~" + coll1,
			Key:       key2,
		},
		VersionedValue: statedb.VersionedValue{
			Value: []byte("v1_2"),
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
	getLedger = func(channelID string) PeerLedger { return mockLedger }
	getGossipAdapter = func() GossipAdapter { return gossip }
	getBlockPublisher = func(channelID string) gossipapi.BlockPublisher { return mocks.NewBlockPublisher() }
	getCollConfigRetriever = func(_ string, _ PeerLedger, _ gossipapi.BlockPublisher) CollectionConfigRetriever {
		return configRetriever
	}
	newCreator = func() ([]byte, error) { return []byte("creator"), creatorError }

	c := New(channelID)
	require.NotNil(t, c)

	t.Run("Query - success", func(t *testing.T) {
		it, err := c.Query(ns1, coll1, query1)
		require.NoError(t, err)
		require.NotNil(t, it)

		next, err := it.Next()
		require.NoError(t, err)
		require.Equal(t, vk1, next)

		next, err = it.Next()
		require.NoError(t, err)
		require.Equal(t, vk2, next)

		next, err = it.Next()
		require.NoError(t, err)
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
