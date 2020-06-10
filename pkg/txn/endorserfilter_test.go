/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package txn

import (
	"testing"

	gcommon "github.com/hyperledger/fabric/gossip/common"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"

	"github.com/trustbloc/fabric-peer-ext/pkg/common/discovery"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
	txnmocks "github.com/trustbloc/fabric-peer-ext/pkg/txn/mocks"
)

const (
	channel1   = "channel1"
	org1MSPID  = "Org1MSP"
	p1Endpoint = "p1.org1.com:7051"
	p2Endpoint = "p2.org1.com:7051"
	p3Endpoint = "p3.org1.com:7051"

	committerRole = string(roles.CommitterRole)
	endorserRole  = string(roles.EndorserRole)
)

var (
	p1PKIID = gcommon.PKIidType("pkiid_P1O1")
	p2PKIID = gcommon.PKIidType("pkiid_P2O1")
	p3PKIID = gcommon.PKIidType("pkiid_P3O1")
)

func TestEndorserFilter(t *testing.T) {
	restoreRoles := setRoles(roles.EndorserRole)
	defer restoreRoles()

	gossip := mocks.NewMockGossipAdapter().
		Self(org1MSPID, mocks.NewMember(p1Endpoint, p1PKIID)).
		Member(org1MSPID, mocks.NewMember(p2Endpoint, p2PKIID, committerRole)).
		Member(org1MSPID, mocks.NewMember(p3Endpoint, p3PKIID, endorserRole))

	d := discovery.New(channel1, gossip)
	f := newEndorserFilter(d, nil)
	require.NotNil(t, f)

	p1 := &txnmocks.Peer{}
	p1.EndpointReturns(p1Endpoint)

	p2 := &txnmocks.Peer{}
	p2.EndpointReturns(p2Endpoint)

	p3 := &txnmocks.Peer{}
	p3.EndpointReturns(p3Endpoint)

	require.True(t, f.Accept(p1))
	require.False(t, f.Accept(p2))
	require.True(t, f.Accept(p3))
}

func setRoles(rls ...roles.Role) (reset func()) {
	rolesValue := make(map[roles.Role]struct{})
	for _, r := range rls {
		rolesValue[r] = struct{}{}
	}
	roles.SetRoles(rolesValue)
	return func() { roles.SetRoles(nil) }
}
