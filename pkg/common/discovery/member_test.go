/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package discovery

import (
	"testing"

	proto "github.com/hyperledger/fabric-protos-go/gossip"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/stretchr/testify/assert"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

const (
	org1MSP = "org1MSP"

	p0Endpoint = "p0.org1.com"
	p1Endpoint = "p1.org1.com"
	p2Endpoint = "p2.org1.com"
	p3Endpoint = "p3.org1.com"
	p4Endpoint = "p4.org1.com"
	p5Endpoint = "p5.org1.com"
	p6Endpoint = "p6.org1.com"
	p7Endpoint = "p7.org1.com"

	r1 = "role1"
	r2 = "role2"
	r3 = "role3"
	r4 = "role4"

	role1 = roles.Role(r1)
	role2 = roles.Role(r2)
	role3 = roles.Role(r3)
	role4 = roles.Role(r4)
)

var (
	p1 = newMember(org1MSP, p1Endpoint, false)
	p2 = newMember(org1MSP, p2Endpoint, false, r2, r3, r4)
	p3 = newMember(org1MSP, p3Endpoint, false)
	p4 = newMember(org1MSP, p4Endpoint, false)
	p5 = newMember(org1MSP, p5Endpoint, true)
	p6 = newMember(org1MSP, p6Endpoint, false)
	p7 = newMember(org1MSP, p7Endpoint, false)
)

func TestMember_HasRole(t *testing.T) {
	assert.True(t, p1.HasRole(role1))
	assert.False(t, p2.HasRole(role1))
	assert.True(t, p2.HasRole(role2))
}

func TestMember_Roles(t *testing.T) {
	assert.Empty(t, p1.Roles())

	roles := p2.Roles()
	assert.Equal(t, role2, roles[0])
	assert.Equal(t, role3, roles[1])
	assert.Equal(t, role4, roles[2])
}

func TestMember_String(t *testing.T) {
	assert.Equal(t, p1.Endpoint, p1.String())
}
func newMember(mspID, endpoint string, local bool, roles ...string) *Member {
	m := &Member{
		NetworkMember: discovery.NetworkMember{
			Endpoint: endpoint,
		},
		MSPID: mspID,
		Local: local,
	}
	if roles != nil {
		m.Properties = &proto.Properties{
			Roles: roles,
		}
	}
	return m
}
