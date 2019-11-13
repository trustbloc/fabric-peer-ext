/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package retriever

import (
	"testing"

	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	supportapi "github.com/hyperledger/fabric/extensions/collections/api/support"
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	"github.com/hyperledger/fabric/gossip/common"
	gproto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

const (
	channel1 = "channel1"
)

func TestNewProvider(t *testing.T) {
	p := NewProvider(
		func(channelID string) storeapi.Store { return mocks.NewDataStore() },
		nil,
		func() supportapi.GossipAdapter { return mocks.NewMockGossipAdapter() },
		func(channelID string) gossipapi.BlockPublisher { return mocks.NewBlockPublisher() },
	)
	require.NotNil(t, p)
	require.NotNil(t, p.RetrieverForChannel("testchannel"))
}

func TestGsspProvider(t *testing.T) {
	p := &gsspProvider{getGossip: func() supportapi.GossipAdapter { return mocks.NewMockGossipAdapter() }}
	require.Empty(t, p.PeersOfChannel(common.ChainID(channel1)))
	require.NotPanics(t, func() { p.Send(&gproto.GossipMessage{}) })
	require.Empty(t, p.IdentityInfo())
	require.NotNil(t, p.SelfMembershipInfo())
}
