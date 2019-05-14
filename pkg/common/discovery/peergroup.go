/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package discovery

import (
	"math/rand"
	"sort"
)

// PeerGroup is a group of peers
type PeerGroup []*Member

// ContainsLocal return true if one of the peers in the group is the local peer
func (g PeerGroup) ContainsLocal() bool {
	for _, p := range g {
		if p.Local {
			return true
		}
	}
	return false
}

func (g PeerGroup) String() string {
	s := "["
	for i, p := range g {
		s += p.String()
		if i+1 < len(g) {
			s += ", "
		}
	}
	s += "]"
	return s
}

// Sort sorts the peer group by endpoint
func (g PeerGroup) Sort() PeerGroup {
	sort.Sort(g)
	return g
}

// ContainsAll returns true if ALL of the peers within the given peer group are contained within this peer group
func (g PeerGroup) ContainsAll(peerGroup PeerGroup) bool {
	for _, p := range peerGroup {
		if !g.Contains(p) {
			return false
		}
	}
	return true
}

// ContainsAny returns true if ANY of the peers within the given peer group are contained within this peer group
func (g PeerGroup) ContainsAny(peerGroup PeerGroup) bool {
	for _, p := range peerGroup {
		if g.Contains(p) {
			return true
		}
	}
	return false
}

// Contains returns true if the given peer is contained within this peer group
func (g PeerGroup) Contains(peer *Member) bool {
	for _, p := range g {
		if p.Endpoint == peer.Endpoint {
			return true
		}
	}
	return false
}

// ContainsPeer returns true if the given peer is contained within this peer group
func (g PeerGroup) ContainsPeer(endpoint string) bool {
	for _, p := range g {
		if p.Endpoint == endpoint {
			return true
		}
	}
	return false
}

func (g PeerGroup) Len() int {
	return len(g)
}

func (g PeerGroup) Less(i, j int) bool {
	return g[i].Endpoint < g[j].Endpoint
}

func (g PeerGroup) Swap(i, j int) {
	g[i], g[j] = g[j], g[i]
}

// Local returns the local peer from the group
func (g PeerGroup) Local() (*Member, bool) {
	for _, p := range g {
		if p.Local {
			return p, true
		}
	}
	return nil, false
}

// Remote returns a PeerGroup subset that only includes remote peers
func (g PeerGroup) Remote() PeerGroup {
	var pg PeerGroup
	for _, p := range g {
		if !p.Local {
			pg = append(pg, p)
		}
	}
	return pg
}

// Shuffle returns a randomly shuffled PeerGroup
func (g PeerGroup) Shuffle() PeerGroup {
	var pg PeerGroup
	for _, i := range rand.Perm(len(g)) {
		pg = append(pg, g[i])
	}

	return pg
}

// Merge adds the given peers to the group if they don't already exist
func Merge(g PeerGroup, peers ...*Member) PeerGroup {
	for _, p := range peers {
		if !g.Contains(p) {
			g = append(g, p)
		}
	}
	return g
}
