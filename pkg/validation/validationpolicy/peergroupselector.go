/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validationpolicy

import (
	cb "github.com/hyperledger/fabric-protos-go/common"
	"github.com/pkg/errors"

	"github.com/trustbloc/fabric-peer-ext/pkg/common/discovery"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

// peerGroupSelector selects the groups of peers for validating a block based on a validation policy
type peerGroupSelector struct {
	policy     *policy
	channelID  string
	block      *cb.Block
	committers discovery.PeerGroup
	validators discovery.PeerGroup
}

func newPeerGroupSelector(channelID string, policy *policy, disc *discovery.Discovery, block *cb.Block) *peerGroupSelector {
	return &peerGroupSelector{
		policy:     policy,
		channelID:  channelID,
		block:      block,
		committers: peersWithRole(disc, roles.CommitterRole),
		validators: peersWithRole(disc, roles.ValidatorRole),
	}
}

// groups selects the groups of peers for validating a block based on a validation policy. The choice of peers is based on the following criteria:
//
// (1) committerTransactionThreshold - This transaction threshold indicates that the committer should validate the block if
//    the number of transactions is less than this value, even if the committer does not have the validator role. If set to 0
//    then the committer (without the validator role) will never perform validation unless there are no validators available.
//
// (2) singlePeerTransactionThreshold - This transaction threshold indicates that only a single peer with the validator role should
//    validate the block if the number of transactions is less than this threshold.
//
// Example 1:
//  Given:
//   - Peer0 = committer
//   - Peer1 = validator
//   - Peer2 = validator
//   - committerTransactionThreshold = 5
//   - singlePeerTransactionThreshold = 20
//  If:
//   - Block 1 has 4 transactions then the peer group is [Peer0] (since Peer0 is a committer)
//   - Block 2 has 5 transactions then the peer group is [Peer1 or Peer2]
//   - Block 3 has 20 transactions then the peer group is [Peer1 and Peer2]
//
// Example 2:
//   Given:
//   - Peer0 = committer,validator
//   - Peer1 = validator
//   - Peer2 = validator
//   - committerTransactionThreshold = 5
//   - singlePeerTransactionThreshold = 20
//   If:
//   - Block 1 has 4 transactions then the peer group is [Peer0] (since Peer0 is a committer)
//   - Block 2 has 5 transactions then the peer group is [Peer0] (since Peer0 is a committer and validator)
//   - Block 3 has 20 transactions then the peer group is [Peer0 and Peer1 and Peer2]
//
func (s *peerGroupSelector) groups() (peerGroups, error) {
	logger.Debugf("[%s] Selecting peer groups to validate block %d ...", s.channelID, s.block.Header.Number)

	numTransactions := len(s.block.Data.Data)

	var selectedValidator *discovery.Member

	// check committer threshold
	if numTransactions < s.policy.committerTransactionThreshold {
		selectedValidator = s.selectCommitter()
	}

	// check single peer threshold
	if selectedValidator == nil && numTransactions < s.policy.singlePeerTransactionThreshold {
		selectedValidator = s.selectValidator()
	}

	// select committer if no validators
	if selectedValidator == nil && len(s.validators) == 0 {
		logger.Debugf("[%s] There are no validators for block %d. Selecting the committer to validate the block.", s.channelID, s.block.Header.Number)

		selectedValidator = s.selectCommitter()

		if selectedValidator == nil {
			logger.Errorf("[%s] There are no validators or committers for block %d. Please ensure that at least one peer has the committer role.", s.channelID, s.block.Header.Number)

			return nil, errors.Errorf("there are no validators or committers for block %d", s.block.Header.Number)
		}
	}

	var groups peerGroups

	if selectedValidator != nil {
		groups = asPeerGroups(selectedValidator)
	} else {
		groups = asPeerGroups(s.validators...)
	}

	groups.sort()

	logger.Debugf("[%s] Peer groups for block %d: %s", s.channelID, s.block.Header.Number, groups)

	return groups, nil
}

func (s *peerGroupSelector) selectCommitter() *discovery.Member {
	if len(s.committers) == 0 {
		return nil
	}

	peer := s.committers[0]

	logger.Debugf("[%s] Selected validator is %s (committer) since the number of transactions in block %d is %d which is less than the committer threshold %d",
		s.channelID, peer.Endpoint, s.block.Header.Number, len(s.block.Data.Data), s.policy.committerTransactionThreshold)

	return peer
}

func (s *peerGroupSelector) selectValidator() *discovery.Member {
	// First check if the peer is also a validator. If so, select the committing peer since it's more efficient, i.s. no Gossip required.
	for _, peer := range s.committers {
		if peer.HasRole(roles.ValidatorRole) {
			logger.Debugf("[%s] Selected validator is %s (committer,validator) since the number of transactions in block %d is %d which is less than the single validator threshold %d",
				s.channelID, peer.Endpoint, s.block.Header.Number, len(s.block.Data.Data), s.policy.singlePeerTransactionThreshold)

			return peer
		}
	}

	if len(s.validators) == 0 {
		return nil
	}

	// Pick one peer deterministically from the set of validators
	peer := s.validators[s.block.Header.Number%uint64(len(s.validators))]

	logger.Debugf("[%s] Selected validator is %s since the number of transactions in block %d is %d which is less than the single validator threshold %d",
		s.channelID, peer.Endpoint, s.block.Header.Number, len(s.block.Data.Data), s.policy.singlePeerTransactionThreshold)

	return peer
}

func peersWithRole(disc *discovery.Discovery, role roles.Role) discovery.PeerGroup {
	return disc.GetMembers(
		func(member *discovery.Member) bool {
			if member.MSPID != disc.Self().MSPID || member.Properties == nil {
				return false
			}

			return roles.FromStrings(member.Properties.Roles...).Contains(role)
		},
	)
}

func asPeerGroups(peers ...*discovery.Member) peerGroups {
	groups := make(peerGroups, len(peers))

	for i, v := range peers {
		groups[i] = discovery.PeerGroup{v}
	}

	return groups
}
