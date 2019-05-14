/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	gossipapi "github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/comm"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	gossipproto "github.com/hyperledger/fabric/protos/gossip"
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
func (m *MockGossipAdapter) PeersOfChannel(common.ChainID) []discovery.NetworkMember {
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
