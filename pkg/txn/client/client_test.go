/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"crypto"
	"io/ioutil"
	"testing"

	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric-protos-go/common"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	sdkmocks "github.com/hyperledger/fabric-sdk-go/pkg/fab/mocks"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/api"
	clientmocks "github.com/trustbloc/fabric-peer-ext/pkg/txn/client/mocks"
)

//go:generate counterfeiter -o ./mocks/channelclient.gen.go --fake-name ChannelClient . ChannelClient
//go:generate counterfeiter -o ./mocks/identityserializer.gen.go --fake-name IdentitySerializer . identitySerializer
//go:generate counterfeiter -o ./mocks/cryptosuiteprovider.gen.go --fake-name CryptoSuiteProvider . cryptoSuiteProvider
//go:generate counterfeiter -o ../../mocks/peerconfig.gen.go --fake-name PeerConfig ../api PeerConfig
//go:generate counterfeiter -o ./mocks/discoveryservice.gen.go --fake-name DiscoveryService github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab.DiscoveryService

func TestNew(t *testing.T) {
	peerCfg := &mocks.PeerConfig{}
	peerCfg.TLSCertPathReturns("./testdata/tls.crt")
	peerCfg.MSPIDReturns("Org1MSP")
	peerCfg.PeerAddressReturns("peer0.org1.com:7051")

	sdkCfgBytes, err := ioutil.ReadFile("./testdata/sdk-config.yaml")
	require.NoError(t, err)

	b := &clientmocks.BCCSP{}
	bKey := &clientmocks.BCCSPKey{}
	bKey.PrivateReturns(true)
	b.GetKeyReturns(bKey, nil)
	b.KeyImportReturns(bKey, nil)

	restoreNewCryptoSuite := newCryptoSuite
	newCryptoSuite = func(bccsp.BCCSP) *cryptoSuite {
		return &cryptoSuite{bccsp: b}
	}
	defer func() { newCryptoSuite = restoreNewCryptoSuite }()

	newChannelClient = func(channelID, userName, org string, sdk *fabsdk.FabricSDK) (*channelProviders, error) {
		return &channelProviders{ChannelClient: &clientmocks.ChannelClient{}}, nil
	}

	t.Run("success", func(t *testing.T) {
		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.NoError(t, err)
		require.NotNil(t, c)
		require.NotPanics(t, c.Close)
	})

	t.Run("Invalid SDK config -> error", func(t *testing.T) {
		c, err := New("channel1", "User1", peerCfg, []byte("sdk config"), "YAML")
		require.Error(t, err)
		require.Contains(t, err.Error(), "unmarshal errors")
		require.Nil(t, c)

		bytes, err := ioutil.ReadFile("./testdata/sdk-config-invalid.yaml")
		require.NoError(t, err)

		c, err = New("channel1", "User1", peerCfg, bytes, "YAML")
		require.Error(t, err)
		require.Contains(t, err.Error(), "org not configured for MSP")
		require.Nil(t, c)
	})

	t.Run("Invalid org -> error", func(t *testing.T) {
		peerCfg.MSPIDReturns("invalid-msp")
		defer func() {
			peerCfg.MSPIDReturns("Org1MSP")
		}()

		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.Error(t, err)
		require.Contains(t, err.Error(), "org not configured for MSP")
		require.Nil(t, c)
	})

	t.Run("newSDK -> error", func(t *testing.T) {
		errExpected := errors.New("injected SDK error")
		restoreNewSDK := newSDK
		newSDK = func(channelID string, configProvider core.ConfigProvider, config fab.EndpointConfig, peerCfg api.PeerConfig) (sdk *fabsdk.FabricSDK, err error) {
			return nil, errExpected
		}
		defer func() {
			newSDK = restoreNewSDK
		}()

		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.Nil(t, c)
	})

	t.Run("newEndpointConfig -> error", func(t *testing.T) {
		errExpected := errors.New("injected endpoint config error")
		restoreNewEndpointConfig := newEndpointConfig
		newEndpointConfig = func(cfg fab.EndpointConfig, peerCfg api.PeerConfig) (config *endpointConfig, err error) {
			return nil, errExpected
		}
		defer func() {
			newEndpointConfig = restoreNewEndpointConfig
		}()

		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.Nil(t, c)
	})

	t.Run("newChannelClient -> error", func(t *testing.T) {
		errExpected := errors.New("injected channel client error")
		restoreNewChannelClient := newChannelClient
		newChannelClient = func(channelID, userName, org string, sdk *fabsdk.FabricSDK) (*channelProviders, error) {
			return nil, errExpected
		}
		defer func() {
			newChannelClient = restoreNewChannelClient
		}()

		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.Nil(t, c)
	})
}

