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
	if len(orgs) == 0 {
		logger.Warnf("[%s] No orgs for key [%s:%s:%s] using hash32 [%d]", d.ChannelID(), d.namespace, d.collection, key, h)
		return nil, errors.Errorf("no orgs for key [%s:%s:%s]", d.namespace, d.collection, key)
	}

	logger.Debugf("[%s] Chosen orgs for key [%s:%s:%s] using hash32 [%d]: %s", d.ChannelID(), d.namespace, d.collection, key, h, orgs)

	endorsers, err := d.chooseEndorsers(h, orgs)
	if err != nil {
		return nil, err
	}

	logger.Debugf("[%s] Chosen endorsers for key [%s:%s:%s] using hash32 [%d] from orgs %s: %s", d.ChannelID(), d.namespace, d.collection, key, h, orgs, endorsers)
	return endorsers, nil
}

// ResolveAllEndorsersInOrgsForKey resolves all endorsers from within the orgs to which transient data should be disseminated,
// excluding the peers in the given exclude list.
func (d *Disseminator) ResolveAllEndorsersInOrgsForKey(key string, excludePeers ...*discovery.Member) (discovery.PeerGroup, error) {
	h, err := getHash32(key)
	if err != nil {
		return nil, errors.WithMessage(err, "error computing int32 hash of key")
	}

	orgs := d.chooseOrgs(h)
	if len(orgs) == 0 {
		logger.Warnf("[%s] No orgs for key [%s:%s:%s] using hash32 [%d]", d.ChannelID(), d.namespace, d.collection, key, h)

		return nil, errors.Errorf("no orgs for key [%s:%s:%s]", d.namespace, d.collection, key)
	}

	logger.Debugf("[%s] Orgs for key [%s:%s:%s] using hash32 [%d]: %s", d.ChannelID(), d.namespace, d.collection, key, h, orgs)

	endorsers := d.getEndorsers(orgs, excludePeers...)

	logger.Debugf("[%s] Chosen endorsers for key [%s:%s:%s]: %s", d.ChannelID(), d.namespace, d.collection, key, endorsers)

	return endorsers, nil
}

// ResolveAllEndorsers resolves all endorsers that are eligible for the transient data, excluding the peers in the given exclude list.
func (d *Disseminator) ResolveAllEndorsers(excludePeers ...*discovery.Member) (discovery.PeerGroup, error) {
	orgs := keys(d.policy.MemberOrgs())
	if len(orgs) == 0 {
		logger.Warnf("[%s] No orgs for [%s:%s]", d.ChannelID(), d.namespace, d.collection)

		return nil, errors.Errorf("no orgs for [%s:%s]", d.namespace, d.collection)
	}

	logger.Debugf("[%s] Orgs for [%s:%s]: %s", d.ChannelID(), d.namespace, d.collection, orgs)

	endorsers := d.getEndorsers(orgs, excludePeers...)

	logger.Debugf("[%s] Chosen endorsers for [%s:%s]: %s", d.ChannelID(), d.namespace, d.collection, endorsers)

	return endorsers, nil
}

func (d *Disseminator) chooseEndorsers(h uint32, orgs []string) (discovery.PeerGroup, error) {
	allEndorsers := d.getEndorsers(orgs)
	logger.Debugf("[%s] All endorsers for orgs %s: %s", d.ChannelID(), orgs, allEndorsers)

	if len(allEndorsers) == 0 {
		logger.Warnf("[%s] No endorsers for orgs %s", d.ChannelID(), orgs)
		return nil, errors.Errorf("no endorsers for orgs %s", orgs)
	}

	var endorsers discovery.PeerGroup
	for i := 0; len(endorsers) < d.policy.MaximumPeerCount(); i++ {
		// Choose a set of endorsers from the set of orgs, incrementing the hash value
		// so that different peers are chosen each time through the loop
		endorsers = d.appendEndorsersFromOrgs(endorsers, int(h)+i, orgs)
		if len(endorsers) >= len(allEndorsers) {
			logger.Debugf("[%s] Stopping after %d endorsers were chosen using hash32 [%d] since all endorsers from orgs %s have already been added", d.ChannelID(), len(endorsers), h, orgs)
			break
		}
	}

	return endorsers, nil
}

func (d *Disseminator) appendEndorsersFromOrgs(endorsers discovery.PeerGroup, h int, orgs []string) discovery.PeerGroup {
	for _, org := range orgs {
		if len(endorsers) == d.policy.MaximumPeerCount() {
			// We have enough endorsers
			break
		}

		// Get a sorted list of endorsers for the org
		endorsersForOrg := d.getEndorsers([]string{org}).Sort()
		if len(endorsersForOrg) == 0 {
			logger.Debugf("[%s] There are no endorsers in org [%s]", d.ChannelID(), org)
			continue
		}

		logger.Debugf("[%s] Endorsers for [%s] using hash32 [%d]: %s", d.ChannelID(), org, h, endorsersForOrg)

		// Deterministically choose an endorser
		endorserForOrg := endorsersForOrg[h%len(endorsersForOrg)]
		if endorsers.Contains(endorserForOrg) {
			logger.Debugf("[%s] Will not add endorser [%s] from org [%s] using hash32 [%d] since it is already added", d.ChannelID(), endorserForOrg, org, h)
			continue
		}

		endorsers = append(endorsers, endorserForOrg)
	}
	return endorsers
}

func (d *Disseminator) getEndorsers(mspIDs []string, excludePeers ...*discovery.Member) discovery.PeerGroup {
	return d.GetMembers(func(m *discovery.Member) bool {
		if discovery.PeerGroup(excludePeers).Contains(m) {
			logger.Debugf("[%s] Not adding peer [%s] as an endorser since it is in the exclude list %s", d.ChannelID(), m.Endpoint, excludePeers)
			return false
		}

		if !contains(mspIDs, m.MSPID) {
			logger.Debugf("[%s] Not adding peer [%s] as an endorser since it is not in org %s", d.ChannelID(), m.Endpoint, mspIDs)
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
	memberOrgs := keys(d.policy.MemberOrgs())
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

func contains(strs []string, str string) bool {
	for _, s := range strs {
		if s == str {
			return true
		}
	}
	return false
}

func keys(m map[string]struct{}) []string {
	var orgs []string
	for org := range m {
		orgs = append(orgs, org)
	}

	return orgs
}
