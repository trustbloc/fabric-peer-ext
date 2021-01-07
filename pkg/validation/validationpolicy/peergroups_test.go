/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validationpolicy

import (
	"testing"

	gdiscovery "github.com/hyperledger/fabric/gossip/discovery"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/common/discovery"
)

const (
	org1MSP = "org1MSP"

	p1Endpoint = "p1.org1.com"
	p2Endpoint = "p2.org1.com"
	p3Endpoint = "p3.org1.com"
	p4Endpoint = "p4.org1.com"
	p5Endpoint = "p5.org1.com"
	p6Endpoint = "p6.org1.com"
	p7Endpoint = "p7.org1.com"
)

var (
	p1 = newMember(org1MSP, p1Endpoint)
	p2 = newMember(org1MSP, p2Endpoint)
	p3 = newMember(org1MSP, p3Endpoint)
	p4 = newMember(org1MSP, p4Endpoint)
	p5 = newMember(org1MSP, p5Endpoint)
	p6 = newMember(org1MSP, p6Endpoint)
	p7 = newMember(org1MSP, p7Endpoint)
)

func TestPeerGroupsSort(t *testing.T) {
	pg1 := discovery.PeerGroup{p2, p1, p3}
	pg2 := discovery.PeerGroup{p5, p4}
	pg3 := discovery.PeerGroup{p7, p6}

	pgs := peerGroups{pg3, pg1, pg2}

	pgs.sort()

	// The peer groups should be sorted
	require.Equal(t, pg1, pgs[0])
	require.Equal(t, pg2, pgs[1])
	require.Equal(t, pg3, pgs[2])

	// Each peer group should be sorted
	require.Equal(t, p1, pg1[0])
	require.Equal(t, p2, pg1[1])
	require.Equal(t, p3, pg1[2])

	require.Equal(t, p4, pg2[0])
	require.Equal(t, p5, pg2[1])

	require.Equal(t, p6, pg3[0])
	require.Equal(t, p7, pg3[1])
}

func TestPeerGroupsContains(t *testing.T) {
	pg1 := discovery.PeerGroup{p1, p2, p3}
	pg2 := discovery.PeerGroup{p3, p4, p5}
	pg3 := discovery.PeerGroup{p1, p4}
	pg4 := discovery.PeerGroup{p2, p5, p6}
	pg5 := discovery.PeerGroup{p6, p7}

	pgs1 := peerGroups{pg1, pg2}

	require.True(t, pgs1.contains(p1))
	require.True(t, pgs1.contains(p2))
	require.True(t, pgs1.contains(p3))
	require.True(t, pgs1.contains(p4))
	require.True(t, pgs1.contains(p5))
	require.False(t, pgs1.contains(p6))

	require.True(t, pgs1.containsAll(pg3))
	require.False(t, pgs1.containsAll(pg4))
	require.True(t, pgs1.containsAny(pg4))
	require.False(t, pgs1.containsAny(pg5))
}

func TestPeerGroups_String(t *testing.T) {
	pg1 := discovery.PeerGroup{p2, p1, p3}
	pg2 := discovery.PeerGroup{p5, p4}
	pg3 := discovery.PeerGroup{p7, p6}

	pgs := peerGroups{pg3, pg1, pg2}

	require.Equal(t, "([p7.org1.com, p6.org1.com], [p2.org1.com, p1.org1.com, p3.org1.com], [p5.org1.com, p4.org1.com])", pgs.String())
}

func newMember(mspID, endpoint string) *discovery.Member {
	return &discovery.Member{
		NetworkMember: gdiscovery.NetworkMember{
			Endpoint: endpoint,
		},
		MSPID: mspID,
	}
}
