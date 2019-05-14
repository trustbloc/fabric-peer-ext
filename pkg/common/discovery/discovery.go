/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package discovery

import (
	"sync"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	proto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

var logger = flogging.MustGetLogger("discovery")

// Discovery provides functions to retrieve info about the local peer and other peers in a given channel
type Discovery struct {
	channelID string
	gossip    gossipAdapter
	self      *Member
	selfInit  sync.Once
}

// New returns a new Discovery
func New(channelID string, gossip gossipAdapter) *Discovery {
	return &Discovery{
		channelID: channelID,
		gossip:    gossip,
	}
}

type filter func(m *Member) bool

type gossipAdapter interface {
	PeersOfChannel(common.ChainID) []discovery.NetworkMember
	SelfMembershipInfo() discovery.NetworkMember
	IdentityInfo() api.PeerIdentitySet
}

// Self returns the local peer
func (r *Discovery) Self() *Member {
	r.selfInit.Do(func() {
		r.self = getSelf(r.channelID, r.gossip)
	})
	return r.self
}

// ChannelID returns the channel ID
func (r *Discovery) ChannelID() string {
	return r.channelID
}

// GetMembers returns members filtered by the given filter
func (r *Discovery) GetMembers(accept filter) []*Member {
	identityInfo := r.gossip.IdentityInfo()
	mapByID := identityInfo.ByID()

	var peers []*Member
	for _, m := range r.gossip.PeersOfChannel(common.ChainID(r.channelID)) {
		identity, ok := mapByID[string(m.PKIid)]
		if !ok {
			logger.Warningf("[%s] Not adding peer [%s] as a validator since unable to determine MSP ID from PKIID for [%s]", r.channelID, m.Endpoint)
			continue
		}

		m := &Member{
			NetworkMember: m,
			MSPID:         string(identity.Organization),
		}

		if accept(m) {
			peers = append(peers, m)
		}
	}

	if accept(r.Self()) {
		peers = append(peers, r.Self())
	}

	return peers
}

// GetMSPID gets the MSP id
func (r *Discovery) GetMSPID(pkiID common.PKIidType) (string, bool) {
	identityInfo := r.gossip.IdentityInfo()
	mapByID := identityInfo.ByID()

	identity, ok := mapByID[string(pkiID)]
	if !ok {
		return "", false
	}
	return string(identity.Organization), true
}

func getSelf(channelID string, gossip gossipAdapter) *Member {
	self := gossip.SelfMembershipInfo()
	self.Properties = &proto.Properties{
		Roles: roles.AsString(),
	}

	identityInfo := gossip.IdentityInfo()
	mapByID := identityInfo.ByID()

	var mspID string
	selfIdentity, ok := mapByID[string(self.PKIid)]
	if ok {
		mspID = string(selfIdentity.Organization)
	} else {
		logger.Warningf("[%s] Unable to determine MSP ID from PKIID for self", channelID)
	}

	return &Member{
		NetworkMember: self,
		MSPID:         mspID,
		Local:         true,
	}
}
