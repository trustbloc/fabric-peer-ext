/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	fabapi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/api"
)

// endpointConfig overrides the SDK's default endpoint config and provides the local peer as the only channel (bootstrap) peer
type endpointConfig struct {
	fabapi.EndpointConfig
	localPeer fabapi.ChannelPeer
}

var newEndpointConfig = func(cfg fabapi.EndpointConfig, peerCfg api.PeerConfig) (*endpointConfig, error) {
	tlsCertPath := peerCfg.TLSCertPath()
	if tlsCertPath == "" {
		return nil, errors.New("no TLS cert path specified")
	}

	tlsCertPem, err := getCertPemFromPath(tlsCertPath)
	if err != nil {
		return nil, err
	}

	peer, err := getLocalPeer(peerCfg)
	if err != nil {
		return nil, err
	}

	url := peer.URL()
	peerConfig, ok := cfg.PeerConfig(url)
	if !ok {
		return nil, errors.Errorf("could not find channel peer for [%s]", url)
	}

	networkPeer, err := newNetworkPeer(peerConfig, peer.mspID, tlsCertPem)
	if err != nil {
		return nil, errors.Errorf("error creating network peer for [%s]: %s", url, err)
	}

	localPeer := fabapi.ChannelPeer{
		PeerChannelConfig: fabapi.PeerChannelConfig{
			EndorsingPeer:  true,
			ChaincodeQuery: true,
			LedgerQuery:    true,
			EventSource:    true,
		},
		NetworkPeer: *networkPeer,
	}

	return &endpointConfig{
		EndpointConfig: cfg,
		localPeer:      localPeer,
	}, nil
}

// ChannelPeers returns the local peer
func (c *endpointConfig) ChannelPeers(_ string) []fabapi.ChannelPeer {
	return []fabapi.ChannelPeer{c.localPeer}
}

func newNetworkPeer(peerConfig *fabapi.PeerConfig, mspID string, pemBytes []byte) (*fabapi.NetworkPeer, error) {
	networkPeer := &fabapi.NetworkPeer{PeerConfig: *peerConfig, MSPID: mspID}

	block, _ := pem.Decode(pemBytes)
	if block != nil {
		pub, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, errors.WithMessage(err, "certificate parsing failed")
		}

		networkPeer.TLSCACert = pub
	}

	return networkPeer, nil
}

func getLocalPeer(peerCfg api.PeerConfig) (*peerDesc, error) {
	mspID := peerCfg.MSPID()
	if mspID == "" {
		return nil, errors.New("MSP ID not defined")
	}

	peerAddress := peerCfg.PeerAddress()
	if peerAddress == "" {
		return nil, errors.New("peer address not defined")
	}

	splitPeerAddress := strings.Split(peerAddress, ":")
	port, err := strconv.Atoi(splitPeerAddress[1])
	if err != nil {
		return nil, errors.WithMessagef(err, "invalid port in peer address [%s]", peerAddress)
	}

	return &peerDesc{
		host:  splitPeerAddress[0],
		port:  port,
		mspID: peerCfg.MSPID(),
	}, nil
}

func getCertPemFromPath(certPath string) ([]byte, error) {
	pemBuffer, err := ioutil.ReadFile(filepath.Clean(certPath))
	if err != nil {
		return nil, errors.Errorf("cert fixture missing at path '%s', err: %s", certPath, err)
	}
	return pemBuffer, nil
}

type peerDesc struct {
	host  string
	port  int
	mspID string
}

func (p *peerDesc) URL() string {
	return fmt.Sprintf("%s:%d", p.host, p.port)
}
