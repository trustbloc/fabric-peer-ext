/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/protoutil"
)

// MockAccessPolicy implements a mock CollectionAccessPolicy
type MockAccessPolicy struct {
	ReqPeerCount int
	MaxPeerCount int
	Orgs         []string
	OnlyRead     bool
	OnlyWrite    bool
	Filter       privdata.Filter
}

// AccessFilter returns a member filter function for a collection
func (m *MockAccessPolicy) AccessFilter() privdata.Filter {
	if m.Filter == nil {
		return func(protoutil.SignedData) bool { return true }
	}
	return m.Filter
}

// RequiredPeerCount The minimum number of peers private data will be sent to upon
// endorsement. The endorsement would fail if dissemination to at least
// this number of peers is not achieved.
func (m *MockAccessPolicy) RequiredPeerCount() int {
	return m.ReqPeerCount
}

// MaximumPeerCount The maximum number of peers that private data will be sent to
// upon endorsement. This number has to be bigger than RequiredPeerCount().
func (m *MockAccessPolicy) MaximumPeerCount() int {
	return m.MaxPeerCount
}

// MemberOrgs returns the collection's members as MSP IDs. This serves as
// a human-readable way of quickly identifying who is part of a collection.
func (m *MockAccessPolicy) MemberOrgs() []string {
	return m.Orgs
}

// IsMemberOnlyRead returns a true if only collection members can read
// the private data
func (m *MockAccessPolicy) IsMemberOnlyRead() bool {
	return m.OnlyRead
}

// IsMemberOnlyWrite returns a true if only collection members can write
// the private data
func (m *MockAccessPolicy) IsMemberOnlyWrite() bool {
	return m.OnlyWrite
}
