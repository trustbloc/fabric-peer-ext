/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPeerGroup_String(t *testing.T) {
	pg := PeerGroup{p3, p1, p6, p5}
	assert.Equal(t, "[p3.org1.com, p1.org1.com, p6.org1.com, p5.org1.com]", pg.String())
}

func TestPeerGroup_Sort(t *testing.T) {
	pg := PeerGroup{p3, p1, p6, p5}
	pg.Sort()

	assert.Equal(t, p1, pg[0])
	assert.Equal(t, p3, pg[1])
	assert.Equal(t, p5, pg[2])
	assert.Equal(t, p6, pg[3])
}

func TestPeerGroup_Contains(t *testing.T) {
	pg1 := PeerGroup{p1, p2, p3}
	pg2 := PeerGroup{p2, p3}
	pg3 := PeerGroup{p2, p3, p4}
	pg4 := PeerGroup{p4, p5}

	assert.True(t, pg1.Contains(p1))
	assert.True(t, pg1.Contains(p2))
	assert.False(t, pg1.Contains(p4))
	assert.True(t, pg1.ContainsAll(pg2))
	assert.False(t, pg1.ContainsAll(pg3))
	assert.True(t, pg1.ContainsAny(pg3))
	assert.False(t, pg1.ContainsAny(pg4))
}

func TestPeerGroup_ContainsLocal(t *testing.T) {
	pLocal := newMember(org1MSP, p0Endpoint, true)

	pg1 := PeerGroup{p1, p2, p3, pLocal}
	pg2 := PeerGroup{p2, p3, p4}

	assert.True(t, pg1.ContainsLocal())
	assert.False(t, pg2.ContainsLocal())
}

func TestPeerGroup_ContainsPeer(t *testing.T) {
	pg := PeerGroup{p1, p2, p3}
	assert.False(t, pg.ContainsPeer(p0Endpoint))
	assert.True(t, pg.ContainsPeer(p1Endpoint))
	assert.True(t, pg.ContainsPeer(p2Endpoint))
	assert.True(t, pg.ContainsPeer(p3Endpoint))
}

func TestPeerGroup_Local(t *testing.T) {
	pg := PeerGroup{p1, p2, p3, p4, p5, p6}
	p, ok := pg.Local()
	assert.True(t, ok)
	assert.Equal(t, p5, p)

	pg = PeerGroup{p1, p2, p3, p4, p6}
	p, ok = pg.Local()
	assert.False(t, ok)
	assert.Nil(t, p)
}

func TestPeerGroup_Remote(t *testing.T) {
	pg := PeerGroup{p1, p2, p3, p4, p5, p6}
	for _, pg := range pg.Remote() {
		assert.False(t, pg.Local)
	}
}

func TestMerge(t *testing.T) {
	pg1 := PeerGroup{p1, p2, p3}
	pg := Merge(pg1, p2, p3, p4)
	assert.Equal(t, p1, pg[0])
	assert.Equal(t, p2, pg[1])
	assert.Equal(t, p3, pg[2])
	assert.Equal(t, p4, pg[3])
}

func TestPeerGroup_Shuffle(t *testing.T) {
	pg := PeerGroup{p1, p2, p3, p4, p5, p6}

	chosen := make(map[string]struct{})

	for i := 0; i < 10; i++ {
		pgShuffled := pg.Shuffle()
		assert.True(t, pgShuffled.ContainsAll(pg))
		chosen[pgShuffled[0].Endpoint] = struct{}{}
	}

	assert.Truef(t, len(chosen) > 1, "Expecting that the first peer is not always the same one.")
}
