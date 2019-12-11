/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"testing"

	cb "github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clientmocks "github.com/trustbloc/fabric-peer-ext/pkg/collections/client/mocks"
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

	distributor := &clientmocks.PvtDataDistributor{}
	configRetriever := mocks.NewCollectionConfigRetriever().WithCollectionConfig(&pb.StaticCollectionConfig{Name: coll1})

	// Mock out all of the dependencies
	signingIdentity := &mocks.SigningIdentity{}
	signingIdentity.SerializeReturns([]byte("creator"), nil)

	identityProvider := &mocks.IdentityProvider{}
	identityProvider.GetDefaultSigningIdentityReturns(signingIdentity, nil)

	providers := &ChannelProviders{
		Ledger:           ledger,
		Distributor:      distributor,
		ConfigRetriever:  configRetriever,
		IdentityProvider: identityProvider,
	}
	c := New(channelID, providers)
	require.NotNil(t, c)

	value1 := []byte("value1")

	t.Run("TxID error", func(t *testing.T) {
		creatorError := errors.New("mock creator error")
		signingIdentity.SerializeReturns(nil, creatorError)
		defer signingIdentity.SerializeReturns([]byte("creator"), nil)

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

	t.Run("Distributor error", func(t *testing.T) {
		distributor.DistributePrivateDataReturns(errors.New("mock distributor error"))
		defer distributor.DistributePrivateDataReturns(nil)

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
		configRetriever.WithError(errors.New("mock config error"))
		defer configRetriever.WithError(nil)

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

	distributor := &clientmocks.PvtDataDistributor{}
	configRetriever := &mocks.CollectionConfigRetriever{}
	var creatorError error

	// Mock out all of the dependencies
	signingIdentity := &mocks.SigningIdentity{}
	signingIdentity.SerializeReturns([]byte("creator"), creatorError)

	identityProvider := &mocks.IdentityProvider{}
	identityProvider.GetDefaultSigningIdentityReturns(signingIdentity, nil)

	providers := &ChannelProviders{
		Ledger:           ledger,
		Distributor:      distributor,
		ConfigRetriever:  configRetriever,
		IdentityProvider: identityProvider,
	}
	c := New(channelID, providers)
	require.NotNil(t, c)
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

	vk1 := &queryresult.KV{
		Namespace: ns1 + "~" + coll1,
		Key:       key1,
		Value:     []byte("v1_1"),
	}
	vk2 := &queryresult.KV{
		Namespace: ns1 + "~" + coll1,
		Key:       key2,
		Value:     []byte("v1_2"),
	}

	mockLedger := &mocks.Ledger{
		QueryExecutor: mocks.NewQueryExecutor().
			WithPrivateQueryResults(ns1, coll1, query1, []*queryresult.KV{vk1, vk2}),
	}

	distributor := &clientmocks.PvtDataDistributor{}
	configRetriever := &mocks.CollectionConfigRetriever{}

	var creatorError error

	// Mock out all of the dependencies
	signingIdentity := &mocks.SigningIdentity{}
	signingIdentity.SerializeReturns([]byte("creator"), creatorError)

	identityProvider := &mocks.IdentityProvider{}
	identityProvider.GetDefaultSigningIdentityReturns(signingIdentity, nil)

	providers := &ChannelProviders{
		Ledger:           mockLedger,
		Distributor:      distributor,
		ConfigRetriever:  configRetriever,
		IdentityProvider: identityProvider,
	}
	c := New(channelID, providers)
	require.NotNil(t, c)
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
