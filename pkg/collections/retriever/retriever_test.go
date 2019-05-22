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

	p := NewProvider(nil, nil, nil, nil)
	require.NotNil(t, p)

	retriever := p.RetrieverForChannel(channelID)
	require.NotNil(t, retriever)

	const key1 = "key1"

	v, err := retriever.GetTransientData(context.Background(), &storeapi.Key{Key: key1})
	assert.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, []byte(key1), v.Value)

	vals, err := retriever.GetTransientDataMultipleKeys(context.Background(), &storeapi.MultiKey{Keys: []string{key1}})
	assert.NoError(t, err)
	require.Equal(t, 1, len(vals))
	assert.Equal(t, []byte(key1), vals[0].Value)
}
