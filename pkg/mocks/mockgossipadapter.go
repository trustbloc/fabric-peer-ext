/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	gossipproto "github.com/hyperledger/fabric-protos-go/gossip"
	"github.com/hyperledger/fabric-protos-go/transientstore"
	gossipapi "github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/comm"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/gossip"
	"github.com/hyperledger/fabric/gossip/protoext"
)

// MessageHandler defines a function that handles a gossip message.
type MessageHandler func(msg *gossipproto.GossipMessage)

// MockGossipAdapter is the gossip adapter
type MockGossipAdapter struct {
	self        discovery.NetworkMember
	members     []discovery.NetworkMember
	identitySet gossipapi.PeerIdentitySet
	handler     MessageHandler
}

// NewMockGossipAdapter returns the adapter
func NewMockGossipAdapter() *MockGossipAdapter {
	return &MockGossipAdapter{}
}

// Self discovers a network member
func (m *MockGossipAdapter) Self(mspID string, self discovery.NetworkMember) *MockGossipAdapter {
	m.self = self
	m.identitySet = append(m.identitySet, gossipapi.PeerIdentityInfo{
		PKIId:        self.PKIid,
		Organization: []byte(mspID),
	})
	return m
}

// Member adds the network member
func (m *MockGossipAdapter) Member(mspID string, member discovery.NetworkMember) *MockGossipAdapter {
	m.members = append(m.members, member)
	m.identitySet = append(m.identitySet, gossipapi.PeerIdentityInfo{
		PKIId:        member.PKIid,
		Organization: []byte(mspID),
	})
	return m
}

// MemberWithNoPKIID appends the member
func (m *MockGossipAdapter) MemberWithNoPKIID(mspID string, member discovery.NetworkMember) *MockGossipAdapter {
	m.members = append(m.members, member)
	return m
}

// MessageHandler sets the handler
func (m *MockGossipAdapter) MessageHandler(handler MessageHandler) *MockGossipAdapter {
	m.handler = handler
	return m
}

// PeersOfChannel returns the members
func (m *MockGossipAdapter) PeersOfChannel(common.ChannelID) []discovery.NetworkMember {
	return m.members
}

// SelfMembershipInfo returns self
func (m *MockGossipAdapter) SelfMembershipInfo() discovery.NetworkMember {
	return m.self
}

// IdentityInfo returns the identitySet of this adapter
func (m *MockGossipAdapter) IdentityInfo() gossipapi.PeerIdentitySet {
	return m.identitySet
}

// Send sends a message to remote peers
func (m *MockGossipAdapter) Send(msg *gossipproto.GossipMessage, peers ...*comm.RemotePeer) {
	if m.handler != nil {
		go m.handler(msg)
	}
}

// SelfChannelInfo returns the peer's latest StateInfo message of a given channel
func (m *MockGossipAdapter) SelfChannelInfo(common.ChannelID) *protoext.SignedGossipMessage {
	panic("not implemented")
}

// SendByCriteria sends a given message to all peers that match the given SendCriteria
func (m *MockGossipAdapter) SendByCriteria(*protoext.SignedGossipMessage, gossip.SendCriteria) error {
	panic("not implemented")
}

// Peers returns the NetworkMembers considered alive
func (m *MockGossipAdapter) Peers() []discovery.NetworkMember {
	panic("not implemented")
}

// Gossip sends a message to other peers to the network
func (m *MockGossipAdapter) Gossip(msg *gossipproto.GossipMessage) {
	panic("not implemented")
}

// Accept returns a dedicated read-only channel for messages sent by other nodes that match a certain predicate.
func (m *MockGossipAdapter) Accept(acceptor common.MessageAcceptor, passThrough bool) (<-chan *gossipproto.GossipMessage, <-chan protoext.ReceivedMessage) {
	panic("not implemented")
}

// IsInMyOrg checks whether a network member is in this peer's org
func (m *MockGossipAdapter) IsInMyOrg(member discovery.NetworkMember) bool {
	panic("not implemented")
}

// DistributePrivateData distributes private data to the peers in the collections
// according to policies induced by the PolicyStore and PolicyParser
func (m *MockGossipAdapter) DistributePrivateData(chainID string, txID string, privateData *transientstore.TxPvtReadWriteSetWithConfigInfo, blkHt uint64) error {
	panic("not implemented")
}

// NewMember creates a new network member
func NewMember(endpoint string, pkiID []byte, roles ...string) discovery.NetworkMember {
	return discovery.NetworkMember{
		Endpoint: endpoint,
		PKIid:    pkiID,
		Properties: &gossipproto.Properties{
			Roles: roles,
		},
	}
}
