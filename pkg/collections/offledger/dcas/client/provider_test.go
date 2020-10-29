/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	olclient "github.com/trustbloc/fabric-peer-ext/pkg/collections/client"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

const (
	channel1 = "channel1"
	channel2 = "channel2"
	channel3 = "channel3"
)

func TestProvider(t *testing.T) {
	l := &mocks.Ledger{}
	lp := &mocks.LedgerProvider{}
	lp.GetLedgerReturns(l)

	providers := &olclient.Providers{
		LedgerProvider:   lp,
		GossipProvider:   &mocks.GossipProvider{},
		ConfigProvider:   &mocks.CollectionConfigProvider{},
		IdentityProvider: &mocks.IdentityProvider{},
	}
	p := NewProvider(providers)
	require.NotNil(t, p)

	client1_1, err := p.GetDCASClient(channel1, ns1, coll1)
	require.NoError(t, err)
	require.NotNil(t, client1_1)

	client1_2, err := p.GetDCASClient(channel1, ns1, coll1)
	require.NoError(t, err)
	require.Equal(t, client1_1, client1_2)

	client2, err := p.GetDCASClient(channel2, ns1, coll1)
	require.NoError(t, err)
	require.NotNil(t, client2)
	require.NotEqual(t, client1_1, client2)

	client3, err := p.GetDCASClient("", ns1, coll1)
	require.EqualError(t, err, "channel ID, ns, and collection must be specified")
	require.Nil(t, client3)

	lp.GetLedgerReturns(nil)
	client4, err := p.GetDCASClient(channel3, ns1, coll1)
	require.EqualError(t, err, fmt.Sprintf("no ledger for channel [%s]", channel3))
	require.Nil(t, client4)
}
