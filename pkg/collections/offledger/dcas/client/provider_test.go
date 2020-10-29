/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"fmt"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/require"

	olclient "github.com/trustbloc/fabric-peer-ext/pkg/collections/client"
	olmocks "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

//go:generate counterfeiter -o ../../mocks/dcasconfig.gen.go --fake-name DCASConfig . config

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

	t.Run("GetDCASClient", func(t *testing.T) {
		p := NewProvider(providers, &olmocks.DCASConfig{})
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
		require.EqualError(t, err, "channel ID, namespace, and collection must be specified")
		require.Nil(t, client3)

		lp.GetLedgerReturns(nil)
		client4, err := p.GetDCASClient(channel3, ns1, coll1)
		require.EqualError(t, err, fmt.Sprintf("no ledger for channel [%s]", channel3))
		require.Nil(t, client4)
	})

	t.Run("CreateDCASClientStubWrapper", func(t *testing.T) {
		p := NewProvider(providers, &olmocks.DCASConfig{})
		require.NotNil(t, p)

		stub := shimtest.NewMockStub(ns1, &mockChaincode{})
		client, err := p.CreateDCASClientStubWrapper(coll1, stub)
		require.NoError(t, err)
		require.NotNil(t, client)

		client, err = p.CreateDCASClientStubWrapper("", stub)
		require.EqualError(t, err, "collection must be specified")
		require.Nil(t, client)
	})
}

type mockChaincode struct {
}

func (cc *mockChaincode) Init(shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}
func (cc *mockChaincode) Invoke(shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}
