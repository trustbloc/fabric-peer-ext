/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package retriever

import (
	"testing"

	supportapi "github.com/hyperledger/fabric/extensions/collections/api/support"
	"github.com/stretchr/testify/require"
	olapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/api"
	olmocks "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/mocks"
	extretriever "github.com/trustbloc/fabric-peer-ext/pkg/collections/retriever"
	tdataapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/api"
	tdatamocks "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/mocks"
)

func TestNewProvider(t *testing.T) {
	extretriever.SetTransientDataProvider(func(storeProvider func(channelID string) tdataapi.Store, support extretriever.Support, gossipProvider func() supportapi.GossipAdapter) tdataapi.Provider {
		return &tdatamocks.TransientDataProvider{}
	})

	extretriever.SetOffLedgerProvider(func(storeProvider func(channelID string) olapi.Store, support extretriever.Support, gossipProvider func() supportapi.GossipAdapter) olapi.Provider {
		return &olmocks.Provider{}
	})

	p := NewProvider(nil, nil, nil, nil)
	require.NotNil(t, p)
	require.NotNil(t, p.RetrieverForChannel("testchannel"))
}
