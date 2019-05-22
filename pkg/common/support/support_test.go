/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package support

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/core/ledger"
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

const (
	ns1   = "chaincode1"
	coll1 = "collection1"
	coll2 = "collection2"
)

func TestSupport(t *testing.T) {
	const channelID = "testchannel"
	const lscc = "lscc"

	nsBuilder := mocks.NewNamespaceBuilder(ns1)
	nsBuilder.Collection(coll1).StaticConfig("OR ('Org1.member','Org2.member')", 3, 3, 100)
	nsBuilder.Collection(coll2).TransientConfig("OR ('Org1.member','Org2.member')", 3, 3, "1m")

	configPkgBytes, err := proto.Marshal(nsBuilder.BuildCollectionConfig())
	require.NoError(t, err)

	state := make(map[string]map[string][]byte)
	state[lscc] = make(map[string][]byte)
	state[lscc][privdata.BuildCollectionKVSKey(ns1)] = configPkgBytes

	ledgerProvider := func(channelID string) ledger.PeerLedger {
		return &mocks.Ledger{
			QueryExecutor: mocks.NewQueryExecutor(state),
		}
	}

	blockPublisherProvider := func(channelID string) gossipapi.BlockPublisher {
		return mocks.NewBlockPublisher()
	}

	s := New(ledgerProvider, blockPublisherProvider)
	require.NotNil(t, s)

	t.Run("Policy", func(t *testing.T) {
		policy, err := s.Policy(channelID, ns1, coll2)
		require.NoError(t, err)
		require.NotNil(t, policy)
	})

	t.Run("Config", func(t *testing.T) {
		config, err := s.Config(channelID, ns1, coll2)
		require.NoError(t, err)
		assert.Equal(t, coll2, config.Name)
		assert.Equal(t, int32(3), config.RequiredPeerCount)
		assert.Equal(t, common.CollectionType_COL_TRANSIENT, config.Type)
		assert.Equal(t, "1m", config.TimeToLive)
	})

	t.Run("BlockPublisher", func(t *testing.T) {
		p := s.BlockPublisher(channelID)
		require.NotNil(t, p)
	})
}
