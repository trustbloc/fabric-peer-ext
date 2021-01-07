/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validationpolicy

import (
	"testing"

	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/common/discovery"
	extmocks "github.com/trustbloc/fabric-peer-ext/pkg/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

//go:generate counterfeiter -o ./mocks/policiesprovider.gen.go --fake-name PoliciesProvider github.com/hyperledger/fabric/common/policies.Provider

var (
	org1MSPID = "Org1MSP"
	org2MSPID = "Org2MSP"

	p1Org1Endpoint = "p1.org1.com"
	p1Org1PKIID    = common.PKIidType("pkiid_P1O1")
	p2Org1Endpoint = "p2.org1.com"
	p2Org1PKIID    = common.PKIidType("pkiid_P2O1")
	p3Org1Endpoint = "p3.org1.com"
	p3Org1PKIID    = common.PKIidType("pkiid_P3O1")

	p1Org2Endpoint = "p1.org2.com"
	p1Org2PKIID    = common.PKIidType("pkiid_P1O2")
)

// Ensure that the roles are initialized
var _ = roles.GetRoles()

func TestEvaluator_PeerGroups(t *testing.T) {
	channelID := "testchannel"

	bb := extmocks.NewBlockBuilder(channelID, 1000)
	bb.Transaction("tx1", peer.TxValidationCode_NOT_VALIDATED)
	bb.Transaction("tx2", peer.TxValidationCode_NOT_VALIDATED)
	bb.Transaction("tx3", peer.TxValidationCode_NOT_VALIDATED)
	bb.Transaction("tx4", peer.TxValidationCode_NOT_VALIDATED)
	bb.Transaction("tx5", peer.TxValidationCode_NOT_VALIDATED)
	block := bb.Build()

	t.Run("Num transactions less than committer threshold -> committer validates", func(t *testing.T) {
		reset := setRoles(roles.ValidatorRole)
		defer reset()

		cfg := &policy{
			committerTransactionThreshold:  6,
			singlePeerTransactionThreshold: 0,
		}

		gossip := extmocks.NewMockGossipAdapter().
			Self(org1MSPID, extmocks.NewMember(p1Org1Endpoint, p1Org1PKIID)).
			Member(org1MSPID, extmocks.NewMember(p2Org1Endpoint, p2Org1PKIID, roles.CommitterRole)).
			Member(org1MSPID, extmocks.NewMember(p3Org1Endpoint, p3Org1PKIID, roles.ValidatorRole)).
			Member(org2MSPID, extmocks.NewMember(p1Org2Endpoint, p1Org2PKIID, roles.ValidatorRole))

		peerGroups, err := newPeerGroupSelector(channelID, cfg, discovery.New(channelID, gossip), block).groups()
		require.NoError(t, err)
		require.Equal(t, 1, len(peerGroups))
		require.Equal(t, []string{p2Org1Endpoint}, asEndpoints(peerGroups[0]...))
	})

	t.Run("Num transactions less than single-peer transaction threshold -> one validator validates", func(t *testing.T) {
		reset := setRoles(roles.ValidatorRole)
		defer reset()

		cfg := &policy{
			committerTransactionThreshold:  2,
			singlePeerTransactionThreshold: 6,
		}

		t.Run("Committer also validator -> committer validates", func(t *testing.T) {
			gossip := extmocks.NewMockGossipAdapter().
				Self(org1MSPID, extmocks.NewMember(p1Org1Endpoint, p1Org1PKIID)).
				Member(org1MSPID, extmocks.NewMember(p2Org1Endpoint, p2Org1PKIID, roles.CommitterRole, roles.ValidatorRole)).
				Member(org1MSPID, extmocks.NewMember(p3Org1Endpoint, p3Org1PKIID, roles.ValidatorRole))

			peerGroups, err := newPeerGroupSelector(channelID, cfg, discovery.New(channelID, gossip), block).groups()
			require.NoError(t, err)
			require.Equal(t, 1, len(peerGroups))
			require.Equal(t, []string{p2Org1Endpoint}, asEndpoints(peerGroups[0]...))
		})

		t.Run("Committer not validator -> non-committer validates", func(t *testing.T) {
			gossip := extmocks.NewMockGossipAdapter().
				Self(org1MSPID, extmocks.NewMember(p1Org1Endpoint, p1Org1PKIID)).
				Member(org1MSPID, extmocks.NewMember(p2Org1Endpoint, p2Org1PKIID, roles.CommitterRole)).
				Member(org1MSPID, extmocks.NewMember(p3Org1Endpoint, p3Org1PKIID, roles.ValidatorRole))

			peerGroups, err := newPeerGroupSelector(channelID, cfg, discovery.New(channelID, gossip), block).groups()
			require.NoError(t, err)
			require.Equal(t, 1, len(peerGroups))
			require.Equal(t, []string{p3Org1Endpoint}, asEndpoints(peerGroups[0]...))
		})

		t.Run("No validators -> committer validates", func(t *testing.T) {
			reset := setRoles(roles.EndorserRole)
			defer reset()

			gossip := extmocks.NewMockGossipAdapter().
				Self(org1MSPID, extmocks.NewMember(p1Org1Endpoint, p1Org1PKIID)).
				Member(org1MSPID, extmocks.NewMember(p2Org1Endpoint, p2Org1PKIID, roles.CommitterRole)).
				Member(org1MSPID, extmocks.NewMember(p3Org1Endpoint, p3Org1PKIID, roles.EndorserRole))

			peerGroups, err := newPeerGroupSelector(channelID, cfg, discovery.New(channelID, gossip), block).groups()
			require.NoError(t, err)
			require.Equal(t, 1, len(peerGroups))
			require.Equal(t, []string{p2Org1Endpoint}, asEndpoints(peerGroups[0]...))
		})
	})

	t.Run("Num transactions greater than single-peer transaction threshold -> all validators validate", func(t *testing.T) {
		reset := setRoles(roles.ValidatorRole)
		defer reset()

		cfg := &policy{
			committerTransactionThreshold:  2,
			singlePeerTransactionThreshold: 5,
		}

		gossip := extmocks.NewMockGossipAdapter().
			Self(org1MSPID, extmocks.NewMember(p1Org1Endpoint, p1Org1PKIID)).
			Member(org1MSPID, extmocks.NewMember(p2Org1Endpoint, p2Org1PKIID, roles.CommitterRole)).
			Member(org1MSPID, extmocks.NewMember(p3Org1Endpoint, p3Org1PKIID, roles.ValidatorRole))

		peerGroups, err := newPeerGroupSelector(channelID, cfg, discovery.New(channelID, gossip), block).groups()
		require.NoError(t, err)
		require.Equal(t, 2, len(peerGroups))
		require.Equal(t, []string{p1Org1Endpoint}, asEndpoints(peerGroups[0]...))
		require.Equal(t, []string{p3Org1Endpoint}, asEndpoints(peerGroups[1]...))
	})

	t.Run("No validators -> committer validates", func(t *testing.T) {
		reset := setRoles(roles.EndorserRole)
		defer reset()

		gossip := extmocks.NewMockGossipAdapter().
			Self(org1MSPID, extmocks.NewMember(p1Org1Endpoint, p1Org1PKIID)).
			Member(org1MSPID, extmocks.NewMember(p2Org1Endpoint, p2Org1PKIID, roles.CommitterRole)).
			Member(org1MSPID, extmocks.NewMember(p3Org1Endpoint, p3Org1PKIID, roles.EndorserRole))

		cfg := &policy{
			committerTransactionThreshold:  2,
			singlePeerTransactionThreshold: 5,
		}

		peerGroups, err := newPeerGroupSelector(channelID, cfg, discovery.New(channelID, gossip), block).groups()
		require.NoError(t, err)
		require.Equal(t, 1, len(peerGroups))
		require.Equal(t, []string{p2Org1Endpoint}, asEndpoints(peerGroups[0]...))
	})

	t.Run("No validators & no committers -> error", func(t *testing.T) {
		reset := setRoles(roles.EndorserRole)
		defer reset()

		gossip := extmocks.NewMockGossipAdapter().
			Self(org1MSPID, extmocks.NewMember(p1Org1Endpoint, p1Org1PKIID)).
			Member(org1MSPID, extmocks.NewMember(p2Org1Endpoint, p2Org1PKIID, roles.EndorserRole)).
			Member(org1MSPID, extmocks.NewMember(p3Org1Endpoint, p3Org1PKIID, roles.EndorserRole))

		cfg := &policy{
			committerTransactionThreshold:  2,
			singlePeerTransactionThreshold: 5,
		}

		peerGroups, err := newPeerGroupSelector(channelID, cfg, discovery.New(channelID, gossip), block).groups()
		require.Error(t, err)
		require.Contains(t, err.Error(), "no validators or committers")
		require.Empty(t, peerGroups)
	})
}

func asEndpoints(members ...*discovery.Member) []string {
	var endpoints []string
	for _, member := range members {
		endpoints = append(endpoints, member.Endpoint)
	}
	return endpoints
}

func setRoles(role ...roles.Role) func() {
	rolesValue := make(map[roles.Role]struct{})

	for _, r := range role {
		rolesValue[r] = struct{}{}
	}

	roles.SetRoles(rolesValue)

	return func() { roles.SetRoles(nil) }
}
