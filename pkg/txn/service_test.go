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
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/mocks"
)

//go:generate counterfeiter -o ./mocks/txnclient.gen.go --fake-name TxnClient . txnClient

func TestClient(t *testing.T) {
	cs := &mocks.ConfigService{}

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

	cl := &mocks.TxnClient{}
	cl.EndorseReturns(&channel.Response{}, nil)
	cl.EndorseAndCommitReturns(&channel.Response{}, nil)

	newClient = func(channelID, userName string, peerConfig api.PeerConfig, sdkCfgBytes []byte, format config.Format) (txnClient, error) {
		return cl, nil
	}

	s, err := New("channel1", peerCfg, cs)
	require.NoError(t, err)
	require.NotNil(t, s)

	req := &api.Request{
		Args: [][]byte{[]byte("arg1")},
	}

	resp, err := s.Endorse(req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	resp, err = s.EndorseAndCommit(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestNew_Error(t *testing.T) {
	cs := &mocks.ConfigService{}

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
	newClient = func(channelID, userName string, peerConfig api.PeerConfig, sdkCfgBytes []byte, format config.Format) (client txnClient, err error) {
		return nil, errExpected
	}

	s, err := New("channel1", peerCfg, cs)
	require.EqualError(t, err, errExpected.Error())
	require.Nil(t, s)
}