func TestClient_InvokeHandler(t *testing.T) {
	peerCfg := &mocks.PeerConfig{}
	peerCfg.TLSCertPathReturns("./testdata/tls.crt")
	peerCfg.MSPIDReturns("Org1MSP")
	peerCfg.PeerAddressReturns("peer0.org1.com:7051")

	sdkCfgBytes, err := ioutil.ReadFile("./testdata/sdk-config.yaml")
	require.NoError(t, err)

	b := &clientmocks.BCCSP{}
	bKey := &clientmocks.BCCSPKey{}
	bKey.PrivateReturns(true)
	b.GetKeyReturns(bKey, nil)
	b.KeyImportReturns(bKey, nil)

	restoreNewCryptoSuite := newCryptoSuite
	newCryptoSuite = func(bccsp.BCCSP) *cryptoSuite {
		return &cryptoSuite{bccsp: b}
	}
	defer func() { newCryptoSuite = restoreNewCryptoSuite }()

	newChannelClient = func(channelID, userName, org string, sdk *fabsdk.FabricSDK) (*channelProviders, error) {
		return &channelProviders{ChannelClient: &clientmocks.ChannelClient{}}, nil
	}

	t.Run("InvokeHandler -> success", func(t *testing.T) {
		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.NoError(t, err)
		require.NotNil(t, c)

		req := channel.Request{}
		_, err = c.InvokeHandler(&mockHandler{}, req)
		require.NoError(t, err)
	})

	t.Run("InvokeHandler on closed client -> error", func(t *testing.T) {
		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.NoError(t, err)
		require.NotNil(t, c)
		c.Close()

		req := channel.Request{}
		_, err = c.InvokeHandler(&mockHandler{}, req)
		require.EqualError(t, err, "attempt to increment count on closed resource")
	})

	t.Run("Decrement counter -> error", func(t *testing.T) {
		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.NoError(t, err)
		require.NotNil(t, c)
		require.NotPanics(t, c.decrementCounter)
	})
}

func TestClient_SigningIdentity(t *testing.T) {
	peerCfg := &mocks.PeerConfig{}
	peerCfg.TLSCertPathReturns("./testdata/tls.crt")
	peerCfg.MSPIDReturns("Org1MSP")
	peerCfg.PeerAddressReturns("peer0.org1.com:7051")

	sdkCfgBytes, err := ioutil.ReadFile("./testdata/sdk-config.yaml")
	require.NoError(t, err)

	t.Run("SigningIdentity -> success", func(t *testing.T) {
		serializedIdentity := []byte("serialized identity")

		newChannelClient = func(channelID, userName, org string, sdk *fabsdk.FabricSDK) (*channelProviders, error) {
			idSerializer := &clientmocks.IdentitySerializer{}
			idSerializer.SerializeReturns(serializedIdentity, nil)
			return &channelProviders{ChannelClient: &clientmocks.ChannelClient{}, identitySerializer: idSerializer}, nil
		}

		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.NoError(t, err)
		require.NotNil(t, c)

		identity, err := c.SigningIdentity()
		require.NoError(t, err)
		require.Equal(t, serializedIdentity, identity)
	})

	t.Run("SigningIdentity -> error", func(t *testing.T) {
		errExpected := errors.New("injected serializer error")

		newChannelClient = func(channelID, userName, org string, sdk *fabsdk.FabricSDK) (*channelProviders, error) {
			idSerializer := &clientmocks.IdentitySerializer{}
			idSerializer.SerializeReturns(nil, errExpected)
			return &channelProviders{ChannelClient: &clientmocks.ChannelClient{}, identitySerializer: idSerializer}, nil
		}

		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.NoError(t, err)
		require.NotNil(t, c)

		identity, err := c.SigningIdentity()
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.Empty(t, identity)
	})
}

