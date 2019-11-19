/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

const (
	channel1 = "channel1"
	channel2 = "channel2"
	channel3 = "channel3"
)

//go:generate counterfeiter -o ./mocks/pvtdatadistributor.gen.go --fake-name PvtDataDistributor . PvtDataDistributor

func TestProvider(t *testing.T) {
	l := &mocks.Ledger{}
	lp := &mocks.LedgerProvider{}
	lp.GetLedgerReturns(l)

	providers := &Providers{
		LedgerProvider:   lp,
		GossipProvider:   &mocks.GossipProvider{},
		ConfigProvider:   &mocks.CollectionConfigProvider{},
		IdentityProvider: &mocks.IdentityProvider{},
	}
	p := NewProvider(providers)
	require.NotNil(t, p)

	client1_1, err := p.ForChannel(channel1)
	require.NoError(t, err)
	require.NotNil(t, client1_1)

	client1_2, err := p.ForChannel(channel1)
	require.NoError(t, err)
	require.Equal(t, client1_1, client1_2)

	client2, err := p.ForChannel(channel2)
	require.NoError(t, err)
	require.NotNil(t, client2)
	require.NotEqual(t, client1_1, client2)

	client3, err := p.ForChannel("")
	require.EqualError(t, err, "channel ID is empty")
	require.Nil(t, client3)

	lp.GetLedgerReturns(nil)
	client4, err := p.ForChannel(channel3)
	require.EqualError(t, err, fmt.Sprintf("no ledger for channel [%s]", channel3))
	require.Nil(t, client4)
}
