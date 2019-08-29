/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mgr

import (
	"encoding/json"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/state/api"
)

const (
	configNamespace = "configcc"

	msp1 = "org1MSP"
	msp2 = "org2MSP"

	peer1 = "peer1.example.com"
	peer2 = "peer2.example.com"

	app1 = "app1"
	app2 = "app2"
	app3 = "app3"

	comp1 = "comp1"
	comp2 = "comp2"
	comp3 = "comp3"

	v1 = "1"
	v2 = "2"

	txID1 = "tx1"
	txID2 = "tx2"
	txID3 = "tx3"

	msp1App1V1Config = "msp1-app1-v1-config"
	msp1App1V2Config = "msp1-app1-v2-config"
	msp1App2V1Config = `{"msp":"msp1", key1": "value1"}`
	msp1App2V2Config = `{"msp":"msp1", "key1_1": "value1"}`

	msp2App1V1Config = "msp2-app1-v1-config"
	msp2App1V2Config = "msp2-app1-v2-config"
	msp2App2V1Config = `{"msp":"msp2", key1": "value1"}`
	msp2App2V2Config = `{"msp":"msp2", "key1_1": "value1"}`

	msp1App1V1ConfigUpdate = `{"msp":"msp1", "key2": "value2"}`

	msp1Peer1App1V1Config = "org1-peer1-app1-v1-config"
	msp1Peer1App1V2Config = "org1-peer1-app1-v2-config"
	msp1Peer1App2V1Config = "org1-peer1-app2-v1-config"
	msp1Peer1App2V2Config = "org1-peer1-app2-v2-config"

	msp1Peer2App1V1Config = "org1-peer2-app1-v1-config"
	msp1Peer2App1V2Config = "org1-peer2-app1-v2-config"
	msp1Peer2App2V1Config = "org1-peer2-app2-v1-config"
	msp1Peer2App2V2Config = "org1-peer2-app2-v2-config"
	msp1Peer2App3V1Config = "org1-peer2-app3-v1-config"

	msp1Peer1App1V1ConfigUpdated = "org1-peer1-app1-v1-config-updated"

	peer1App1V1Comp1V1Config = "org1-peer1-app1-v1-comp1-v1-config"
	peer1App1V1Comp1V2Config = "org1-peer1-app1-v1-comp1-v2-config"
	peer1App1V1Comp2V1Config = "org1-peer1-app1-v1-comp2-v1-config"
	peer1App1V1Comp2V2Config = "org1-peer1-app1-v1-comp2-v2-config"

	peer2App1V1Comp1V1Config = "org1-peer2-app1-v1-comp1-v1-config"
	peer2App1V1Comp1V2Config = "org1-peer2-app1-v1-comp1-v2-config"
	peer2App1V1Comp2V1Config = "org1-peer2-app1-v1-comp2-v1-config"
	peer2App1V1Comp2V2Config = "org1-peer2-app1-v1-comp2-v2-config"

	peer1App1V1Comp1V1ConfigUpdate = "org1-peer1-app1-v1-comp1-v1-config-update"
	peer2App2V1Comp1V1Config       = "org1-peer2-app2-v1-comp1-v1-config"

	comp1V1Config = "comp1-v1-config"
	comp1V2Config = "comp1-v2-config"
	comp2V1Config = "comp2-v1-config"
	comp3V1Config = "comp3-v1-config"
	comp3V2Config = "comp3-v2-config"

	comp1V1ConfigUpdate = "comp1-v1-config-update"
)

