/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package txn

import (
	"errors"
	"io/ioutil"
	"testing"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/client"
	txnmocks "github.com/trustbloc/fabric-peer-ext/pkg/txn/mocks"
)

//go:generate counterfeiter -o ./mocks/txnclient.gen.go --fake-name TxnClient ./client ChannelClient

func TestClient(t *testing.T) {
	cs := &txnmocks.ConfigService{}

	cs.GetReturnsOnCall(0, &config.Value{
		TxID:   "txid1",
		Format: "json",
		Config: `{"User":"User1"}`,
	}, nil)

	sdkCfgBytes, err := ioutil.ReadFile("./client/testdata/sdk-config.yaml")
	require.NoError(t, err)

	cs.GetReturnsOnCall(1, &config.Value{
		TxID:   "txid2",
		Format: "yaml",
		Config: string(sdkCfgBytes),
	}, nil)

	peerCfg := &mocks.PeerConfig{}

	restoreNewClient := newClient
	defer func() {
		newClient = restoreNewClient
	}()

	cl := &txnmocks.TxnClient{}

	newClient = func(channelID, userName string, peerConfig api.PeerConfig, sdkCfgBytes []byte, format config.Format) (client.ChannelClient, error) {
		return cl, nil
	}

	s, err := New("channel1", peerCfg, cs)
	require.NoError(t, err)
	require.NotNil(t, s)

	req := &api.Request{
		Args: [][]byte{[]byte("arg1")},
		InvocationChain: []*api.ChaincodeCall{
			{
				ChaincodeName: "cc2",
				Collections:   []string{"coll1"},
			},
		},
	}

	t.Run("Endorse -> success", func(t *testing.T) {
		cl.QueryReturns(channel.Response{}, nil)
		resp, err := s.Endorse(req)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("Endorse -> error", func(t *testing.T) {
		errExpected := errors.New("injected query error")
		cl.QueryReturns(channel.Response{}, errExpected)
		resp, err := s.Endorse(req)
		require.EqualError(t, err, errExpected.Error())
		require.Nil(t, resp)
	})

	t.Run("EndorseAndCommit -> success", func(t *testing.T) {
		cl.ExecuteReturns(channel.Response{}, nil)
		resp, err := s.EndorseAndCommit(req)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("EndorseAndCommit -> error", func(t *testing.T) {
		errExpected := errors.New("injected query error")
		cl.ExecuteReturns(channel.Response{}, errExpected)
		resp, err := s.EndorseAndCommit(req)
		require.EqualError(t, err, errExpected.Error())
		require.Nil(t, resp)
	})
}

func TestNew_Error(t *testing.T) {
	cs := &txnmocks.ConfigService{}

	cs.GetReturnsOnCall(0, &config.Value{
		TxID:   "txid1",
		Format: "json",
		Config: `{"User":"User1"}`,
	}, nil)

	sdkCfgBytes, err := ioutil.ReadFile("./client/testdata/sdk-config.yaml")
	require.NoError(t, err)

	cs.GetReturnsOnCall(1, &config.Value{
		TxID:   "txid2",
		Format: "yaml",
		Config: string(sdkCfgBytes),
	}, nil)

	peerCfg := &mocks.PeerConfig{}

	restoreNewClient := newClient
	defer func() {
		newClient = restoreNewClient
	}()

	errExpected := errors.New("injected new client error")
	newClient = func(channelID, userName string, peerConfig api.PeerConfig, sdkCfgBytes []byte, format config.Format) (channelClient client.ChannelClient, err error) {
		return nil, errExpected
	}

	s, err := New("channel1", peerCfg, cs)
	require.EqualError(t, err, errExpected.Error())
	require.Nil(t, s)
}
