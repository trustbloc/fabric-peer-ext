/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package support

import (
	"fmt"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

const (
	coll3 = "collection3"
)

func TestConfigRetriever(t *testing.T) {
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

	r := NewCollectionConfigRetriever(channelID, &mocks.Ledger{
		QueryExecutor: mocks.NewQueryExecutor(state),
	})
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
}

func TestConfigRetrieverError(t *testing.T) {
	const channelID = "testchannel"

	state := make(map[string]map[string][]byte)

	expectedErr := fmt.Errorf("injected error")
	r := NewCollectionConfigRetriever(channelID, &mocks.Ledger{
		QueryExecutor: mocks.NewQueryExecutor(state).WithError(expectedErr),
	})
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
}
