/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dissemination

import (
	"os"
	"testing"

	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/extensions/collections/api/dissemination"
	gcommon "github.com/hyperledger/fabric/gossip/common"
	gdiscovery "github.com/hyperledger/fabric/gossip/discovery"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

var (
	ns1   = "chaincode1"
	ns2   = "chaincode2"
	coll1 = "collection1"
	coll2 = "collection2"
	key1  = "key1"
	key2  = "key2"
	key6  = "key6"
	tx1   = "tx1"

	org1MSPID      = "Org1MSP"
	p1Org1Endpoint = "p1.org1.com"
	p1Org1PKIID    = gcommon.PKIidType("pkiid_P1O1")
	p2Org1Endpoint = "p2.org1.com"
	p2Org1PKIID    = gcommon.PKIidType("pkiid_P2O1")
	p3Org1Endpoint = "p3.org1.com"
	p3Org1PKIID    = gcommon.PKIidType("pkiid_P3O1")

	org2MSPID      = "Org2MSP"
	p1Org2Endpoint = "p1.org2.com"
	p1Org2PKIID    = gcommon.PKIidType("pkiid_P1O2")
	p2Org2Endpoint = "p2.org2.com"
	p2Org2PKIID    = gcommon.PKIidType("pkiid_P2O2")
	p3Org2Endpoint = "p3.org2.com"
	p3Org2PKIID    = gcommon.PKIidType("pkiid_P3O2")

	org3MSPID      = "Org3MSP"
	p1Org3Endpoint = "p1.org3.com"
	p1Org3PKIID    = gcommon.PKIidType("pkiid_P1O3")
	p2Org3Endpoint = "p2.org3.com"
	p2Org3PKIID    = gcommon.PKIidType("pkiid_P2O3")
	p3Org3Endpoint = "p3.org3.com"
	p3Org3PKIID    = gcommon.PKIidType("pkiid_P3O3")

	org4MSPID = "Org4MSP"

	validatorRole = string(roles.ValidatorRole)
	endorserRole  = string(roles.EndorserRole)
)

func TestDissemination(t *testing.T) {
	channelID := "testchannel"

	gossip := mocks.NewMockGossipAdapter().
		Self(org1MSPID, mocks.NewMember(p1Org1Endpoint, p1Org1PKIID)).
		Member(org1MSPID, mocks.NewMember(p2Org1Endpoint, p2Org1PKIID, validatorRole)).
		Member(org1MSPID, mocks.NewMember(p3Org1Endpoint, p3Org1PKIID, validatorRole)).
		Member(org2MSPID, mocks.NewMember(p1Org2Endpoint, p1Org2PKIID, endorserRole)).
		Member(org2MSPID, mocks.NewMember(p2Org2Endpoint, p2Org2PKIID, validatorRole)).
		Member(org2MSPID, mocks.NewMember(p3Org2Endpoint, p3Org2PKIID, endorserRole)).
		Member(org3MSPID, mocks.NewMember(p1Org3Endpoint, p1Org3PKIID, endorserRole)).
		Member(org3MSPID, mocks.NewMember(p2Org3Endpoint, p2Org3PKIID, validatorRole)).
		Member(org3MSPID, mocks.NewMember(p3Org3Endpoint, p3Org3PKIID, endorserRole))

	t.Run("2 peers", func(t *testing.T) {
		maxPeers := 2

		d := New(channelID, ns1, coll1,
			&mocks.MockAccessPolicy{
				ReqPeerCount: 1,
				MaxPeerCount: maxPeers,
				Orgs:         []string{org1MSPID, org2MSPID, org3MSPID},
			}, gossip)

		// key1
		endorsers, err := d.ResolveEndorsers(key1)
		require.NoError(t, err)
		require.Equal(t, maxPeers, len(endorsers))

		t.Logf("Endorsers: %s", endorsers)

		assert.Equal(t, p3Org3Endpoint, endorsers[0].Endpoint)
		assert.Equal(t, p1Org1Endpoint, endorsers[1].Endpoint)

		// key2
		endorsers, err = d.ResolveEndorsers(key2)
		require.NoError(t, err)
		require.Equal(t, maxPeers, len(endorsers))

		t.Logf("Endorsers: %s", endorsers)

		assert.Equal(t, p1Org2Endpoint, endorsers[0].Endpoint)
		assert.Equal(t, p1Org3Endpoint, endorsers[1].Endpoint)
	})

	t.Run("5 peers", func(t *testing.T) {
		maxPeers := 5

		d := New(channelID, ns1, coll1,
			&mocks.MockAccessPolicy{
				ReqPeerCount: 1,
				MaxPeerCount: maxPeers,
				Orgs:         []string{org1MSPID, org2MSPID, org3MSPID},
			}, gossip)

		// key1
		endorsers, err := d.ResolveEndorsers(key1)
		require.NoError(t, err)
		require.Equal(t, maxPeers, len(endorsers))

		t.Logf("Endorsers: %s", endorsers)

		assert.Equal(t, p3Org3Endpoint, endorsers[0].Endpoint)
		assert.Equal(t, p1Org1Endpoint, endorsers[1].Endpoint)
		assert.Equal(t, p3Org2Endpoint, endorsers[2].Endpoint)
		assert.Equal(t, p1Org3Endpoint, endorsers[3].Endpoint)
		assert.Equal(t, p1Org2Endpoint, endorsers[4].Endpoint)
	})

	t.Run("Not enough peers", func(t *testing.T) {
		maxPeers := 6

		d := New(channelID, ns1, coll1,
			&mocks.MockAccessPolicy{
				ReqPeerCount: 1,
				MaxPeerCount: maxPeers,
				Orgs:         []string{org1MSPID, org2MSPID, org3MSPID},
			}, gossip)

		// key1
		endorsers, err := d.ResolveEndorsers(key1)
		require.NoError(t, err)
		require.Equal(t, 5, len(endorsers))

		t.Logf("Endorsers: %s", endorsers)

		assert.Equal(t, p3Org3Endpoint, endorsers[0].Endpoint)
		assert.Equal(t, p1Org1Endpoint, endorsers[1].Endpoint)
		assert.Equal(t, p3Org2Endpoint, endorsers[2].Endpoint)
		assert.Equal(t, p1Org3Endpoint, endorsers[3].Endpoint)
		assert.Equal(t, p1Org2Endpoint, endorsers[4].Endpoint)
	})

	t.Run("1 org", func(t *testing.T) {
		maxPeers := 3

		d := New(channelID, ns1, coll1,
			&mocks.MockAccessPolicy{
				MaxPeerCount: maxPeers,
				Orgs:         []string{org3MSPID},
			}, gossip)

		// key1
		endorsers, err := d.ResolveEndorsers(key1)
		require.NoError(t, err)
		require.Equal(t, 2, len(endorsers))

		t.Logf("Endorsers: %s", endorsers)

		assert.Equal(t, p3Org3Endpoint, endorsers[0].Endpoint)
		assert.Equal(t, p1Org3Endpoint, endorsers[1].Endpoint)
	})

	t.Run("Subset of orgs", func(t *testing.T) {
		maxPeers := 3

		d := New(channelID, ns1, coll1,
			&mocks.MockAccessPolicy{
				ReqPeerCount: 1,
				MaxPeerCount: maxPeers,
				Orgs:         []string{org2MSPID, org3MSPID},
			}, gossip)

		// key1
		endorsers, err := d.ResolveEndorsers(key1)
		require.NoError(t, err)
		require.Equal(t, maxPeers, len(endorsers))

		t.Logf("Endorsers: %s", endorsers)

		assert.Equal(t, p3Org3Endpoint, endorsers[0].Endpoint)
		assert.Equal(t, p3Org2Endpoint, endorsers[1].Endpoint)
		assert.Equal(t, p1Org3Endpoint, endorsers[2].Endpoint)
	})

	t.Run("No peers", func(t *testing.T) {
		maxPeers := 6

		d := New(channelID, ns1, coll1,
			&mocks.MockAccessPolicy{
				MaxPeerCount: maxPeers,
				Orgs:         []string{org4MSPID},
			}, gossip)

		endorsers, err := d.ResolveEndorsers(key1)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no endorsers")
		require.Empty(t, endorsers)
	})

	t.Run("No orgs", func(t *testing.T) {
		maxPeers := 2

		d := New(channelID, ns1, coll1,
			&mocks.MockAccessPolicy{
				MaxPeerCount: maxPeers,
				Orgs:         []string{},
			}, gossip)

		endorsers, err := d.ResolveEndorsers(key1)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no orgs")
		require.Empty(t, endorsers)
	})

	t.Run("Org with no peers", func(t *testing.T) {
		maxPeers := 4

		d := New(channelID, ns1, coll1,
			&mocks.MockAccessPolicy{
				MaxPeerCount: maxPeers,
				Orgs:         []string{org1MSPID, org2MSPID, org3MSPID, org4MSPID}, // org4 has no peers
			}, gossip)

		// key1
		endorsers, err := d.ResolveEndorsers(key2)
		require.NoError(t, err)
		require.Equal(t, maxPeers, len(endorsers))

		t.Logf("Endorsers: %s", endorsers)

		assert.Equal(t, p1Org3Endpoint, endorsers[0].Endpoint)
		assert.Equal(t, p1Org1Endpoint, endorsers[1].Endpoint)
		assert.Equal(t, p1Org2Endpoint, endorsers[2].Endpoint)
		assert.Equal(t, p3Org3Endpoint, endorsers[3].Endpoint)
	})
}

func TestComputeDisseminationPlan(t *testing.T) {
	channelID := "testchannel"

	p1Org1 := mocks.NewMember(p1Org1Endpoint, p1Org1PKIID, endorserRole)
	p2Org1 := mocks.NewMember(p2Org1Endpoint, p2Org1PKIID, endorserRole)
	p3Org1 := mocks.NewMember(p3Org1Endpoint, p3Org1PKIID, validatorRole)
	p1Org2 := mocks.NewMember(p1Org2Endpoint, p1Org2PKIID, endorserRole)
	p2Org2 := mocks.NewMember(p2Org2Endpoint, p2Org2PKIID, validatorRole)
	p3Org2 := mocks.NewMember(p3Org2Endpoint, p3Org2PKIID, endorserRole)
	p1Org3 := mocks.NewMember(p1Org3Endpoint, p1Org3PKIID, endorserRole)
	p2Org3 := mocks.NewMember(p2Org3Endpoint, p2Org3PKIID, validatorRole)
	p3Org3 := mocks.NewMember(p3Org3Endpoint, p3Org3PKIID, endorserRole)

	gossip := mocks.NewMockGossipAdapter().
		Self(org1MSPID, p1Org1).
		Member(org1MSPID, p2Org1).
		Member(org1MSPID, p3Org1).
		Member(org2MSPID, p1Org2).
		Member(org2MSPID, p2Org2).
		Member(org2MSPID, p3Org2).
		Member(org3MSPID, p1Org3).
		Member(org3MSPID, p2Org3).
		Member(org3MSPID, p3Org3)

	allPeers := []gdiscovery.NetworkMember{p1Org1, p2Org1, p3Org1, p1Org2, p2Org2, p3Org2, p1Org3, p2Org3, p3Org3}

	t.Run("Orgs: 2, Max Peers: 2, Keys: 1", func(t *testing.T) {
		maxPeers := 2

		colAP := &mocks.MockAccessPolicy{
			ReqPeerCount: 1,
			MaxPeerCount: maxPeers,
			Orgs:         []string{org2MSPID, org3MSPID},
		}

		coll1Builder := mocks.NewPvtReadWriteSetCollectionBuilder(coll1)
		coll1Builder.
			Write(key1, []byte("value1")).
			Delete(key2) // Deletes should be ignored

		rwSet := coll1Builder.Build()
		pvtDataMsg, err := createPrivateDataMessage(channelID, tx1, ns1, rwSet, &pb.CollectionConfigPackage{}, 1000)
		require.NoError(t, err)

		dPlan, handled, err := ComputeDisseminationPlan(channelID, ns1, rwSet, colAP, pvtDataMsg, gossip)
		require.NoError(t, err)
		require.True(t, handled)
		require.Equal(t, 2, len(dPlan))

		eligibiltyMap := getEligibilityMap(t, dPlan, allPeers)

		assert.False(t, eligibiltyMap[p1Org1.Endpoint])
		assert.False(t, eligibiltyMap[p2Org1.Endpoint])
		assert.False(t, eligibiltyMap[p3Org1.Endpoint])
		assert.False(t, eligibiltyMap[p1Org2.Endpoint])
		assert.False(t, eligibiltyMap[p2Org2.Endpoint])
		assert.True(t, eligibiltyMap[p3Org2.Endpoint])
		assert.False(t, eligibiltyMap[p1Org3.Endpoint])
		assert.False(t, eligibiltyMap[p2Org3.Endpoint])
		assert.True(t, eligibiltyMap[p3Org3.Endpoint])
	})

	t.Run("Orgs: 3, Max Peers: 3, Keys: 1", func(t *testing.T) {
		maxPeers := 3

		colAP := &mocks.MockAccessPolicy{
			ReqPeerCount: 1,
			MaxPeerCount: maxPeers,
			Orgs:         []string{org2MSPID, org3MSPID},
		}

		coll1Builder := mocks.NewPvtReadWriteSetCollectionBuilder(coll1)
		coll1Builder.
			Write(key1, []byte("value1"))

		rwSet := coll1Builder.Build()
		pvtDataMsg, err := createPrivateDataMessage(channelID, tx1, ns1, rwSet, &pb.CollectionConfigPackage{}, 1000)
		require.NoError(t, err)

		dPlan, handled, err := ComputeDisseminationPlan(channelID, ns1, rwSet, colAP, pvtDataMsg, gossip)
		require.NoError(t, err)
		require.True(t, handled)
		require.Equal(t, 3, len(dPlan))

		eligibiltyMap := getEligibilityMap(t, dPlan, allPeers)

		// The transient data should be stored to p1Org1 and p3Org3 but, since p1Org1 is
		// a local peer, it is not included in the dissemination plan.
		assert.False(t, eligibiltyMap[p1Org1.Endpoint])
		assert.False(t, eligibiltyMap[p2Org1.Endpoint])
		assert.False(t, eligibiltyMap[p3Org1.Endpoint])
		assert.False(t, eligibiltyMap[p1Org2.Endpoint])
		assert.False(t, eligibiltyMap[p2Org2.Endpoint])
		assert.True(t, eligibiltyMap[p3Org2.Endpoint])
		assert.True(t, eligibiltyMap[p1Org3.Endpoint])
		assert.False(t, eligibiltyMap[p2Org3.Endpoint])
		assert.True(t, eligibiltyMap[p3Org3.Endpoint])
	})

	t.Run("Orgs: 3, Max Peers: 2, Keys: 2", func(t *testing.T) {
		maxPeers := 2

		colAP := &mocks.MockAccessPolicy{
			ReqPeerCount: 1,
			MaxPeerCount: maxPeers,
			Orgs:         []string{org1MSPID, org2MSPID, org3MSPID},
		}

		coll1Builder := mocks.NewPvtReadWriteSetCollectionBuilder(coll1)
		coll1Builder.
			Write(key1, []byte("value1")).
			Write(key2, []byte("value2")).
			Write(key6, []byte("value6"))

		rwSet := coll1Builder.Build()
		pvtDataMsg, err := createPrivateDataMessage(channelID, tx1, ns1, rwSet, &pb.CollectionConfigPackage{}, 1000)
		require.NoError(t, err)

		dPlan, handled, err := ComputeDisseminationPlan(channelID, ns1, rwSet, colAP, pvtDataMsg, gossip)
		require.NoError(t, err)
		require.True(t, handled)
		require.Equal(t, 4, len(dPlan))

		eligibiltyMap := getEligibilityMap(t, dPlan, allPeers)

		// The transient data for:
		// - key1: p2Org1, p3Org3
		// - key2: p1Org2, p1Org3
		// - key6: p1Org1, p1Org3
		// Since p1Org1 is a local peer, it's not included in the dissemination plan.
		// So the data should be disseminated to p2Org1, p1Org2, p1Org3, and p3Org3
		assert.False(t, eligibiltyMap[p1Org1.Endpoint])
		assert.True(t, eligibiltyMap[p2Org1.Endpoint])
		assert.False(t, eligibiltyMap[p3Org1.Endpoint])
		assert.True(t, eligibiltyMap[p1Org2.Endpoint])
		assert.False(t, eligibiltyMap[p2Org2.Endpoint])
		assert.False(t, eligibiltyMap[p3Org2.Endpoint])
		assert.True(t, eligibiltyMap[p1Org3.Endpoint])
		assert.False(t, eligibiltyMap[p2Org3.Endpoint])
		assert.True(t, eligibiltyMap[p3Org3.Endpoint])
	})

	t.Run("No endorsers error", func(t *testing.T) {
		maxPeers := 2

		colAP := &mocks.MockAccessPolicy{
			MaxPeerCount: maxPeers,
			Orgs:         []string{org4MSPID},
		}

		coll1Builder := mocks.NewPvtReadWriteSetCollectionBuilder(coll1)
		coll1Builder.Write(key1, []byte("value1"))

		rwSet := coll1Builder.Build()
		pvtDataMsg, err := createPrivateDataMessage(channelID, tx1, ns1, rwSet, &pb.CollectionConfigPackage{}, 1000)
		require.NoError(t, err)

		dPlan, handled, err := ComputeDisseminationPlan(channelID, ns1, rwSet, colAP, pvtDataMsg, gossip)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no endorsers")
		require.True(t, handled)
		require.Empty(t, dPlan)
	})
}

func TestMain(m *testing.M) {
	viper.SetDefault("ledger.roles", "committer,endorser")

	os.Exit(m.Run())
}

func getEligibilityMap(t *testing.T, dPlan []*dissemination.Plan, allPeers []gdiscovery.NetworkMember) map[string]bool {
	eligibiltyMap := make(map[string]bool)
	for _, plan := range dPlan {
		criteria := plan.Criteria
		require.Equal(t, 1, criteria.MinAck)
		require.Equal(t, 1, criteria.MaxPeers)

		for _, p := range allPeers {
			if !eligibiltyMap[p.Endpoint] {
				eligibiltyMap[p.Endpoint] = criteria.IsEligible(p)
			}
		}
	}
	return eligibiltyMap
}