func TestClient_ComputeTxnID(t *testing.T) {
	peerCfg := &mocks.PeerConfig{}
	peerCfg.TLSCertPathReturns("./testdata/tls.crt")
	peerCfg.MSPIDReturns("Org1MSP")
	peerCfg.PeerAddressReturns("peer0.org1.com:7051")

	sdkCfgBytes, err := ioutil.ReadFile("./testdata/sdk-config.yaml")
	require.NoError(t, err)

	chClient := &clientmocks.ChannelClient{}
	idSerializer := &clientmocks.IdentitySerializer{}

	cs := &clientmocks.CryptoSuite{}

	csp := &clientmocks.CryptoSuiteProvider{}
	csp.CryptoSuiteReturns(cs)

	newChannelClient = func(channelID, userName, org string, sdk *fabsdk.FabricSDK) (*channelProviders, error) {
		return &channelProviders{ChannelClient: chClient, identitySerializer: idSerializer, cryptoSuiteProvider: csp}, nil
	}

	t.Run("ComputeTxn -> success", func(t *testing.T) {
		serializedIdentity := []byte("serialized identity")

		idSerializer.SerializeReturns(serializedIdentity, nil)
		cs.GetHashReturns(crypto.SHA256.New(), nil)

		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.NoError(t, err)
		require.NotNil(t, c)

		txnID, err := c.ComputeTxnID([]byte("nonce"))
		require.NoError(t, err)
		require.NotEmpty(t, txnID)
	})

	t.Run("Serialize -> error", func(t *testing.T) {
		errExpected := errors.New("injected serializer error")
		idSerializer.SerializeReturns(nil, errExpected)
		cs.GetHashReturns(crypto.SHA256.New(), nil)

		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.NoError(t, err)
		require.NotNil(t, c)

		txnID, err := c.ComputeTxnID([]byte("nonce"))
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.Empty(t, txnID)
	})

	t.Run("CryptoSuite -> error", func(t *testing.T) {
		errExpected := errors.New("injected serializer error")
		serializedIdentity := []byte("serialized identity")

		idSerializer.SerializeReturns(serializedIdentity, nil)
		cs.GetHashReturns(nil, errExpected)

		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.NoError(t, err)
		require.NotNil(t, c)

		txnID, err := c.ComputeTxnID([]byte("nonce"))
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.Empty(t, txnID)
	})

	t.Run("HashWrite -> error", func(t *testing.T) {
		errExpected := errors.New("injected hash write error")
		idSerializer.SerializeReturns(nil, nil)

		h := &clientmocks.Hash{}
		h.WriteReturns(0, errExpected)
		cs.GetHashReturns(h, nil)

		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.NoError(t, err)
		require.NotNil(t, c)

		txnID, err := c.ComputeTxnID(nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.Empty(t, txnID)
	})
}

func TestClient_GetPeer(t *testing.T) {
	peerCfg := &mocks.PeerConfig{}
	peerCfg.TLSCertPathReturns("./testdata/tls.crt")
	peerCfg.MSPIDReturns("Org1MSP")
	peerCfg.PeerAddressReturns("peer0.org1.com:7051")

	sdkCfgBytes, err := ioutil.ReadFile("./testdata/sdk-config.yaml")
	require.NoError(t, err)

	chClient := &clientmocks.ChannelClient{}

	t.Run("By endpoint", func(t *testing.T) {
		const p1Endpoint = "peer1:7051"

		peer1 := sdkmocks.NewMockPeer("peer1", p1Endpoint)
		discovery := &clientmocks.DiscoveryService{}
		discovery.GetPeersReturns([]fab.Peer{peer1}, nil)

		newChannelClient = func(channelID, userName, org string, sdk *fabsdk.FabricSDK) (*channelProviders, error) {
			return &channelProviders{ChannelClient: chClient, DiscoveryService: discovery}, nil
		}

		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.NoError(t, err)
		require.NotNil(t, c)

		peer, err := c.GetPeer(p1Endpoint)
		require.NoError(t, err)
		require.NotNil(t, peer)
		require.Equal(t, "peer1:7051", peer.URL())
	})

	t.Run("By URL", func(t *testing.T) {
		const p1Endpoint = "peer1:7051"
		const p1URL = "grpc://peer1:7051"

		peer1 := sdkmocks.NewMockPeer("peer1", p1URL)
		discovery := &clientmocks.DiscoveryService{}
		discovery.GetPeersReturns([]fab.Peer{peer1}, nil)

		newChannelClient = func(channelID, userName, org string, sdk *fabsdk.FabricSDK) (*channelProviders, error) {
			return &channelProviders{ChannelClient: chClient, DiscoveryService: discovery}, nil
		}

		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.NoError(t, err)
		require.NotNil(t, c)

		peer, err := c.GetPeer(p1Endpoint)
		require.NoError(t, err)
		require.NotNil(t, peer)
	})

	t.Run("Peer not found", func(t *testing.T) {
		const p1Endpoint = "peer1:7051"

		peer1 := sdkmocks.NewMockPeer("peer1", p1Endpoint)
		discovery := &clientmocks.DiscoveryService{}
		discovery.GetPeersReturns([]fab.Peer{peer1}, nil)

		newChannelClient = func(channelID, userName, org string, sdk *fabsdk.FabricSDK) (*channelProviders, error) {
			return &channelProviders{ChannelClient: chClient, DiscoveryService: discovery}, nil
		}

		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.NoError(t, err)
		require.NotNil(t, c)

		txnID, err := c.GetPeer("peer2:7051")
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
		require.Empty(t, txnID)
	})

	t.Run("Discovery error", func(t *testing.T) {
		errExpected := errors.New("injected Discovery error")
		discovery := &clientmocks.DiscoveryService{}
		discovery.GetPeersReturns(nil, errExpected)

		newChannelClient = func(channelID, userName, org string, sdk *fabsdk.FabricSDK) (*channelProviders, error) {
			return &channelProviders{ChannelClient: chClient, DiscoveryService: discovery}, nil
		}

		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.NoError(t, err)
		require.NotNil(t, c)

		txnID, err := c.GetPeer("peer2:7051")
		require.Error(t, err)
		require.Contains(t, err.Error(), "injected Discovery error")
		require.Empty(t, txnID)
	})
}

func TestClient_VerifyProposalSignature(t *testing.T) {
	peerCfg := &mocks.PeerConfig{}
	peerCfg.TLSCertPathReturns("./testdata/tls.crt")
	peerCfg.MSPIDReturns("Org1MSP")
	peerCfg.PeerAddressReturns("peer0.org1.com:7051")

	sdkCfgBytes, err := ioutil.ReadFile("./testdata/sdk-config.yaml")
	require.NoError(t, err)

	newChannelClient = func(channelID, userName, org string, sdk *fabsdk.FabricSDK) (*channelProviders, error) {
		return &channelProviders{ChannelClient: &clientmocks.ChannelClient{}, ChannelMembership: sdkmocks.NewMockMembership()}, nil
	}

	c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
	require.NoError(t, err)
	require.NotNil(t, c)

	t.Run("Nil proposal bytes -> error", func(t *testing.T) {
		err = c.VerifyProposalSignature(&pb.SignedProposal{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "ProposalBytes is nil in SignedProposal")
	})

	t.Run("Proposal unmarshal -> error", func(t *testing.T) {
		err = c.VerifyProposalSignature(&pb.SignedProposal{ProposalBytes: []byte("ppp")})
		require.Error(t, err)
		require.Contains(t, err.Error(), "error unmarshaling Proposal")
	})

	t.Run("Nil proposal header -> error", func(t *testing.T) {
		proposalBytes, err := proto.Marshal(&pb.Proposal{})
		require.NoError(t, err)

		err = c.VerifyProposalSignature(&pb.SignedProposal{ProposalBytes: proposalBytes})
		require.Error(t, err)
		require.Contains(t, err.Error(), "Header is nil in Proposal")
	})

	t.Run("Proposal header unmarshal -> error", func(t *testing.T) {
		proposalBytes, err := proto.Marshal(&pb.Proposal{Header: []byte("hhh")})
		require.NoError(t, err)

		err = c.VerifyProposalSignature(&pb.SignedProposal{ProposalBytes: proposalBytes})
		require.Error(t, err)
		require.Contains(t, err.Error(), "error unmarshaling Header")
	})

	t.Run("Nil signature header -> error", func(t *testing.T) {
		headerBytes, err := proto.Marshal(&cb.Header{ChannelHeader: []byte("ccc")})
		require.NoError(t, err)

		proposalBytes, err := proto.Marshal(&pb.Proposal{Header: headerBytes})
		require.NoError(t, err)

		err = c.VerifyProposalSignature(&pb.SignedProposal{ProposalBytes: proposalBytes})
		require.Error(t, err)
		require.Contains(t, err.Error(), "signatureHeader is nil in proposalHeader")
	})

	t.Run("Signature unmarshal -> error", func(t *testing.T) {
		headerBytes, err := proto.Marshal(&cb.Header{SignatureHeader: []byte("sss")})
		require.NoError(t, err)

		proposalBytes, err := proto.Marshal(&pb.Proposal{Header: headerBytes})
		require.NoError(t, err)

		err = c.VerifyProposalSignature(&pb.SignedProposal{ProposalBytes: proposalBytes})
		require.Error(t, err)
		require.Contains(t, err.Error(), "error unmarshaling SignatureHeader")
	})

	t.Run("Signature unmarshal -> error", func(t *testing.T) {
		headerBytes, err := proto.Marshal(&cb.Header{SignatureHeader: []byte("sss")})
		require.NoError(t, err)

		proposalBytes, err := proto.Marshal(&pb.Proposal{Header: headerBytes})
		require.NoError(t, err)

		err = c.VerifyProposalSignature(&pb.SignedProposal{ProposalBytes: proposalBytes})
		require.Error(t, err)
		require.Contains(t, err.Error(), "error unmarshaling SignatureHeader")
	})

	t.Run("Invalid creator certificate -> error", func(t *testing.T) {
		newChannelClient = func(channelID, userName, org string, sdk *fabsdk.FabricSDK) (*channelProviders, error) {
			return &channelProviders{ChannelClient: &clientmocks.ChannelClient{}, ChannelMembership: &sdkmocks.MockMembership{ValidateErr: errors.New("injected validation error")}}, nil
		}

		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.NoError(t, err)
		require.NotNil(t, c)

		signatureBytes, err := proto.Marshal(&cb.SignatureHeader{Creator: []byte("ccc"), Nonce: []byte("nnn")})
		require.NoError(t, err)

		headerBytes, err := proto.Marshal(&cb.Header{SignatureHeader: signatureBytes})
		require.NoError(t, err)

		proposalBytes, err := proto.Marshal(&pb.Proposal{Header: headerBytes})
		require.NoError(t, err)

		err = c.VerifyProposalSignature(&pb.SignedProposal{ProposalBytes: proposalBytes})
		require.Error(t, err)
		require.Contains(t, err.Error(), "creator certificate is not valid")
	})

	t.Run("Signature not valid -> error", func(t *testing.T) {
		newChannelClient = func(channelID, userName, org string, sdk *fabsdk.FabricSDK) (*channelProviders, error) {
			return &channelProviders{ChannelClient: &clientmocks.ChannelClient{}, ChannelMembership: &sdkmocks.MockMembership{VerifyErr: errors.New("injected verification error")}}, nil
		}

		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.NoError(t, err)
		require.NotNil(t, c)

		signatureBytes, err := proto.Marshal(&cb.SignatureHeader{Creator: []byte("ccc"), Nonce: []byte("nnn")})
		require.NoError(t, err)

		headerBytes, err := proto.Marshal(&cb.Header{SignatureHeader: signatureBytes})
		require.NoError(t, err)

		proposalBytes, err := proto.Marshal(&pb.Proposal{Header: headerBytes})
		require.NoError(t, err)

		err = c.VerifyProposalSignature(&pb.SignedProposal{ProposalBytes: proposalBytes})
		require.Error(t, err)
		require.Contains(t, err.Error(), "The creator's signature over the proposal is not valid")
	})

	t.Run("Signature valid -> success", func(t *testing.T) {
		newChannelClient = func(channelID, userName, org string, sdk *fabsdk.FabricSDK) (*channelProviders, error) {
			return &channelProviders{ChannelClient: &clientmocks.ChannelClient{}, ChannelMembership: &sdkmocks.MockMembership{}}, nil
		}

		c, err := New("channel1", "User1", peerCfg, sdkCfgBytes, "YAML")
		require.NoError(t, err)
		require.NotNil(t, c)

		signatureBytes, err := proto.Marshal(&cb.SignatureHeader{Creator: []byte("ccc"), Nonce: []byte("nnn")})
		require.NoError(t, err)

		headerBytes, err := proto.Marshal(&cb.Header{SignatureHeader: signatureBytes})
		require.NoError(t, err)

		proposalBytes, err := proto.Marshal(&pb.Proposal{Header: headerBytes})
		require.NoError(t, err)

		err = c.VerifyProposalSignature(&pb.SignedProposal{ProposalBytes: proposalBytes})
		require.NoError(t, err)
	})
}

type mockHandler struct {
}

func (c *mockHandler) Handle(*invoke.RequestContext, *invoke.ClientContext) {
}
