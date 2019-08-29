/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/mgr"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/mocks"
)

const (
	channelID = "testchannel"
	msp1      = "org1MSP"
	peer1     = "peer1.example.com"

	app1 = "app1"
	app2 = "app2"

	comp1 = "comp1"
	v1    = "1"
)

func TestConfigService_Get(t *testing.T) {
	key2 := &config.Key{MspID: msp1, AppName: app1, AppVersion: v1}
	cfg := config.NewValue("tx1", "some config", config.FormatOther)
	bytes, err := json.Marshal(cfg)
	require.NoError(t, err)

	r := mocks.NewStateRetriever()
	r.WithState(ConfigNS, mgr.MarshalKey(key2), bytes)
	p := mocks.NewStateRetrieverProvider().WithStateRetriever(r)

	svc := New(channelID, p)
	require.NotNil(t, svc)

	t.Run("Invalid key", func(t *testing.T) {
		value, err := svc.Get(&config.Key{})
		require.Error(t, err)
		require.Nil(t, value)

		value, err = svc.Get(&config.Key{MspID: msp1, PeerID: peer1})
		require.Error(t, err)
		require.Nil(t, value)

		value, err = svc.Get(&config.Key{MspID: msp1, AppName: app1})
		require.Error(t, err)
		require.Nil(t, value)

		_, err = svc.Get(&config.Key{MspID: msp1, AppVersion: v1})
		require.Error(t, err)

		_, err = svc.Get(&config.Key{MspID: msp1, AppName: app1, AppVersion: v1, ComponentName: comp1})
		require.Error(t, err)

		_, err = svc.Get(&config.Key{MspID: msp1, AppName: app1, AppVersion: v1, ComponentVersion: v1})
		require.Error(t, err)
	})

	t.Run("Retriever error", func(t *testing.T) {
		errExpected := errors.New("retriever error")
		r.WithError(errExpected)
		defer func() { r.WithError(nil) }()

		value, err := svc.Get(key2)
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.Nil(t, value)
	})

	t.Run("One result", func(t *testing.T) {
		value, err := svc.Get(key2)
		require.NoError(t, err)
		require.Equal(t, cfg, value)
	})

	t.Run("No results", func(t *testing.T) {
		value, err := svc.Get(&config.Key{MspID: msp1, AppName: app2, AppVersion: v1})
		require.EqualError(t, err, ErrConfigNotFound.Error())
		require.Empty(t, value)
	})
}

func TestCache(t *testing.T) {
	rp := mocks.NewStateRetrieverProvider()

	svc := GetSvcMgr().ForChannel(channelID)
	require.Nil(t, svc)

	err := GetSvcMgr().Init(channelID, rp)
	require.NoError(t, err)

	svc = GetSvcMgr().ForChannel(channelID)
	require.NotNil(t, svc)

	require.EqualError(t, GetSvcMgr().Init(channelID, rp), "Config service already exists for channel [testchannel]")
}
