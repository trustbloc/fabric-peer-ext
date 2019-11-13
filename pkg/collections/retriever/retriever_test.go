/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package retriever

import (
	"context"
	"testing"

	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	olmocks "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/mocks"
	tdatamocks "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/mocks"
)

const (
	channelID = "testchannel"
)

func TestRetriever(t *testing.T) {
	p := NewProvider().Initialize(&tdatamocks.TransientDataProvider{}, &olmocks.Provider{})
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

	t.Run("Query", func(t *testing.T) {
		retriever := p.RetrieverForChannel(channelID)
		require.NotNil(t, retriever)

		it, err := retriever.Query(context.Background(), storeapi.NewQueryKey("tx1", "ns1", "coll1", "some query"))
		require.NoError(t, err)
		require.NotNil(t, it)
	})
}
