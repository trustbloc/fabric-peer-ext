/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package retriever

import (
	"context"
	"testing"

	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	supportapi "github.com/hyperledger/fabric/extensions/collections/api/support"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	olapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/api"
	olmocks "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/mocks"
	tdataapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/api"
	tdatamocks "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/mocks"
)

const (
	channelID = "testchannel"
)

func TestRetriever(t *testing.T) {
	getTransientDataProvider = func(storeProvider func(channelID string) tdataapi.Store, support Support, gossipProvider func() supportapi.GossipAdapter) tdataapi.Provider {
		return &tdatamocks.TransientDataProvider{}
	}

	getOffLedgerProvider = func(storeProvider func(channelID string) olapi.Store, support Support, gossipProvider func() supportapi.GossipAdapter) olapi.Provider {
		return &olmocks.Provider{}
	}

	p := NewProvider(nil, nil, nil, nil)
	require.NotNil(t, p)

	const key1 = "key1"

	t.Run("GetTransientData", func(t *testing.T) {
		retriever := p.RetrieverForChannel(channelID)
		require.NotNil(t, retriever)

		v, err := retriever.GetTransientData(context.Background(), &storeapi.Key{Key: key1})
		assert.NoError(t, err)
		require.NotNil(t, v)
		assert.Equal(t, []byte(key1), v.Value)
	})

	t.Run("GetTransientDataMultipleKeys", func(t *testing.T) {
		retriever := p.RetrieverForChannel(channelID)
		require.NotNil(t, retriever)

		vals, err := retriever.GetTransientDataMultipleKeys(context.Background(), &storeapi.MultiKey{Keys: []string{key1}})
		assert.NoError(t, err)
		require.Equal(t, 1, len(vals))
		assert.Equal(t, []byte(key1), vals[0].Value)
	})

	t.Run("GetData", func(t *testing.T) {
		retriever := p.RetrieverForChannel(channelID)
		require.NotNil(t, retriever)

		v, err := retriever.GetData(context.Background(), &storeapi.Key{Key: key1})
		assert.NoError(t, err)
		require.NotNil(t, v)
		assert.Equal(t, []byte(key1), v.Value)
	})

	t.Run("GetDataMultipleKeys", func(t *testing.T) {
		retriever := p.RetrieverForChannel(channelID)
		require.NotNil(t, retriever)

		vals, err := retriever.GetDataMultipleKeys(context.Background(), &storeapi.MultiKey{Keys: []string{key1}})
		assert.NoError(t, err)
		require.Equal(t, 1, len(vals))
		assert.Equal(t, []byte(key1), vals[0].Value)
	})
}
