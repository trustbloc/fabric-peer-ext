/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package discovery

import (
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

// Member wraps a NetworkMember and provides additional info
type Member struct {
	discovery.NetworkMember
	ChannelID string
	MSPID     string
	Local     bool // Indicates whether this member is the local peer
}

func (m *Member) String() string {
	return m.Endpoint
}

// Roles returns the roles of the peer
func (m *Member) Roles() roles.Roles {
	if m.Properties == nil {
		logger.Debugf("[%s] Peer [%s] does not have any properties", m.ChannelID, m.Endpoint)
		return nil
	}
	return roles.FromStrings(m.Properties.Roles...)
}

// HasRole returns true if the member has the given role
func (m *Member) HasRole(role roles.Role) bool {
	return m.Roles().Contains(role)
}
