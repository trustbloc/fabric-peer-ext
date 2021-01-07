/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validationpolicy

import (
	"testing"

	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/validation/validationresults"

	"github.com/trustbloc/fabric-peer-ext/pkg/common/discovery"
	extmocks "github.com/trustbloc/fabric-peer-ext/pkg/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
	"github.com/trustbloc/fabric-peer-ext/pkg/validation/validationpolicy/mocks"
)

//go:generate counterfeiter -o ./mocks/policiesprovider.gen.go --fake-name PoliciesProvider github.com/hyperledger/fabric/common/policies.Provider

func TestPolicyEvaluator_New(t *testing.T) {
	channelID := "testchannel"

	require.NotNil(t, New(channelID, discovery.New(channelID, extmocks.NewMockGossipAdapter()), &mocks.PolicyProvider{}))
}

func TestPolicyEvaluator_GetValidatingPeers(t *testing.T) {
	channelID := "testchannel"

	bb := extmocks.NewBlockBuilder(channelID, 1000)
	bb.Transaction("tx1", peer.TxValidationCode_NOT_VALIDATED)
	bb.Transaction("tx2", peer.TxValidationCode_NOT_VALIDATED)
	bb.Transaction("tx3", peer.TxValidationCode_NOT_VALIDATED)
	bb.Transaction("tx4", peer.TxValidationCode_NOT_VALIDATED)
	bb.Transaction("tx5", peer.TxValidationCode_NOT_VALIDATED)
	block := bb.Build()

	t.Run("Success", func(t *testing.T) {
		reset := setRoles(roles.ValidatorRole)
		defer reset()

		gossip := extmocks.NewMockGossipAdapter().
			Self(org1MSPID, extmocks.NewMember(p1Org1Endpoint, p1Org1PKIID)).
			Member(org1MSPID, extmocks.NewMember(p2Org1Endpoint, p2Org1PKIID, roles.CommitterRole)).
			Member(org1MSPID, extmocks.NewMember(p3Org1Endpoint, p3Org1PKIID, roles.ValidatorRole))

		policy := &policy{
			committerTransactionThreshold:  1,
			singlePeerTransactionThreshold: 3,
		}

		evaluator := &PolicyEvaluator{
			channelID:      channelID,
			Discovery:      discovery.New(channelID, gossip),
			policy:         policy,
			policyProvider: &mocks.PolicyProvider{},
		}

		peers, err := evaluator.GetValidatingPeers(block)
		require.NoError(t, err)
		require.Equal(t, 2, len(peers))
	})

	t.Run("No validators or committers -> error", func(t *testing.T) {
		reset := setRoles(roles.EndorserRole)
		defer reset()

		gossip := extmocks.NewMockGossipAdapter().
			Self(org1MSPID, extmocks.NewMember(p1Org1Endpoint, p1Org1PKIID)).
			Member(org1MSPID, extmocks.NewMember(p2Org1Endpoint, p2Org1PKIID, roles.EndorserRole)).
			Member(org1MSPID, extmocks.NewMember(p3Org1Endpoint, p3Org1PKIID, roles.EndorserRole))

		policy := &policy{
			committerTransactionThreshold:  1,
			singlePeerTransactionThreshold: 3,
		}

		evaluator := &PolicyEvaluator{
			channelID:      channelID,
			Discovery:      discovery.New(channelID, gossip),
			policy:         policy,
			policyProvider: &mocks.PolicyProvider{},
		}

		peers, err := evaluator.GetValidatingPeers(block)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no validators or committers")
		require.Empty(t, peers)
	})
}

