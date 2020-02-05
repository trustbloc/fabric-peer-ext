/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package txn

import (
	"errors"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/client"
	txnmocks "github.com/trustbloc/fabric-peer-ext/pkg/txn/mocks"
)

//go:generate counterfeiter -o ./mocks/txnclient.gen.go --fake-name TxnClient ./client ChannelClient

const (
	msp1  = "Org1MSP"
	peer1 = "peer1"
)

func TestClient(t *testing.T) {
	cs := &txnmocks.ConfigService{}

	sdkCfgBytes, err := ioutil.ReadFile("./client/testdata/sdk-config.yaml")
	require.NoError(t, err)

	sdkCfgValue := &config.Value{
		TxID:   "txid2",
		Format: "yaml",
		Config: string(sdkCfgBytes),
	}

	txnCfgValue := &config.Value{
		TxID:   "txid3",
		Format: "json",
		Config: `{"User":"User1"}`,
	}

	cs.GetReturnsOnCall(0, txnCfgValue, nil)
	cs.GetReturnsOnCall(1, sdkCfgValue, nil)
	cs.GetReturnsOnCall(2, txnCfgValue, nil)
	cs.GetReturnsOnCall(3, sdkCfgValue, nil)
	cs.GetReturnsOnCall(4, txnCfgValue, nil)
	cs.GetReturnsOnCall(5, sdkCfgValue, nil)

	peerCfg := &mocks.PeerConfig{}
	peerCfg.MSPIDReturns(msp1)
	peerCfg.PeerIDReturns(peer1)

	cliReturned := &mockClosableClient{}
	clientProvider := &mockClientProvider{cl: cliReturned}

	p := &providers{peerConfig: peerCfg, configService: cs, clientProvider: clientProvider}
	s, err := newService("channel1", p)
	require.NoError(t, err)
	require.NotNil(t, s)

	defer s.Close()

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
		cliReturned.QueryReturns(channel.Response{}, nil)
		resp, err := s.Endorse(req)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("Endorse -> error", func(t *testing.T) {
		errExpected := errors.New("injected query error")
		cliReturned.QueryReturns(channel.Response{}, errExpected)
		resp, err := s.Endorse(req)
		require.EqualError(t, err, errExpected.Error())
		require.Nil(t, resp)
	})

	t.Run("EndorseAndCommit -> success", func(t *testing.T) {
		cliReturned.ExecuteReturns(channel.Response{}, nil)
		resp, err := s.EndorseAndCommit(req)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("EndorseAndCommit -> error", func(t *testing.T) {
		errExpected := errors.New("injected query error")
		cliReturned.ExecuteReturns(channel.Response{}, errExpected)
		resp, err := s.EndorseAndCommit(req)
		require.EqualError(t, err, errExpected.Error())
		require.Nil(t, resp)
	})

	t.Run("Config update", func(t *testing.T) {
		clUpdated1 := &mockClosableClient{}
		clientProvider.setClient(clUpdated1)

		require.False(t, cliReturned.isClosed())

		txnCfgValue := &config.Value{
			TxID:   "tx10",
			Format: "json",
			Config: `{"User":"User1"}`,
		}

		s.handleConfigUpdate(config.NewKeyValue(config.NewAppKey("org1", "someapp", "v1"), txnCfgValue))
		time.Sleep(500 * time.Millisecond)

		require.False(t, cliReturned.isClosed(), "expecting original client to not be closed since the config update was not relevant to us")
		require.Equal(t, cliReturned, s.client())

		s.handleConfigUpdate(config.NewKeyValue(s.txnCfgKey, txnCfgValue))
		time.Sleep(500 * time.Millisecond)

		require.True(t, cliReturned.isClosed(), "expecting original client to be closed")
		require.Equal(t, clUpdated1, s.client())
		require.False(t, clUpdated1.isClosed())

		clUpdated2 := &mockClosableClient{}
		clientProvider.setClient(clUpdated2)

		s.handleConfigUpdate(config.NewKeyValue(s.txnCfgKey, txnCfgValue))
		time.Sleep(500 * time.Millisecond)

		require.False(t, clUpdated1.isClosed(), "expecting original client NOT to be closed since the config update was for the same transaction")
		require.Equal(t, clUpdated1, s.client())
		require.False(t, clUpdated1.isClosed())
	})

	t.Run("Config update error", func(t *testing.T) {
		origClient := s.client()

		clientProvider.setClient(&mockClosableClient{})
		clientProvider.setError(errors.New("injected load error"))

		txnCfgValue := &config.Value{
			TxID:   "tx11",
			Format: "json",
			Config: `{"User":"User1"}`,
		}

		s.handleConfigUpdate(config.NewKeyValue(s.txnCfgKey, txnCfgValue))
		time.Sleep(500 * time.Millisecond)

		require.Equal(t, origClient, s.client(), "expecting original client to still be used")
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

	errExpected := errors.New("injected new client error")

	p := &providers{peerConfig: peerCfg, configService: cs, clientProvider: &mockClientProvider{err: errExpected}}
	s, err := newService("channel1", p)
	require.EqualError(t, err, errExpected.Error())
	require.Nil(t, s)
}

type mockClosableClient struct {
	txnmocks.TxnClient
	closed bool
	mutex  sync.RWMutex
}

func (m *mockClosableClient) Close() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.closed = true
}

func (m *mockClosableClient) isClosed() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.closed
}

type mockClientProvider struct {
	cl    client.ChannelClient
	err   error
	mutex sync.RWMutex
}

func (m *mockClientProvider) CreateClient(channelID, userName string, peerConfig api.PeerConfig, sdkCfgBytes []byte, format config.Format) (client.ChannelClient, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.cl, m.err
}

func (m *mockClientProvider) setClient(cl client.ChannelClient) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.cl = cl
}

func (m *mockClientProvider) setError(err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.err = err
}
