/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package implicitpolicy

import (
	"github.com/hyperledger/fabric/core/common/privdata"
)

const (
	// ImplicitOrg is used in the collection policy to indicate that the
	// collection policy is for the local peer's MSP
	ImplicitOrg = "IMPLICIT-ORG"
)

// Resolver wraps a collection policy and resolves any occurrence of 'IMPLICIT-ORG' to the local MSP
type Resolver struct {
	privdata.CollectionAccessPolicy
	localMSP string
}

// NewResolver returns a new implicit collection policy resolver
func NewResolver(localMSP string, policy privdata.CollectionAccessPolicy) *Resolver {
	return &Resolver{
		localMSP:               localMSP,
		CollectionAccessPolicy: policy,
	}
}

// MemberOrgs returns the collection's members as MSP IDs. If any of the orgs is set to 'IMPLICIT-ORG' then it
// is replaced with the ID of the local MSP.
func (p *Resolver) MemberOrgs() map[string]struct{} {
	memberOrgs := p.CollectionAccessPolicy.MemberOrgs()

	orgs := make(map[string]struct{})

	for org := range memberOrgs {
		if org == ImplicitOrg {
			org = p.localMSP
		}

		orgs[org] = struct{}{}
	}

	return orgs
}
