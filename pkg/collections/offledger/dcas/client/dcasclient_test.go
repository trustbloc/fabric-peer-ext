/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"bytes"
	"errors"
	"testing"

	"github.com/btcsuite/btcutil/base58"
	cb "github.com/hyperledger/fabric-protos-go/common"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	coreledger "github.com/hyperledger/fabric/core/ledger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	olclient "github.com/trustbloc/fabric-peer-ext/pkg/collections/client"
	clientmocks "github.com/trustbloc/fabric-peer-ext/pkg/collections/client/mocks"
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
	txSimulator := &mocks.TxSimulator{}
	txSimulator.GetTxSimulationResultsReturns(&coreledger.TxSimulationResults{}, nil)

	ledger := &mocks.Ledger{
		TxSimulator: txSimulator,
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

	providers := &olclient.ChannelProviders{
		Ledger:           ledger,
		Distributor:      distributor,
		ConfigRetriever:  configRetriever,
		IdentityProvider: identityProvider,
	}
	c := New(channelID, ns1, coll1, providers)
	require.NotNil(t, c)

	value1 := []byte("value1")

	jsonValue := []byte(`{"fieldx":"valuex","field1":"value1","field2":"value2"}`)

	t.Run("Non-JSON -> Success", func(t *testing.T) {
		key, err := c.Put(bytes.NewReader(value1))
		require.NoError(t, err)
		casKey, _, err := dcas.GetCASKeyAndValue(value1)
		require.NoError(t, err)
		assert.Equal(t, casKey, key)
	})

	t.Run("JSON -> Success", func(t *testing.T) {
		key, err := c.Put(bytes.NewReader(jsonValue))
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

		_, err := c.Put(bytes.NewReader(jsonValue))
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

	distributor := &clientmocks.PvtDataDistributor{}
	configRetriever := &mocks.CollectionConfigRetriever{}

	// Mock out all of the dependencies
	signingIdentity := &mocks.SigningIdentity{}
	signingIdentity.SerializeReturns([]byte("creator"), nil)

	identityProvider := &mocks.IdentityProvider{}
	identityProvider.GetDefaultSigningIdentityReturns(signingIdentity, nil)

	providers := &olclient.ChannelProviders{
		Ledger:           ledger,
		Distributor:      distributor,
		ConfigRetriever:  configRetriever,
		IdentityProvider: &mocks.IdentityProvider{},
	}
	c := New(channelID, ns1, coll1, providers)
	require.NotNil(t, c)

	t.Run("Get - success", func(t *testing.T) {
		b := bytes.NewBuffer(nil)
		err := c.Get(key1, b)
		require.NoError(t, err)
		require.Equal(t, value1, b.Bytes())
	})
}
