/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package authfilter

import (
	"context"
	"testing"

	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/resource"
)

const (
	filter1 = "filter1"
	filter2 = "filter2"
)

func TestAuthFilter(t *testing.T) {
	require.Nil(t, Get(filter1))
	require.Nil(t, Get(filter2))

	Register(newMockAuthFilter1)
	Register(newMockAuthFilter2)

	require.NoError(t, resource.Mgr.Initialize())

	require.NotNil(t, Get(filter1))
	require.NotNil(t, Get(filter2))
	require.Nil(t, Get("unknown"))
}

func TestAuthFilterAlreadyRegistered(t *testing.T) {
	r := newRegistry()
	r.register(newMockAuthFilter1())

	require.Panics(t, func() {
		r.register(newMockAuthFilter1())
	})
}

func newMockAuthFilter1() *mockAuthFilter {
	return &mockAuthFilter{name: filter1}
}

func newMockAuthFilter2() *mockAuthFilter {
	return &mockAuthFilter{name: filter2}
}

type mockAuthFilter struct {
	name string
}

func (m *mockAuthFilter) Name() string {
	return m.name
}

func (m *mockAuthFilter) Init(peer.EndorserServer) {
}

func (m *mockAuthFilter) ProcessProposal(context.Context, *peer.SignedProposal) (*peer.ProposalResponse, error) {
	return nil, nil
}