var (
	msp1App1ComponentsConfig = &config.Config{
		MspID: msp1,
		Apps: []*config.App{
			{
				AppName: app1, Version: v1,
				Components: []*config.Component{
					{Name: comp1, Version: v1, Config: comp1V1Config, Format: config.FormatOther},
					{Name: comp1, Version: v2, Config: comp1V2Config, Format: config.FormatOther},
					{Name: comp2, Version: v1, Config: comp2V1Config, Format: config.FormatOther},
				},
			},
		},
	}
	msp1App1ComponentsConfigUpdate1 = &config.Config{
		MspID: msp1,
		Apps: []*config.App{
			{
				AppName: app1, Version: v1,
				Components: []*config.Component{
					{Name: comp3, Version: v1, Config: comp3V1Config, Format: config.FormatOther},
				},
			},
		},
	}
	msp1App1ComponentsConfigUpdate2 = &config.Config{
		MspID: msp1,
		Apps: []*config.App{
			{
				AppName: app1, Version: v1,
				Components: []*config.Component{
					{Name: comp1, Version: v1, Config: comp1V1ConfigUpdate, Format: config.FormatOther},
					{Name: comp3, Version: v2, Config: comp3V2Config, Format: config.FormatOther},
				},
			},
		},
	}
	msp1AppConfig = &config.Config{
		MspID: msp1,
		Apps: []*config.App{
			{AppName: app1, Version: v1, Config: msp1App1V1Config, Format: config.FormatOther},
			{AppName: app1, Version: v2, Config: msp1App1V2Config, Format: config.FormatOther},
			{AppName: app2, Version: v1, Config: msp1App2V1Config, Format: config.FormatJSON},
			{AppName: app2, Version: v2, Config: msp1App2V2Config, Format: config.FormatJSON},
		},
	}
	msp2AppConfig = &config.Config{
		MspID: msp2,
		Apps: []*config.App{
			{AppName: app1, Version: v1, Config: msp2App1V1Config, Format: config.FormatOther},
			{AppName: app1, Version: v2, Config: msp2App1V2Config, Format: config.FormatOther},
			{AppName: app2, Version: v1, Config: msp2App2V1Config, Format: config.FormatJSON},
			{AppName: app2, Version: v2, Config: msp2App2V2Config, Format: config.FormatJSON},
		},
	}
	msp1AppConfigUpdate = &config.Config{
		MspID: msp1,
		Apps: []*config.App{
			{AppName: app1, Version: v1, Config: msp1App1V1ConfigUpdate, Format: config.FormatJSON},
		},
	}
	peerConfig = &config.Config{
		MspID: msp1,
		Peers: []*config.Peer{
			{
				PeerID: peer1,
				Apps: []*config.App{
					{AppName: app1, Version: v1, Config: msp1Peer1App1V1Config, Format: config.FormatOther},
					{AppName: app1, Version: v2, Config: msp1Peer1App1V2Config, Format: config.FormatOther},
					{AppName: app2, Version: v1, Config: msp1Peer1App2V1Config, Format: config.FormatOther},
					{AppName: app2, Version: v2, Config: msp1Peer1App2V2Config, Format: config.FormatOther},
				},
			},
			{
				PeerID: peer2,
				Apps: []*config.App{
					{AppName: app1, Version: v1, Config: msp1Peer2App1V1Config, Format: config.FormatOther},
					{AppName: app1, Version: v2, Config: msp1Peer2App1V2Config, Format: config.FormatOther},
					{AppName: app2, Version: v1, Config: msp1Peer2App2V1Config, Format: config.FormatOther},
					{AppName: app2, Version: v2, Config: msp1Peer2App2V2Config, Format: config.FormatOther},
				},
			},
		},
	}
	peerConfigUpdate = &config.Config{
		MspID: msp1,
		Peers: []*config.Peer{
			{
				PeerID: peer1,
				Apps: []*config.App{
					{AppName: app1, Version: v1, Config: msp1Peer1App1V1ConfigUpdated, Format: config.FormatOther},
				},
			},
			{
				PeerID: peer2,
				Apps: []*config.App{
					{AppName: app3, Version: v1, Config: msp1Peer2App3V1Config, Format: config.FormatOther},
				},
			},
		},
	}
	peerComponentConfig = &config.Config{
		MspID: msp1,
		Peers: []*config.Peer{
			{
				PeerID: peer1,
				Apps: []*config.App{
					{
						AppName: app1, Version: v1,
						Components: []*config.Component{
							{Name: comp1, Version: v1, Config: peer1App1V1Comp1V1Config, Format: config.FormatOther},
							{Name: comp1, Version: v2, Config: peer1App1V1Comp1V2Config, Format: config.FormatOther},
							{Name: comp2, Version: v1, Config: peer1App1V1Comp2V1Config, Format: config.FormatOther},
							{Name: comp2, Version: v2, Config: peer1App1V1Comp2V2Config, Format: config.FormatOther},
						},
					},
				},
			},
			{
				PeerID: peer2,
				Apps: []*config.App{
					{
						AppName: app1, Version: v1,
						Components: []*config.Component{
							{Name: comp1, Version: v1, Config: peer2App1V1Comp1V1Config, Format: config.FormatOther},
							{Name: comp1, Version: v2, Config: peer2App1V1Comp1V2Config, Format: config.FormatOther},
							{Name: comp2, Version: v1, Config: peer2App1V1Comp2V1Config, Format: config.FormatOther},
							{Name: comp2, Version: v2, Config: peer2App1V1Comp2V2Config, Format: config.FormatOther},
						},
					},
				},
			},
		},
	}
	peerComponentConfigUpdate = &config.Config{
		MspID: msp1,
		Peers: []*config.Peer{
			{
				PeerID: peer1,
				Apps: []*config.App{
					{
						AppName: app1, Version: v1,
						Components: []*config.Component{
							{Name: comp1, Version: v1, Config: peer1App1V1Comp1V1ConfigUpdate, Format: config.FormatOther},
						},
					},
				},
			},
			{
				PeerID: peer2,
				Apps: []*config.App{
					{
						AppName: app2, Version: v1,
						Components: []*config.Component{
							{Name: comp1, Version: v1, Config: peer2App2V1Comp1V1Config, Format: config.FormatYAML},
						},
					},
				},
			},
		},
	}
	noAppName = &config.Config{
		MspID: msp1,
		Peers: []*config.Peer{
			{
				PeerID: peer1,
				Apps: []*config.App{
					{Version: v1, Config: "data"},
				},
			},
		},
	}
)

