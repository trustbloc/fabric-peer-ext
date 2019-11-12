/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package support

import (
	"fmt"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/extensions/gossip/api"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

const (
	channel1 = "channel1"
	channel2 = "channel2"
	coll3    = "collection3"
)

func TestConfigRetriever(t *testing.T) {
	const channelID = "testchannel"
	const lscc = "lscc"

	nsBuilder := mocks.NewNamespaceBuilder(ns1)
	nsBuilder.Collection(coll1).StaticConfig("OR ('Org1.member','Org2.member')", 3, 3, 100)
	nsBuilder.Collection(coll2).TransientConfig("OR ('Org1.member','Org2.member')", 3, 3, "1m")

	configPkgBytes, err := proto.Marshal(nsBuilder.BuildCollectionConfig())
	require.NoError(t, err)

	blockPublisher := mocks.NewBlockPublisher()

	qe := mocks.NewQueryExecutor().WithState(lscc, privdata.BuildCollectionKVSKey(ns1), configPkgBytes)

	r := newCollectionConfigRetriever(channelID, &mocks.Ledger{
		QueryExecutor: qe,
	}, blockPublisher, &mocks.IdentityDeserializer{})
	require.NotNil(t, r)

	t.Run("Policy", func(t *testing.T) {
		policy, err := r.Policy(ns1, coll2)
		require.NoError(t, err)
		require.NotNil(t, policy)
		assert.Equal(t, 2, len(policy.MemberOrgs()))

		policy2, err := r.Policy(ns1, coll2)
		require.NoError(t, err)
		assert.Truef(t, policy == policy2, "expecting policy to be retrieved from cache")

		policy, err = r.Policy(ns1, coll1)
		require.NoError(t, err)
		require.NotNil(t, policy)
		assert.Equal(t, 2, len(policy.MemberOrgs()))
	})

	t.Run("Config", func(t *testing.T) {
		config, err := r.Config(ns1, coll2)
		require.NoError(t, err)
		assert.Equal(t, coll2, config.Name)
		assert.Equal(t, int32(3), config.RequiredPeerCount)
		assert.Equal(t, common.CollectionType_COL_TRANSIENT, config.Type)
		assert.Equal(t, "1m", config.TimeToLive)

		config2, err := r.Config(ns1, coll2)
		require.NoError(t, err)
		assert.Truef(t, config == config2, "expecting config to be retrieved from cache")
	})

	t.Run("Config not found -> error", func(t *testing.T) {
		config, err := r.Config(ns1, coll3)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "configuration not found")
		assert.Nil(t, config)
	})

	t.Run("Chaincode instantiated/upgraded", func(t *testing.T) {
		nsBuilder := mocks.NewNamespaceBuilder(ns1)
		nsBuilder.Collection(coll1).StaticConfig("OR ('Org1.member','Org2.member','Org3.member')", 3, 3, 100)
		nsBuilder.Collection(coll2).TransientConfig("OR ('Org1.member','Org2.member','Org3.member')", 4, 3, "10m")
		ccp := nsBuilder.BuildCollectionConfig()

		err = blockPublisher.HandleLSCCWrite(api.TxMetadata{BlockNum: 1001, TxID: "tx1"}, ns1, &ccprovider.ChaincodeData{Name: ns1}, ccp)
		assert.NoError(t, err)

		// Make sure the new config is loaded
		config, err := r.Config(ns1, coll2)
		require.NoError(t, err)
		assert.Equal(t, coll2, config.Name)
		assert.Equal(t, int32(4), config.RequiredPeerCount)
		assert.Equal(t, common.CollectionType_COL_TRANSIENT, config.Type)
		assert.Equal(t, "10m", config.TimeToLive)

		policy, err := r.Policy(ns1, coll2)
		require.NoError(t, err)
		require.NotNil(t, policy)
		assert.Equal(t, 3, len(policy.MemberOrgs()))

		policy, err = r.Policy(ns1, coll1)
		require.NoError(t, err)
		require.NotNil(t, policy)
		assert.Equal(t, 3, len(policy.MemberOrgs()))
	})
}

func TestConfigRetrieverError(t *testing.T) {
	const channelID = "testchannel"

	blockPublisher := mocks.NewBlockPublisher()

	expectedErr := fmt.Errorf("injected error")
	r := newCollectionConfigRetriever(channelID, &mocks.Ledger{
		QueryExecutor: mocks.NewQueryExecutor().WithError(expectedErr),
	}, blockPublisher, &mocks.IdentityDeserializer{})
	require.NotNil(t, r)

	t.Run("Policy", func(t *testing.T) {
		policy, err := r.Policy(ns1, coll2)
		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
		assert.Nil(t, policy)
	})

	t.Run("Config", func(t *testing.T) {
		config, err := r.Config(ns1, coll2)
		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
		assert.Nil(t, config)
	})

	t.Run("Instantiate/upgrade error", func(t *testing.T) {
		nsBuilder := mocks.NewNamespaceBuilder(ns1)
		nsBuilder.Collection(coll1).StaticConfig("OR ('Org1.member','Org2.member','Org3.member')", 3, 3, 100)
		ccp := nsBuilder.BuildCollectionConfig()
		ccp.Config[0].Payload = nil

		err := blockPublisher.HandleLSCCWrite(api.TxMetadata{BlockNum: 1001, TxID: "tx1"}, ns1, &ccprovider.ChaincodeData{Name: ns1}, ccp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no config found for a collection in namespace")
	})
}

func TestCollectionConfigRetrieverForChannel(t *testing.T) {
	p := NewCollectionConfigRetrieverProvider(
		&mocks.LedgerProvider{},
		mocks.NewBlockPublisherProvider(),
		&mocks.IdentityDeserializerProvider{},
	)
	require.NotNil(t, p)

	r1 := p.ForChannel(channel1)
	require.NotNil(t, r1)
	r1_1 := p.ForChannel(channel1)
	require.Equal(t, r1, r1_1)

	r2 := p.ForChannel(channel2)
	require.NotNil(t, r1)
	require.NotEqual(t, r1, r2)
}
