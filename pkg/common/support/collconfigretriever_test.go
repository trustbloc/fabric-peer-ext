/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package support

import (
	"fmt"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/ledger/rwset/kvrwset"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/extensions/gossip/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cmocks "github.com/trustbloc/fabric-peer-ext/pkg/common/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

//go:generate counterfeiter -o ./../smocks/statedbprovider.gen.go -fake-name StateDBProvider . stateDBProvider

const (
	channel1 = "channel1"
	channel2 = "channel2"
	ns1      = "chaincode1"
	coll1    = "collection1"
	coll2    = "collection2"
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

	ccInfo := &ledger.DeployedChaincodeInfo{
		ExplicitCollectionConfigPkg: nsBuilder.BuildCollectionConfig(),
	}

	ccinfoProvider := mocks.NewChaincodeInfoProvider().WithData(ns1, ccInfo)

	r := newCollectionConfigRetriever(
		channelID, qe,
		blockPublisher,
		&mocks.IdentityDeserializer{},
		&mocks.IdentifierProvider{},
		ccinfoProvider,
	)
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
		assert.Equal(t, pb.CollectionType_COL_TRANSIENT, config.Type)
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
		assert.Equal(t, pb.CollectionType_COL_TRANSIENT, config.Type)
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

	t.Run("Chaincode committed", func(t *testing.T) {
		defer func() {
			ccinfoProvider.WithData(ns1, ccInfo)
		}()

		nsBuilder := mocks.NewNamespaceBuilder(ns1)
		nsBuilder.Collection(coll1).StaticConfig("OR ('Org1.member','Org2.member','Org3.member')", 3, 3, 100)
		nsBuilder.Collection(coll2).TransientConfig("OR ('Org1.member','Org2.member','Org3.member')", 4, 3, "10m")

		ccinfoProvider.WithData(ns1, &ledger.DeployedChaincodeInfo{
			ExplicitCollectionConfigPkg: nsBuilder.BuildCollectionConfig(),
		})

		require.NoError(t, blockPublisher.HandleWrite(
			api.TxMetadata{BlockNum: 1001, TxID: "tx1"},
			"non-lifecycle-namespace",
			&kvrwset.KVWrite{Key: "some key"},
		))

		require.NoError(t, blockPublisher.HandleWrite(
			api.TxMetadata{BlockNum: 1001, TxID: "tx1"},
			lifecycleNamespace,
			&kvrwset.KVWrite{Key: fmt.Sprintf("namespaces/fields/%s/Collections", ns1)},
		))

		// Make sure the new config is loaded
		config, err := r.Config(ns1, coll2)
		require.NoError(t, err)
		assert.Equal(t, coll2, config.Name)
		assert.Equal(t, int32(4), config.RequiredPeerCount)
		assert.Equal(t, pb.CollectionType_COL_TRANSIENT, config.Type)
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

	r := newCollectionConfigRetriever(
		channelID,
		mocks.NewQueryExecutor(),
		blockPublisher,
		&mocks.IdentityDeserializer{},
		&mocks.IdentifierProvider{},
		mocks.NewChaincodeInfoProvider().WithError(expectedErr),
	)
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
		&cmocks.StateDBProvider{},
		mocks.NewBlockPublisherProvider(),
		&mocks.IdentityDeserializerProvider{},
		&mocks.IdentifierProvider{},
		mocks.NewChaincodeInfoProvider(),
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

func TestGetLifecycleChaincodeID(t *testing.T) {
	const cc1 = "cc1"

	t.Run("Valid key", func(t *testing.T) {
		require.Equal(t, cc1, getLifecycleChaincodeID(&kvrwset.KVWrite{
			Key: fmt.Sprintf("%s%s%s", fieldsPrefix, cc1, collectionsSuffix),
		}))
	})

	t.Run("Invalid prefix", func(t *testing.T) {
		require.Empty(t, getLifecycleChaincodeID(&kvrwset.KVWrite{
			Key: fmt.Sprintf("xxx%s%s", cc1, collectionsSuffix),
		}))
	})

	t.Run("Invalid collections suffix", func(t *testing.T) {
		require.Empty(t, getLifecycleChaincodeID(&kvrwset.KVWrite{
			Key: fmt.Sprintf("%s%sxxx", fieldsPrefix, cc1),
		}))
	})
}