func TestUpdateManager_StoreError(t *testing.T) {
	t.Run("Provider error", func(t *testing.T) {
		expectedErr := errors.New("provider error")
		sp := mocks.NewStoreProvider().WithError(expectedErr)
		m := NewUpdateManager(configNamespace, sp)
		require.EqualError(t, m.Save("tx1", msp1App1ComponentsConfig), expectedErr.Error())
	})
	t.Run("StateStore error", func(t *testing.T) {
		expectedErr := errors.New("store error")
		s := mocks.NewStateStore().WithStoreError(expectedErr)
		sp := mocks.NewStoreProvider().WithStore(s)
		m := NewUpdateManager(configNamespace, sp)
		err := m.Save("tx1", msp1App1ComponentsConfig)
		require.Error(t, err)
		require.Contains(t, err.Error(), expectedErr.Error())
	})
	t.Run("StateRetriever error", func(t *testing.T) {
		expectedErr := errors.New("retriever error")
		s := mocks.NewStateStore().WithRetrieverError(expectedErr)
		sp := mocks.NewStoreProvider().WithStore(s)

		m := NewUpdateManager(configNamespace, sp)
		err := m.Save("tx1", msp1App1ComponentsConfig)
		require.NoError(t, err)

		_, err = s.GetState(configNamespace, "xxx")
		require.Error(t, err)
		require.Contains(t, err.Error(), expectedErr.Error())
	})
}