func TestPolicyEvaluator_GetTxFilter(t *testing.T) {
	channelID := "testchannel"

	bb := extmocks.NewBlockBuilder(channelID, 1000)
	bb.Transaction("tx1", peer.TxValidationCode_NOT_VALIDATED)
	bb.Transaction("tx2", peer.TxValidationCode_NOT_VALIDATED)
	bb.Transaction("tx3", peer.TxValidationCode_NOT_VALIDATED)
	bb.Transaction("tx4", peer.TxValidationCode_NOT_VALIDATED)
	bb.Transaction("tx5", peer.TxValidationCode_NOT_VALIDATED)
	block := bb.Build()

	t.Run("Success", func(t *testing.T) {
		reset := setRoles(roles.ValidatorRole)
		defer reset()

		gossip := extmocks.NewMockGossipAdapter().
			Self(org1MSPID, extmocks.NewMember(p1Org1Endpoint, p1Org1PKIID)).
			Member(org1MSPID, extmocks.NewMember(p2Org1Endpoint, p2Org1PKIID, roles.CommitterRole)).
			Member(org1MSPID, extmocks.NewMember(p3Org1Endpoint, p3Org1PKIID, roles.ValidatorRole))

		policy := &policy{
			committerTransactionThreshold:  1,
			singlePeerTransactionThreshold: 3,
		}

		evaluator := &PolicyEvaluator{
			channelID:      channelID,
			Discovery:      discovery.New(channelID, gossip),
			policy:         policy,
			policyProvider: &mocks.PolicyProvider{},
		}

		txFilter := evaluator.GetTxFilter(block)
		require.NotNil(t, txFilter)

		require.True(t, txFilter(0))
		require.False(t, txFilter(1))
		require.True(t, txFilter(2))
		require.False(t, txFilter(3))
		require.True(t, txFilter(4))
	})

	t.Run("Policy error -> local peer validates all transactions", func(t *testing.T) {
		reset := setRoles(roles.EndorserRole)
		defer reset()

		gossip := extmocks.NewMockGossipAdapter().
			Self(org1MSPID, extmocks.NewMember(p1Org1Endpoint, p1Org1PKIID)).
			Member(org1MSPID, extmocks.NewMember(p2Org1Endpoint, p2Org1PKIID, roles.EndorserRole)).
			Member(org1MSPID, extmocks.NewMember(p3Org1Endpoint, p3Org1PKIID, roles.EndorserRole))

		policy := &policy{
			committerTransactionThreshold:  1,
			singlePeerTransactionThreshold: 3,
		}

		evaluator := &PolicyEvaluator{
			channelID:      channelID,
			Discovery:      discovery.New(channelID, gossip),
			policy:         policy,
			policyProvider: &mocks.PolicyProvider{},
		}

		txFilter := evaluator.GetTxFilter(block)
		require.NotNil(t, txFilter)
		require.True(t, txFilter(0))
		require.True(t, txFilter(1))
		require.True(t, txFilter(2))
		require.True(t, txFilter(3))
		require.True(t, txFilter(4))
	})
}

func TestPolicyEvaluator_ValidateResults(t *testing.T) {
	channelID := "testchannel"

	gossip := extmocks.NewMockGossipAdapter().
		Self(org1MSPID, extmocks.NewMember(p1Org1Endpoint, p1Org1PKIID)).
		Member(org1MSPID, extmocks.NewMember(p2Org1Endpoint, p2Org1PKIID, roles.EndorserRole)).
		Member(org1MSPID, extmocks.NewMember(p3Org1Endpoint, p3Org1PKIID, roles.EndorserRole)).
		Member(org2MSPID, extmocks.NewMember(p1Org2Endpoint, p1Org2PKIID, roles.ValidatorRole))

	evaluator := New(channelID, discovery.New(channelID, gossip), &mocks.PolicyProvider{})
	require.NotNil(t, evaluator)

	t.Run("Local peer", func(t *testing.T) {
		err := evaluator.ValidateResults(
			[]*validationresults.Results{
				{
					BlockNumber: 1000,
					Endpoint:    p1Org1Endpoint,
					MSPID:       org1MSPID,
					TxFlags:     []uint8{0},
					Local:       true,
				},
			},
		)
		require.NoError(t, err)
	})

	t.Run("Remote peers", func(t *testing.T) {
		err := evaluator.ValidateResults(
			[]*validationresults.Results{
				{
					BlockNumber: 1000,
					Endpoint:    p1Org2Endpoint,
					MSPID:       org2MSPID,
					TxFlags:     []uint8{0},
					Identity:    []byte(p1Org2Endpoint),
					Signature:   []byte("p1Org2signature"),
				},
				{
					BlockNumber: 1000,
					Endpoint:    p2Org1Endpoint,
					MSPID:       org1MSPID,
					TxFlags:     []uint8{0},
					Identity:    []byte(p2Org1Endpoint),
					Signature:   []byte("p2Org1signature"),
				},
			},
		)
		require.NoError(t, err)
	})
}
