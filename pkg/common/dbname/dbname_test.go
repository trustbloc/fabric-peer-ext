/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dbname

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/dbname/mocks"
)

//go:generate counterfeiter -o ./mocks/peerconfig.gen.go --fake-name PeerConfig . peerConfig

const (
	org1MSP = "Org1MSP"
	peer1   = "peer1.example.com"
)

func TestResolver_Resolve(t *testing.T) {
	const name1 = "channel1_cc1$$coll1"
	const systemName = "_users"

	peerCfg := &mocks.PeerConfig{}
	peerCfg.MSPIDReturns(org1MSP)
	peerCfg.PeerIDReturns(peer1)

	r := &Resolver{}

	t.Run("Not partitioned", func(t *testing.T) {
		peerCfg.DBPartitionTypeReturns(string(PartitionNone))
		require.True(t, r == r.Initialize(peerCfg))
		require.Equal(t, name1, r.Resolve(name1))
		require.Equal(t, systemName, r.Resolve(systemName))
	})

	t.Run("Partitioned by peer", func(t *testing.T) {
		peerCfg.DBPartitionTypeReturns(string(PartitionPeer))
		require.True(t, r == r.Initialize(peerCfg))
		require.Equal(t, "peer1-example-com_channel1_cc1$$coll1", r.Resolve(name1))
		require.Equal(t, systemName, r.Resolve(systemName))
	})

	t.Run("Partitioned by MSP", func(t *testing.T) {
		peerCfg.DBPartitionTypeReturns(string(PartitionMSP))
		require.True(t, r == r.Initialize(peerCfg))
		require.Equal(t, "org1msp_channel1_cc1$$coll1", r.Resolve(name1))
		require.Equal(t, systemName, r.Resolve(systemName))
	})
}

func TestResolver_IsRelevant(t *testing.T) {
	const name1 = "channel1_cc1$$coll1"
	const systemName = "_users"

	peerCfg := &mocks.PeerConfig{}
	peerCfg.MSPIDReturns(org1MSP)
	peerCfg.PeerIDReturns(peer1)

	r := &Resolver{}

	t.Run("Not partitioned", func(t *testing.T) {
		peerCfg.DBPartitionTypeReturns(string(PartitionNone))
		require.True(t, r == r.Initialize(peerCfg))
		require.True(t, r.IsRelevant(name1))
		require.True(t, r.IsRelevant(systemName))
	})

	t.Run("Partitioned by peer", func(t *testing.T) {
		peerCfg.DBPartitionTypeReturns(string(PartitionPeer))
		require.True(t, r == r.Initialize(peerCfg))
		require.True(t, r.IsRelevant("peer1-example-com_channel1_cc1$$coll1"))
		require.False(t, r.IsRelevant("peer2-example-com_channel1_cc1$$coll1"))
		require.True(t, r.IsRelevant(systemName))
	})

	t.Run("Partitioned by MSP", func(t *testing.T) {
		peerCfg.DBPartitionTypeReturns(string(PartitionMSP))
		require.True(t, r == r.Initialize(peerCfg))
		require.Equal(t, "org1msp_channel1_cc1$$coll1", r.Resolve(name1))
		require.True(t, r.IsRelevant("org1msp_channel1_cc1$$coll1"))
		require.False(t, r.IsRelevant("org2msp_channel1_cc1$$coll1"))
		require.True(t, r.IsRelevant(systemName))
	})
}