func TestUpdateManager_Save(t *testing.T) {
	sp := mocks.NewStoreProvider()
	r, err := sp.GetStateRetriever()
	require.NoError(t, err)

	m := NewUpdateManager(configNamespace, sp)

	t.Run("Invalid config", func(t *testing.T) {
		err := m.Save(txID1, noAppName)
		require.Error(t, err)
		require.Contains(t, err.Error(), "field [Name] is required")
	})
	t.Run("Apps", func(t *testing.T) {
		// Save MSP1
		require.NoError(t, m.Save(txID1, msp1AppConfig))
		requireEqualValue(t, r, config.NewAppKey(msp1, app1, v1), config.NewValue(txID1, msp1App1V1Config, config.FormatOther))
		requireEqualValue(t, r, config.NewAppKey(msp1, app1, v2), config.NewValue(txID1, msp1App1V2Config, config.FormatOther))
		requireEqualValue(t, r, config.NewAppKey(msp1, app2, v1), config.NewValue(txID1, msp1App2V1Config, config.FormatJSON))
		requireEqualValue(t, r, config.NewAppKey(msp1, app2, v2), config.NewValue(txID1, msp1App2V2Config, config.FormatJSON))

		// Save MSP2
		require.NoError(t, m.Save(txID2, msp2AppConfig))
		requireEqualValue(t, r, config.NewAppKey(msp2, app1, v1), config.NewValue(txID2, msp2App1V1Config, config.FormatOther))
		requireEqualValue(t, r, config.NewAppKey(msp2, app1, v2), config.NewValue(txID2, msp2App1V2Config, config.FormatOther))
		requireEqualValue(t, r, config.NewAppKey(msp2, app2, v1), config.NewValue(txID2, msp2App2V1Config, config.FormatJSON))
		requireEqualValue(t, r, config.NewAppKey(msp2, app2, v2), config.NewValue(txID2, msp2App2V2Config, config.FormatJSON))

		// Update app1 in msp1
		require.NoError(t, m.Save(txID3, msp1AppConfigUpdate))
		requireEqualValue(t, r, config.NewAppKey(msp1, app1, v1), config.NewValue(txID3, msp1App1V1ConfigUpdate, config.FormatJSON))

		// Ensure other data is still there
		requireEqualValue(t, r, config.NewAppKey(msp1, app1, v2), config.NewValue(txID1, msp1App1V2Config, config.FormatOther))
		requireEqualValue(t, r, config.NewAppKey(msp1, app2, v1), config.NewValue(txID1, msp1App2V1Config, config.FormatJSON))
		requireEqualValue(t, r, config.NewAppKey(msp1, app2, v2), config.NewValue(txID1, msp1App2V2Config, config.FormatJSON))
	})
	t.Run("App components", func(t *testing.T) {
		// Save MSP1 components
		require.NoError(t, m.Save(txID1, msp1App1ComponentsConfig))
		requireEqualValue(t, r, config.NewComponentKey(msp1, app1, v1, comp1, v1), config.NewValue(txID1, comp1V1Config, config.FormatOther))
		requireEqualValue(t, r, config.NewComponentKey(msp1, app1, v1, comp1, v2), config.NewValue(txID1, comp1V2Config, config.FormatOther))
		requireEqualValue(t, r, config.NewComponentKey(msp1, app1, v1, comp2, v1), config.NewValue(txID1, comp2V1Config, config.FormatOther))

		// Add comp3
		require.NoError(t, m.Save(txID2, msp1App1ComponentsConfigUpdate1))
		requireEqualValue(t, r, config.NewComponentKey(msp1, app1, v1, comp3, v1), config.NewValue(txID2, comp3V1Config, config.FormatOther))

		// Add comp3/v2 and update comp1/v1
		require.NoError(t, m.Save(txID3, msp1App1ComponentsConfigUpdate2))
		requireEqualValue(t, r, config.NewComponentKey(msp1, app1, v1, comp3, v2), config.NewValue(txID3, comp3V2Config, config.FormatOther))
		requireEqualValue(t, r, config.NewComponentKey(msp1, app1, v1, comp1, v1), config.NewValue(txID3, comp1V1ConfigUpdate, config.FormatOther))
	})
	t.Run("Peer apps", func(t *testing.T) {
		// Save peer apps
		require.NoError(t, m.Save(txID1, peerConfig))
		requireEqualValue(t, r, config.NewPeerKey(msp1, peer1, app1, v1), config.NewValue(txID1, msp1Peer1App1V1Config, config.FormatOther))
		requireEqualValue(t, r, config.NewPeerKey(msp1, peer1, app1, v2), config.NewValue(txID1, msp1Peer1App1V2Config, config.FormatOther))
		requireEqualValue(t, r, config.NewPeerKey(msp1, peer2, app1, v1), config.NewValue(txID1, msp1Peer2App1V1Config, config.FormatOther))
		requireEqualValue(t, r, config.NewPeerKey(msp1, peer2, app1, v2), config.NewValue(txID1, msp1Peer2App1V2Config, config.FormatOther))

		// Update peer1/app1/v1 and add peer2/app3/v1
		require.NoError(t, m.Save(txID2, peerConfigUpdate))
		requireEqualValue(t, r, config.NewPeerKey(msp1, peer1, app1, v1), config.NewValue(txID2, msp1Peer1App1V1ConfigUpdated, config.FormatOther))
		requireEqualValue(t, r, config.NewPeerKey(msp1, peer2, app3, v1), config.NewValue(txID2, msp1Peer2App3V1Config, config.FormatOther))

		// Ensure other data is still there
		requireEqualValue(t, r, config.NewPeerKey(msp1, peer1, app1, v2), config.NewValue(txID1, msp1Peer1App1V2Config, config.FormatOther))
		requireEqualValue(t, r, config.NewPeerKey(msp1, peer2, app1, v1), config.NewValue(txID1, msp1Peer2App1V1Config, config.FormatOther))
		requireEqualValue(t, r, config.NewPeerKey(msp1, peer2, app1, v2), config.NewValue(txID1, msp1Peer2App1V2Config, config.FormatOther))
	})
	t.Run("Peer app components", func(t *testing.T) {
		// Save MSP1 components
		require.NoError(t, m.Save(txID1, peerComponentConfig))
		requireEqualValue(t, r, config.NewPeerComponentKey(msp1, peer1, app1, v1, comp1, v1), config.NewValue(txID1, peer1App1V1Comp1V1Config, config.FormatOther))
		requireEqualValue(t, r, config.NewPeerComponentKey(msp1, peer1, app1, v1, comp1, v2), config.NewValue(txID1, peer1App1V1Comp1V2Config, config.FormatOther))
		requireEqualValue(t, r, config.NewPeerComponentKey(msp1, peer1, app1, v1, comp2, v1), config.NewValue(txID1, peer1App1V1Comp2V1Config, config.FormatOther))
		requireEqualValue(t, r, config.NewPeerComponentKey(msp1, peer1, app1, v1, comp2, v2), config.NewValue(txID1, peer1App1V1Comp2V2Config, config.FormatOther))

		requireEqualValue(t, r, config.NewPeerComponentKey(msp1, peer2, app1, v1, comp1, v1), config.NewValue(txID1, peer2App1V1Comp1V1Config, config.FormatOther))
		requireEqualValue(t, r, config.NewPeerComponentKey(msp1, peer2, app1, v1, comp1, v2), config.NewValue(txID1, peer2App1V1Comp1V2Config, config.FormatOther))
		requireEqualValue(t, r, config.NewPeerComponentKey(msp1, peer2, app1, v1, comp2, v1), config.NewValue(txID1, peer2App1V1Comp2V1Config, config.FormatOther))
		requireEqualValue(t, r, config.NewPeerComponentKey(msp1, peer2, app1, v1, comp2, v2), config.NewValue(txID1, peer2App1V1Comp2V2Config, config.FormatOther))

		// Update peer1/app1/v1/comp1/v1 and add peer2/app2/v1/comp1/v1
		require.NoError(t, m.Save(txID2, peerComponentConfigUpdate))
		requireEqualValue(t, r, config.NewPeerComponentKey(msp1, peer1, app1, v1, comp1, v1), config.NewValue(txID2, peer1App1V1Comp1V1ConfigUpdate, config.FormatOther))
		requireEqualValue(t, r, config.NewPeerComponentKey(msp1, peer2, app2, v1, comp1, v1), config.NewValue(txID2, peer2App2V1Comp1V1Config, config.FormatYAML))
	})
}

