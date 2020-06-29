/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package implicitpolicy

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

func TestResolver(t *testing.T) {
	const org1 = "msp1"

	policy := &mocks.MockAccessPolicy{
		Orgs: []string{ImplicitOrg},
	}

	p := NewResolver(org1, policy)
	require.NotNil(t, p)

	orgs := keys(p.MemberOrgs())
	require.Len(t, orgs, 1)
	require.Equal(t, org1, orgs[0])
}

func keys(m map[string]struct{}) []string {
	var orgs []string
	for org := range m {
		orgs = append(orgs, org)
	}

	return orgs
}
