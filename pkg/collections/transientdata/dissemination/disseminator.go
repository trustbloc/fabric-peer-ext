/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dissemination

import (
	"hash/fnv"
	"sort"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/discovery"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

var logger = flogging.MustGetLogger("transientdata")

// Disseminator disseminates transient data to a deterministic set of endorsers
type Disseminator struct {
	*discovery.Discovery
	namespace  string
	collection string
	policy     privdata.CollectionAccessPolicy
}

// New returns a new transient data disseminator
func New(channelID, namespace, collection string, policy privdata.CollectionAccessPolicy, gossip gossipAdapter) *Disseminator {
	return &Disseminator{
		Discovery:  discovery.New(channelID, gossip),
		namespace:  namespace,
		collection: collection,
		policy:     policy,
	}
}

// ResolveEndorsers resolves to a set of endorsers to which transient data should be disseminated
func (d *Disseminator) ResolveEndorsers(key string) (discovery.PeerGroup, error) {
	h, err := getHash32(key)
	if err != nil {
		return nil, errors.WithMessage(err, "error computing int32 hash of key")
	}

	orgs := d.chooseOrgs(h)

	logger.Debugf("[%s] Chosen orgs: %s", d.ChannelID(), orgs)

	endorsers := d.chooseEndorsers(h, orgs)

	logger.Debugf("[%s] Chosen endorsers from orgs %s: %s", d.ChannelID(), orgs, endorsers)
	return endorsers, nil
}

func (d *Disseminator) chooseEndorsers(h uint32, orgs []string) discovery.PeerGroup {
	var endorsers discovery.PeerGroup

	for i := 0; i < d.policy.MaximumPeerCount(); i++ {
		for _, org := range orgs {
			if len(endorsers) == d.policy.MaximumPeerCount() {
				// We have enough endorsers
				break
			}

			// Get a sorted list of endorsers for the org
			endorsersForOrg := d.getEndorsers(org).Sort()
			if len(endorsersForOrg) == 0 {
				logger.Debugf("[%s] There are no endorsers in org [%s]", d.ChannelID(), org)
				continue
			}

			logger.Debugf("[%s] Endorsers for [%s]: %s", d.ChannelID(), org, endorsersForOrg)

			// Deterministically choose an endorser
			endorserForOrg := endorsersForOrg[(int(h)+i)%len(endorsersForOrg)]
			if endorsers.Contains(endorserForOrg) {
				logger.Debugf("[%s] Will not add endorser [%s] from org [%s] since it is already added", d.ChannelID(), endorserForOrg, org)
				continue
			}

			endorsers = append(endorsers, endorserForOrg)
		}
	}
	return endorsers
}

func (d *Disseminator) getEndorsers(mspID string) discovery.PeerGroup {
	return d.GetMembers(func(m *discovery.Member) bool {
		if m.MSPID != mspID {
			logger.Debugf("[%s] Not adding peer [%s] as an endorser since it is not in org [%s]", d.ChannelID(), m.Endpoint, mspID)
			return false
		}
		if !m.HasRole(roles.EndorserRole) {
			logger.Debugf("[%s] Not adding peer [%s] as an endorser since it does not have the endorser role", d.ChannelID(), m.Endpoint)
			return false
		}
		return true
	})

}

func (d *Disseminator) chooseOrgs(h uint32) []string {
	memberOrgs := d.policy.MemberOrgs()
	numOrgs := min(d.policy.MaximumPeerCount(), len(memberOrgs))

	// Copy and sort the orgs
	var sortedOrgs []string
	sortedOrgs = append(sortedOrgs, memberOrgs...)
	sort.Strings(sortedOrgs)

	var chosenOrgs []string
	for i := 0; i < numOrgs; i++ {
		chosenOrgs = append(chosenOrgs, sortedOrgs[(int(h)+i)%len(sortedOrgs)])
	}

	return chosenOrgs
}

func getHash32(key string) (uint32, error) {
	h := fnv.New32a()
	_, err := h.Write([]byte(key))
	if err != nil {
		return 0, err
	}
	return h.Sum32(), nil
}

func min(i, j int) int {
	if i < j {
		return i
	}
	return j
}