func TestUpdateManager_Delete(t *testing.T) {
	m := NewUpdateManager(configNamespace, mocks.NewStoreProvider())
	require.NotNil(t, m)

	require.NoError(t, m.Save(txID1, msp1PeerConfig))
	require.NoError(t, m.Save(txID1, msp1App1ComponentsConfig))
	require.NoError(t, m.Save(txID1, msp1App1ComponentsConfigUpdate))
	require.NoError(t, m.Save(txID1, msp1App1ComponentsConfigUpdate2))
	require.NoError(t, m.Save(txID1, msp2Peer2Config))

	t.Run("Invalid criteria -> error", func(t *testing.T) {
		_, err := m.Query(&config.Criteria{MspID: msp1, PeerID: peer1, AppVersion: v1})
		require.EqualError(t, err, "field [Name] is required")
	})

	t.Run("StateStoreProvider error", func(t *testing.T) {
		expectedErr := errors.New("store provider error")
		sp := mocks.NewStoreProvider()
		m := NewUpdateManager(configNamespace, sp)
		require.NoError(t, m.Save(txID1, msp1PeerConfig))
		sp.WithError(expectedErr)
		err := m.Delete(&config.Criteria{MspID: msp1})
		require.Error(t, err)
		require.Contains(t, err.Error(), expectedErr.Error())
	})

	t.Run("StateStore error", func(t *testing.T) {
		expectedErr := errors.New("store error")
		s := mocks.NewStateStore()
		sp := mocks.NewStoreProvider().WithStore(s)
		m := NewUpdateManager(configNamespace, sp)
		require.NoError(t, m.Save(txID1, msp1PeerConfig))
		s.WithStoreError(expectedErr)
		err := m.Delete(&config.Criteria{MspID: msp1})
		require.Error(t, err)
		require.Contains(t, err.Error(), expectedErr.Error())
	})

	t.Run("PeerID app config -> success", func(t *testing.T) {
		key := &config.Criteria{MspID: msp1, PeerID: peer1, AppName: app3, AppVersion: v1}
		config, err := m.Query(key)
		require.NoError(t, err)
		require.NotEmpty(t, config)
		require.NotNil(t, config[0].Value)

		require.NoError(t, m.Delete(key))
		config, err = m.Query(key)
		require.Empty(t, config)
	})

	t.Run("Apps component -> success", func(t *testing.T) {
		criteria := &config.Criteria{MspID: msp1, AppName: app1, AppVersion: v1, ComponentName: comp1, ComponentVersion: v1}
		cfg, err := m.Query(criteria)
		require.NoError(t, err)
		require.Equal(t, 1, len(cfg))
		require.NotNil(t, cfg[0].Value)

		require.NoError(t, m.Delete(criteria))

		cfg, err = m.Query(criteria)
		require.NoError(t, err)
		require.Empty(t, cfg)

		cfg, err = m.Query(&config.Criteria{MspID: msp1, AppName: app1, AppVersion: v1, ComponentName: comp1, ComponentVersion: v2})
		require.NoError(t, err)
		require.NotEmpty(t, cfg)
	})

	t.Run("ComponentName config (all versions) -> success", func(t *testing.T) {
		k := &config.Criteria{MspID: msp1, AppName: app1, AppVersion: v1, ComponentName: comp3}
		cfg, err := m.Query(k)
		require.NoError(t, err)
		require.Equal(t, 2, len(cfg))
		require.NoError(t, m.Delete(k))

		cfg, err = m.Query(k)
		require.NoError(t, err)
		require.Empty(t, cfg)

		k = &config.Criteria{MspID: msp1, AppName: app1, AppVersion: v1, ComponentName: comp3}
		cfg, err = m.Query(k)
		require.NoError(t, err)
		require.Empty(t, cfg)
	})

	t.Run("MSP config -> success", func(t *testing.T) {
		key := &config.Criteria{MspID: msp2}
		config, err := m.Query(key)
		require.NoError(t, err)
		require.NotEmpty(t, config)

		require.NoError(t, m.Delete(key))
		config, err = m.Query(key)
		require.Empty(t, config)
	})
}

func requireEqualValue(t *testing.T, retriever api.StateRetriever, key *config.Key, expectedValue *config.Value) {
	bytes, err := retriever.GetState(configNamespace, marshalKey(*key))
	require.NoError(t, err)
	require.NotEmpty(t, bytes)

	value := &config.Value{}
	require.NoError(t, json.Unmarshal(bytes, value))
	require.Equal(t, expectedValue, value)
}
