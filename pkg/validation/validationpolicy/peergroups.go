/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validationpolicy

import (
	"sort"

	"github.com/trustbloc/fabric-peer-ext/pkg/common/discovery"
)

type peerGroups []discovery.PeerGroup

func (g peerGroups) String() string {
	s := "("

	for i, p := range g {
		s += p.String()
		if i+1 < len(g) {
			s += ", "
		}
	}

	s += ")"

	return s
}

func (g peerGroups) sort() {
	// First sort each peer group
	for _, pg := range g {
		pg.Sort()
	}

	// Now sort the peer groups
	sort.Sort(g)
}

// contains returns true if the given peer is contained within any of the peer groups
func (g peerGroups) contains(p *discovery.Member) bool {
	for _, pg := range g {
		if pg.Contains(p) {
			return true
		}
	}
	return false
}

// containsAll returns true if all of the peers within the given peer group are contained within the peer groups
func (g peerGroups) containsAll(peerGroup discovery.PeerGroup) bool {
	for _, p := range peerGroup {
		if !g.contains(p) {
			return false
		}
	}
	return true
}

// containsAny returns true if any of the peers within the given peer group are contained within the peer groups
func (g peerGroups) containsAny(peerGroup discovery.PeerGroup) bool {
	for _, pg := range g {
		if pg.ContainsAny(peerGroup) {
			return true
		}
	}
	return false
}

func (g peerGroups) Len() int {
	return len(g)
}

func (g peerGroups) Less(i, j int) bool {
	return g[i].String() < g[j].String()
}

func (g peerGroups) Swap(i, j int) {
	g[i], g[j] = g[j], g[i]
}
