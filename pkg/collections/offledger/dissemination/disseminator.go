/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dissemination

import (
	"math/rand"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/discovery"
	"github.com/trustbloc/fabric-peer-ext/pkg/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

var logger = flogging.MustGetLogger("ext_offledger")

// Disseminator disseminates collection data to other endorsers
type Disseminator struct {
	*discovery.Discovery
	namespace  string
	collection string
	policy     privdata.CollectionAccessPolicy
}

// New returns a new disseminator
func New(channelID, namespace, collection string, policy privdata.CollectionAccessPolicy, gossip gossipAdapter) *Disseminator {
	return &Disseminator{
		Discovery:  discovery.New(channelID, gossip),
		namespace:  namespace,
		collection: collection,
		policy:     policy,
	}
}

// resolvePeersForDissemination resolves to a set of committers to which data should be disseminated
func (d *Disseminator) resolvePeersForDissemination() discovery.PeerGroup {
	orgs := d.policy.MemberOrgs()
	maxPeerCount := d.policy.MaximumPeerCount()

	logger.Debugf("[%s] MaximumPeerCount: %d, RequiredPeerCount: %d, Member orgs: %s", d.ChannelID(), maxPeerCount, d.policy.RequiredPeerCount(), orgs)

	var committers discovery.PeerGroup

	logger.Debugf("maxPeerCount: %d, Roles: %s", maxPeerCount, d.Self().Roles())
	if maxPeerCount == 0 && !d.Self().HasRole(roles.CommitterRole) {
		logger.Debugf("[%s] MaximumPeerCount is 0 and I am not a committer. Getting a random peer for dissemination from orgs %s", d.ChannelID(), orgs)
		committers = getRandomPeers(d.getPeersWithRole(roles.CommitterRole, orgs), 1)
	} else {
		logger.Debugf("[%s] Getting %d random peer(s) with the 'committer' role for dissemination from orgs %s", d.ChannelID(), maxPeerCount, orgs)
		committers = getRandomPeers(d.getPeersWithRole(roles.CommitterRole, orgs), maxPeerCount)
	}

	if len(committers) < maxPeerCount {
		logger.Debugf("[%s] MaximumPeerCount in collection policy is %d and we only have %d peer(s) with the 'committer' role. Adding some endorsers too...", d.ChannelID(), maxPeerCount, len(committers))
		for _, peer := range d.getPeersWithRole(roles.EndorserRole, orgs).Remote().Shuffle() {
			if len(committers) >= maxPeerCount {
				// We have enough peers
				break
			}
			logger.Debugf("Adding endorser [%s] ...", peer)
			committers = append(committers, peer)
		}
	}

	logger.Debugf("[%s] Peers for dissemination from orgs %s: %s", d.ChannelID(), orgs, committers)

	return committers
}

// ResolvePeersForRetrieval resolves to a set of peers from which data should may be retrieved
func (d *Disseminator) ResolvePeersForRetrieval() discovery.PeerGroup {
	orgs := d.policy.MemberOrgs()

	logger.Debugf("[%s] Member orgs: %s", d.ChannelID(), orgs)

	// Maximum number of peers to ask for the data
	maxPeers := getMaxPeersForRetrieval()

	var peersForRetrieval discovery.PeerGroup
	for _, peer := range d.getPeersWithRole(roles.EndorserRole, orgs).Remote().Shuffle() {
		if len(peersForRetrieval) >= maxPeers {
			// We have enough peers
			break
		}
		logger.Debugf("Adding endorser [%s] ...", peer)
		peersForRetrieval = append(peersForRetrieval, peer)
	}

	if len(peersForRetrieval) < maxPeers {
		// Add some committers too
		for _, peer := range d.getPeersWithRole(roles.CommitterRole, orgs).Remote().Shuffle() {
			if len(peersForRetrieval) >= maxPeers {
				// We have enough peers
				break
			}
			logger.Debugf("Adding committer [%s] ...", peer)
			peersForRetrieval = append(peersForRetrieval, peer)
		}
	}

	logger.Debugf("[%s] Peers for retrieval from orgs %s: %s", d.ChannelID(), orgs, peersForRetrieval)

	return peersForRetrieval
}

func (d *Disseminator) getPeersWithRole(role roles.Role, mspIDs []string) discovery.PeerGroup {
	return d.GetMembers(func(m *discovery.Member) bool {
		if !m.HasRole(role) {
			logger.Debugf("[%s] Not adding peer [%s] since it does not have the role [%s]", d.ChannelID(), m.Endpoint, role)
			return false
		}
		if !contains(mspIDs, m.MSPID) {
			logger.Debugf("[%s] Not adding peer [%s] since it is not in any of the orgs [%s]", d.ChannelID(), m.Endpoint, mspIDs)
			return false
		}
		return true
	})
}

func contains(mspIDs []string, mspID string) bool {
	for _, m := range mspIDs {
		if m == mspID {
			return true
		}
	}
	return false
}

// getMaxPeersForRetrieval may be overridden by unit tests
var getMaxPeersForRetrieval = func() int {
	return config.GetOLCollMaxPeersForRetrieval()
}

// getRandomPeers returns a random set of peers from the given set - up to a maximum of maxPeers
func getRandomPeers(peers discovery.PeerGroup, maxPeers int) discovery.PeerGroup {
	var result discovery.PeerGroup
	for _, index := range rand.Perm(len(peers)) {
		if len(result) == maxPeers {
			break
		}
		result = append(result, peers[index])
	}
	return result
}
