/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package txn

import (
	"errors"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
	txnmocks "github.com/trustbloc/fabric-peer-ext/pkg/txn/mocks"
)

//go:generate counterfeiter -o ./mocks/configserviceprovider.gen.go --fake-name ConfigServiceProvider . configServiceProvider
//go:generate counterfeiter -o ./mocks/peerconfig.gen.go --fake-name PeerConfig ./api PeerConfig
//go:generate counterfeiter -o ./mocks/configservice.gen.go --fake-name ConfigService ../config/ledgerconfig/config Service

func TestNewProvider(t *testing.T) {
	require.NotNil(t, NewProvider(&txnmocks.ConfigServiceProvider{}, &mocks.PeerConfig{}))
}

func TestProvider(t *testing.T) {
	csp := &txnmocks.ConfigServiceProvider{}
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
	csp.ForChannelReturns(cs)

	p := newProvider(csp, &mocks.PeerConfig{}, &mockClientProvider{cl: &txnmocks.TxnClient{}})
	require.NotNil(t, p)

	s, err := p.ForChannel("channel1")
	require.NoError(t, err)
	require.NotNil(t, s)

	cs.GetReturnsOnCall(2, &config.Value{
		TxID:   "txid3",
		Format: "json",
		Config: "",
	}, nil)

	s, err = p.ForChannel("channel2")
	require.Error(t, err)
	require.Contains(t, err.Error(), "error unmarshalling TXN config")
	require.Nil(t, s)

	cs.GetReturnsOnCall(3, &config.Value{
		TxID:   "txid4",
		Format: "json",
		Config: `{"User":"User1"}`,
	}, nil)

	errExpected := errors.New("injected SDK config error")
	cs.GetReturnsOnCall(4, nil, errExpected)

	s, err = p.ForChannel("channel3")
	require.Error(t, err)
	require.Contains(t, err.Error(), errExpected.Error())
	require.Nil(t, s)

	p.Close()
}
