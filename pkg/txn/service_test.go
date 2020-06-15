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

	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/status"
	sdkmocks "github.com/hyperledger/fabric-sdk-go/pkg/fab/mocks"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/api"
	txnmocks "github.com/trustbloc/fabric-peer-ext/pkg/txn/mocks"
)

//go:generate counterfeiter -o ./mocks/txnclient.gen.go --fake-name TxnClient . channelClient
//go:generate counterfeiter -o ./mocks/proprespvalidator.gen.go --fake-name ProposalResponseValidator ./api ProposalResponseValidator

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

	p := &providers{peerConfig: peerCfg, configService: cs, clientProvider: clientProvider, proposalResponseValidator: &txnmocks.ProposalResponseValidator{}}
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
		cliReturned.InvokeHandlerReturns(channel.Response{}, nil)
		resp, err := s.Endorse(req)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("Endorse -> error", func(t *testing.T) {
		errExpected := errors.New("injected query error")
		cliReturned.InvokeHandlerReturns(channel.Response{}, errExpected)
		resp, err := s.Endorse(req)
		require.EqualError(t, err, errExpected.Error())
		require.Nil(t, resp)
	})

	t.Run("Endorse with peer filter -> success", func(t *testing.T) {
		req := &api.Request{
			Args:       [][]byte{[]byte("arg1")},
			PeerFilter: &mockPeerFilter{},
		}

		cliReturned.InvokeHandlerReturns(channel.Response{}, nil)
		resp, err := s.Endorse(req)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("Endorse with TxnID - no nonce -> error", func(t *testing.T) {
		req := &api.Request{
			Args:          [][]byte{[]byte("arg1")},
			TransactionID: "txn1",
		}

		cliReturned.InvokeHandlerReturns(channel.Response{}, nil)
		resp, err := s.Endorse(req)
		require.EqualError(t, err, "nonce must be provided if TransactionID is present")
		require.Nil(t, resp)
	})

	t.Run("Endorse with nonce - no TxnID -> error", func(t *testing.T) {
		req := &api.Request{
			Args:  [][]byte{[]byte("arg1")},
			Nonce: []byte("nonce1"),
		}

		cliReturned.InvokeHandlerReturns(channel.Response{}, nil)
		resp, err := s.Endorse(req)
		require.EqualError(t, err, "TransactionID must be provided if nonce is present")
		require.Nil(t, resp)
	})

	t.Run("Endorse with invalid TxnID -> error", func(t *testing.T) {
		req := &api.Request{
			Args:          [][]byte{[]byte("arg1")},
			TransactionID: "txn1",
			Nonce:         []byte("nonce1"),
		}

		const txnIDExpected = "txn1234"
		cliReturned.InvokeHandlerReturns(channel.Response{}, nil)
		cliReturned.ComputeTxnIDReturns(txnIDExpected, nil)

		resp, err := s.Endorse(req)
		require.EqualError(t, err, "transaction ID is invalid for the given nonce")
		require.Nil(t, resp)
	})

	t.Run("Endorse with valid TxnID -> success", func(t *testing.T) {
		const txnID = "txn1234"

		req := &api.Request{
			Args:          [][]byte{[]byte("arg1")},
			TransactionID: txnID,
			Nonce:         []byte("nonce1"),
		}

		cliReturned.InvokeHandlerReturns(channel.Response{}, nil)
		cliReturned.ComputeTxnIDReturns(txnID, nil)

		resp, err := s.Endorse(req)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("EndorseAndCommit -> success", func(t *testing.T) {
		cliReturned.InvokeHandlerReturns(channel.Response{}, nil)
		resp, committed, err := s.EndorseAndCommit(req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, committed)
	})

	t.Run("c -> error", func(t *testing.T) {
		errExpected := errors.New("injected query error")
		cliReturned.InvokeHandlerReturns(channel.Response{}, errExpected)
		resp, committed, err := s.EndorseAndCommit(req)
		require.EqualError(t, err, errExpected.Error())
		require.Nil(t, resp)
		require.False(t, committed)
	})

	t.Run("EndorseAndCommit with TxnID - no nonce -> error", func(t *testing.T) {
		req := &api.Request{
			Args:          [][]byte{[]byte("arg1")},
			TransactionID: "txn1",
		}

		cliReturned.InvokeHandlerReturns(channel.Response{}, nil)
		resp, committed, err := s.EndorseAndCommit(req)
		require.EqualError(t, err, "nonce must be provided if TransactionID is present")
		require.False(t, committed)
		require.Nil(t, resp)
	})

	t.Run("CommitEndorsements -> success", func(t *testing.T) {
		req := &api.CommitRequest{}

		cliReturned.InvokeHandlerReturns(channel.Response{}, nil)
		resp, _, err := s.CommitEndorsements(req)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("SigningIdentity -> success", func(t *testing.T) {
		identity := []byte("identity")
		cliReturned.InvokeHandlerReturns(channel.Response{}, nil)
		cliReturned.SigningIdentityReturns(identity, nil)

		identity, err := s.SigningIdentity()
		require.NoError(t, err)
		require.NotEmpty(t, identity)
	})

	t.Run("GetPeer -> success", func(t *testing.T) {
		peer, err := s.GetPeer("peer1:7051")
		require.NoError(t, err)
		require.Nil(t, peer)
	})

	t.Run("VerifyProposalSignature -> success", func(t *testing.T) {
		require.NoError(t, s.VerifyProposalSignature(&pb.SignedProposal{}))
	})

	t.Run("VerifyProposalSignature -> success", func(t *testing.T) {
		code, err := s.ValidateProposalResponses(&pb.SignedProposal{}, []*pb.ProposalResponse{})
		require.NoError(t, err)
		require.Equal(t, pb.TxValidationCode_VALID, code)
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

	t.Run("BeforeRetryHandler", func(t *testing.T) {
		errExpected := errors.New("injected error")

		numRetries := 1
		var err error
		s.beforeRetryHandler(&numRetries, &err)(errExpected)
		require.EqualError(t, err, errExpected.Error())
		require.Equal(t, 2, numRetries)
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

func TestNewRetryOpts(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		cfg := &txnConfig{
			User:           "",
			RetryAttempts:  3,
			InitialBackoff: "250ms",
			MaxBackoff:     "5s",
			BackoffFactor:  2.5,
			RetryableCodes: []int{500, 501},
		}

		opts := newRetryOpts(cfg)
		require.Equal(t, cfg.RetryAttempts, opts.Attempts)
		require.Equal(t, 250*time.Millisecond, opts.InitialBackoff)
		require.Equal(t, 5*time.Second, opts.MaxBackoff)
		require.Equal(t, cfg.BackoffFactor, opts.BackoffFactor)

		codes := opts.RetryableCodes[status.ChaincodeStatus]
		require.Len(t, codes, 2)
		require.Equal(t, status.Code(500), codes[0])
		require.Equal(t, status.Code(501), codes[1])
	})

	t.Run("Default values", func(t *testing.T) {
		opts := newRetryOpts(&txnConfig{})
		require.Equal(t, retry.DefaultAttempts, opts.Attempts)
		require.Equal(t, retry.DefaultInitialBackoff, opts.InitialBackoff)
		require.Equal(t, retry.DefaultMaxBackoff, opts.MaxBackoff)
		require.Equal(t, retry.DefaultBackoffFactor, opts.BackoffFactor)
	})
}

func TestPeerFilter(t *testing.T) {
	f := newTargetFilter(&mockPeerFilter{})
	require.False(t, f.Accept(&sdkmocks.MockPeer{MockMSP: msp1, MockURL: peer1}))

	f = newTargetFilter(&mockPeerFilter{endpoint: peer1, mspID: msp1})
	require.True(t, f.Accept(&sdkmocks.MockPeer{MockMSP: msp1, MockURL: peer1}))
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
	cl    channelClient
	err   error
	mutex sync.RWMutex
}

func (m *mockClientProvider) CreateClient(channelID, userName string, peerConfig api.PeerConfig, sdkCfgBytes []byte, format config.Format) (channelClient, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.cl, m.err
}

func (m *mockClientProvider) setClient(cl channelClient) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.cl = cl
}

func (m *mockClientProvider) setError(err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.err = err
}

type mockPeerFilter struct {
	endpoint string
	mspID    string
}

func (f *mockPeerFilter) Accept(p api.Peer) bool {
	logger.Infof("P [%s], [%s]", p.Endpoint(), p.MSPID())
	return p.MSPID() == f.mspID && p.Endpoint() == f.endpoint
}
