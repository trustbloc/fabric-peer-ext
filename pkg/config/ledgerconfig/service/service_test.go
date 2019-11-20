/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/hyperledger/fabric/protos/peer"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/mgr"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/gossip/blockpublisher"
	mocks2 "github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

const (
	channelID = "testchannel"
	msp1      = "org1MSP"
	peer1     = "peer1.example.com"

	app1 = "app1"
	app2 = "app2"
	app3 = "app3"
	app4 = "app4"

	comp1 = "comp1"
	v1    = "1"

	tx1 = "tx1"
	tx2 = "tx2"
	tx3 = "tx3"

	config1        = "config1"
	config2        = "config2"
	config3        = "config3"
	config1Updated = "config1-updated"
)

func TestConfigService_Get(t *testing.T) {
	key2 := &config.Key{MspID: msp1, AppName: app1, AppVersion: v1}
	cfg := config.NewValue(tx1, config1, config.FormatOther)
	bytes, err := json.Marshal(cfg)
	require.NoError(t, err)

	r := mocks.NewStateRetriever()
	r.WithState(ConfigNS, mgr.MarshalKey(key2), bytes)
	p := mocks.NewStateRetrieverProvider().WithStateRetriever(r)

	svc := New(channelID, p, mocks2.NewBlockPublisher())
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

func TestConfigService_CacheUpdate(t *testing.T) {
	r := mocks.NewStateRetriever()
	p := mocks.NewStateRetrieverProvider().WithStateRetriever(r)

	publisher := blockpublisher.New(channelID)
	svc := New(channelID, p, publisher)
	require.NotNil(t, svc)

	key1 := config.NewPeerComponentKey(msp1, peer1, app1, v1, comp1, v1)
	_, err := svc.Get(key1)
	require.EqualError(t, err, ErrConfigNotFound.Error())

	val1 := config.NewValue(tx1, config1, config.FormatOther)
	val1Bytes, err := json.Marshal(val1)
	require.NoError(t, err)

	key2 := config.NewPeerComponentKey(msp1, peer1, app2, v1, comp1, v1)
	_, err = svc.Get(key2)
	require.EqualError(t, err, ErrConfigNotFound.Error())
	val2 := config.NewValue(tx1, config2, config.FormatOther)
	val2Bytes, err := json.Marshal(val2)
	require.NoError(t, err)

	key3 := config.NewPeerComponentKey(msp1, peer1, app3, v1, comp1, v1)
	val3 := config.NewValue(tx1, config2, config.FormatOther)
	val3Bytes, err := json.Marshal(val3)
	require.NoError(t, err)

	key4 := config.NewPeerComponentKey(msp1, peer1, app4, v1, comp1, v1)

	// Add key1 and key2
	b := mocks2.NewBlockBuilder(channelID, 1000)
	txb := b.Transaction(tx1, peer.TxValidationCode_VALID)
	txb.ChaincodeAction(ConfigNS).
		Write(mgr.MarshalKey(key1), val1Bytes).
		Write(mgr.MarshalKey(key2), val2Bytes).
		// Invalid keys should be ignored
		Write("some-other-key", []byte("some value")).
		Write(mgr.MarshalKey(key4), []byte("invalid JSON")) // Invalid JSON should be logged and ignored
	// Other namespaces should be ignored
	txb.ChaincodeAction("some-other-cc").
		Write(mgr.MarshalKey(key3), val3Bytes)
	publisher.Publish(b.Build())

	// Give the event enough time to be processed
	time.Sleep(100 * time.Millisecond)

	value, err := svc.Get(key1)
	require.NoError(t, err)
	require.Equal(t, val1, value)

	value, err = svc.Get(key2)
	require.NoError(t, err)
	require.Equal(t, val2, value)

	value, err = svc.Get(key3)
	require.Nil(t, value)
	require.EqualError(t, err, ErrConfigNotFound.Error())

	val1Updated := config.NewValue(tx1, config1Updated, config.FormatOther)
	v1UpdatedBytes, err := json.Marshal(val1Updated)
	require.NoError(t, err)

	// Update key1, delete key2, add key3, delete key4 (which isn't in the cache)
	b = mocks2.NewBlockBuilder(channelID, 1001)
	b.Transaction(tx2, peer.TxValidationCode_VALID).
		ChaincodeAction(ConfigNS).
		Write(mgr.MarshalKey(key1), v1UpdatedBytes).
		Delete(mgr.MarshalKey(key2)).
		Write(mgr.MarshalKey(key3), val3Bytes).
		Delete(mgr.MarshalKey(key4))
	publisher.Publish(b.Build())

	// Give the event enough time to be processed
	time.Sleep(100 * time.Millisecond)

	value, err = svc.Get(key1)
	require.NoError(t, err)
	require.Equal(t, val1Updated, value)

	value, err = svc.Get(key2)
	require.Nil(t, value)
	require.EqualError(t, err, ErrConfigNotFound.Error())

	value, err = svc.Get(key3)
	require.NoError(t, err)
	require.Equal(t, val3, value)

	value, err = svc.Get(key4)
	require.Nil(t, value)
	require.EqualError(t, err, ErrConfigNotFound.Error())
}

func TestConfigService_AddUpdateHandler(t *testing.T) {
	r := mocks.NewStateRetriever()
	p := mocks.NewStateRetrieverProvider().WithStateRetriever(r)

	publisher := blockpublisher.New(channelID)
	svc := New(channelID, p, publisher)
	require.NotNil(t, svc)

	key1 := config.NewPeerComponentKey(msp1, peer1, app1, v1, comp1, v1)
	_, err := svc.Get(key1)
	require.EqualError(t, err, ErrConfigNotFound.Error())

	val1 := config.NewValue(tx1, config1, config.FormatOther)
	val1Bytes, err := json.Marshal(val1)
	require.NoError(t, err)

	key2 := config.NewPeerComponentKey(msp1, peer1, app2, v1, comp1, v1)
	_, err = svc.Get(key2)
	require.EqualError(t, err, ErrConfigNotFound.Error())
	val2 := config.NewValue(tx1, config2, config.FormatOther)
	val2Bytes, err := json.Marshal(val2)
	require.NoError(t, err)

	key3 := config.NewPeerComponentKey(msp1, peer1, app3, v1, comp1, v1)

	updates := make(map[config.Key]*config.Value)
	var mutex sync.Mutex
	svc.AddUpdateHandler(func(kv *config.KeyValue) {
		mutex.Lock()
		defer mutex.Unlock()
		updates[*kv.Key] = kv.Value
	})

	// Add key1 and key2 and delete key3
	b := mocks2.NewBlockBuilder(channelID, 1000)
	txb := b.Transaction(tx1, peer.TxValidationCode_VALID)
	txb.ChaincodeAction(ConfigNS).
		Write(mgr.MarshalKey(key1), val1Bytes).
		Write(mgr.MarshalKey(key2), val2Bytes).
		Delete(mgr.MarshalKey(key3))
	publisher.Publish(b.Build())

	// Give the event enough time to be processed
	time.Sleep(100 * time.Millisecond)

	mutex.Lock()
	require.Equal(t, val1, updates[*key1])
	require.Equal(t, val2, updates[*key2])
	v, ok := updates[*key3]
	require.True(t, ok)
	require.Nil(t, v)
	mutex.Unlock()
}

func TestManager(t *testing.T) {
	lp := &mocks2.LedgerProvider{}
	lp.GetLedgerReturns(&mocks2.Ledger{QueryExecutor: mocks2.NewQueryExecutor()})

	manager := NewSvcMgr(lp, mocks2.NewBlockPublisherProvider())

	svc := manager.ForChannel(channelID)
	require.NotNil(t, svc)

	value, err := svc.Get(&config.Key{MspID: msp1, AppName: app2, AppVersion: v1})
	require.EqualError(t, err, ErrConfigNotFound.Error())
	require.Nil(t, value)
}
