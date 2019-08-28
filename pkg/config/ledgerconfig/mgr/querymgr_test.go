/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mgr

import (
	"testing"

	commonledger "github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
	configmocks "github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

const (
	app4 = "app4"
	app5 = "app5"
	app6 = "app6"

	app2Config = "app2 data"

	msp1Peer1App3V1Config = "org1-peer1-app3-v1-config"
	msp1Peer1App3V2Config = "org1-peer1-app3-v2-config"
	msp1Peer1App4V1Config = "org1-peer1-app4-v1-config"
	msp1Peer1App4V2Config = "org1-peer1-app4-v2-config"
	msp1Peer1App6V1Config = "org1-peer1-app6-v1-config"

	msp1Peer2App3V2Config = "org1-peer2-app3-v2-config"
	msp1Peer2App4V1Config = "org1-peer2-app4-v1-config"
	msp1Peer2App4V2Config = "org1-peer2-app4-v2-config"

	msp1Peer1App3V1ConfigUpdated = "org1-peer1-app3-v1-config-updated"

	msp1Peer1App5Comp1V1Config = "org1-peer1-app5-v1-comp1-v1-config"
	msp1Peer1App5Comp1V2Config = "org1-peer1-app5-v1-comp1-v2-config"
	msp1Peer1App5Comp2V1Config = "org1-peer1-app5-v1-comp2-v1-config"

	msp1Peer2App5Comp1V1Config = "org1-peer2-app5-v1-comp1-v1-config"
	msp1Peer2App5Comp1V2Config = "org1-peer2-app5-v1-comp1-v2-config"
	msp1Peer2App5Comp2V1Config = "org1-peer2-app5-v1-comp2-v1-config"

	msp2Peer2App3V1Config = "org2-peer2-app3-v1-config"
	msp2Peer2App3V2Config = "org2-peer2-app3-v2-config"
	msp2Peer2App4V1Config = "org2-peer2-app4-v1-config"
	msp2Peer2App4V2Config = "org2-peer2-app4-v2-config"
	msp2Peer2App5V1Config = "org2-peer2-app5-v1-config"
)

var (
	msp1PeerConfig = &config.Config{
		MspID: msp1,
		Peers: []*config.Peer{
			{
				PeerID: peer1,
				Apps: []*config.App{
					{AppName: app3, Version: v1, Config: msp1Peer1App3V1Config, Format: config.FormatOther},
					{AppName: app3, Version: v2, Config: msp1Peer1App3V2Config, Format: config.FormatOther},
					{AppName: app4, Version: v1, Config: msp1Peer1App4V1Config, Format: config.FormatOther},
					{AppName: app4, Version: v2, Config: msp1Peer1App4V2Config, Format: config.FormatOther},
					{
						AppName: app5, Version: v1,
						Components: []*config.Component{
							{Name: comp1, Version: v1, Config: msp1Peer1App5Comp1V1Config, Format: config.FormatOther},
							{Name: comp1, Version: v2, Config: msp1Peer1App5Comp1V2Config, Format: config.FormatOther},
							{Name: comp2, Version: v1, Config: msp1Peer1App5Comp2V1Config, Format: config.FormatOther},
						},
					},
				},
			},
			{
				PeerID: peer2,
				Apps: []*config.App{
					{AppName: app3, Version: v1, Config: msp1Peer2App3V1Config, Format: config.FormatOther},
					{AppName: app3, Version: v2, Config: msp1Peer2App3V2Config, Format: config.FormatOther},
					{AppName: app4, Version: v1, Config: msp1Peer2App4V1Config, Format: config.FormatOther},
					{AppName: app4, Version: v2, Config: msp1Peer2App4V2Config, Format: config.FormatOther},
					{
						AppName: app5, Version: v1,
						Components: []*config.Component{
							{Name: comp1, Version: v1, Config: msp1Peer2App5Comp1V1Config, Format: config.FormatOther},
							{Name: comp1, Version: v2, Config: msp1Peer2App5Comp1V2Config, Format: config.FormatOther},
							{Name: comp2, Version: v1, Config: msp1Peer2App5Comp2V1Config, Format: config.FormatOther},
						},
					},
				},
			},
		},
	}
	msp1App1ComponentsConfigUpdate = &config.Config{
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
	msp1App2Config = &config.Config{
		MspID: msp1,
		Apps: []*config.App{
			{AppName: app2, Version: v1, Config: app2Config, Format: config.FormatOther},
		},
	}
	msp1Peer1ConfigUpdate = &config.Config{
		MspID: msp1,
		Peers: []*config.Peer{
			{
				PeerID: peer1,
				Apps: []*config.App{
					{AppName: app3, Version: v1, Config: msp1Peer1App3V1ConfigUpdated, Format: config.FormatOther},
					{AppName: app6, Version: v1, Config: msp1Peer1App6V1Config, Format: config.FormatOther},
				},
			},
		},
	}
	msp2Peer2Config = &config.Config{
		MspID: msp2,
		Peers: []*config.Peer{
			{
				PeerID: peer2,
				Apps: []*config.App{
					{AppName: app3, Version: v1, Config: msp2Peer2App3V1Config, Format: config.FormatOther},
					{AppName: app3, Version: v2, Config: msp2Peer2App3V2Config, Format: config.FormatOther},
					{AppName: app4, Version: v1, Config: msp2Peer2App4V1Config, Format: config.FormatOther},
					{AppName: app4, Version: v2, Config: msp2Peer2App4V2Config, Format: config.FormatOther},
					{AppName: app5, Version: v1, Config: msp2Peer2App5V1Config, Format: config.FormatOther},
				},
			},
		},
	}
)

func TestManager_QueryExecutorError(t *testing.T) {
	t.Run("Provider error", func(t *testing.T) {
		expectedErr := errors.New("provider error")
		sp := configmocks.NewStateRetrieverProvider().WithError(expectedErr)
		m := NewQueryManager(configNamespace, sp)

		// Query by unique key
		_, err := m.Query(&config.Criteria{MspID: msp1, AppName: app1, AppVersion: v1})
		require.EqualError(t, err, expectedErr.Error())

		// Query by search
		_, err = m.Query(&config.Criteria{MspID: msp1})
		require.EqualError(t, err, expectedErr.Error())
	})
	t.Run("State retriever error", func(t *testing.T) {
		expectedErr := errors.New("getState error")
		r := configmocks.NewStateRetriever()
		// Add an index so that the query finds one item
		r.WithState(configNamespace, getIndexKey(MarshalKey(&config.Key{MspID: msp1}), []string{msp1}), []byte("{}"))
		r.WithError(expectedErr)
		qep := configmocks.NewStateRetrieverProvider().WithStateRetriever(r)

		m := NewQueryManager(configNamespace, qep)

		// Query by unique key
		_, err := m.Query(&config.Criteria{MspID: msp1, AppName: app1, AppVersion: v1})
		require.Error(t, err)
		require.Contains(t, err.Error(), expectedErr.Error())

		// Query by search
		_, err = m.Query(&config.Criteria{MspID: msp1})
		require.Error(t, err)
		require.Contains(t, err.Error(), expectedErr.Error())
	})
	t.Run("State retriever query error", func(t *testing.T) {
		expectedErr := errors.New("query error")
		r := configmocks.NewStateRetriever()
		r.WithQueryError(expectedErr)
		qep := configmocks.NewStateRetrieverProvider().WithStateRetriever(r)

		m := NewQueryManager(configNamespace, qep)
		_, err := m.Query(&config.Criteria{MspID: msp1})
		require.Error(t, err)
		require.Contains(t, err.Error(), expectedErr.Error())
	})
	t.Run("Iterator error", func(t *testing.T) {
		expectedErr := errors.New("iterator error")
		r := configmocks.NewStateRetriever()
		r.WithKVIteratorProvider(func(kvs []*statedb.VersionedKV) *mocks.KVIterator {
			return mocks.NewKVIterator(kvs).WithError(expectedErr)
		})
		m := NewQueryManager(configNamespace, configmocks.NewStateRetrieverProvider().WithStateRetriever(r))
		_, err := m.Query(&config.Criteria{MspID: msp1})
		require.Error(t, err)
		require.Contains(t, err.Error(), expectedErr.Error())
	})
	t.Run("Iterator Close error", func(t *testing.T) {
		expectedErr := errors.New("iterator close error")
		r := configmocks.NewStateRetriever()
		r.WithKVResultsIteratorProvider(func(it commonledger.ResultsIterator) *configmocks.KVResultsIter {
			return configmocks.NewKVResultsIter(it).WithCloseError(expectedErr)
		})
		m := NewQueryManager(configNamespace, configmocks.NewStateRetrieverProvider().WithStateRetriever(r))
		_, err := m.Query(&config.Criteria{MspID: msp1})
		require.NoError(t, err) // Should just log a message
	})
}

func TestManager_Search_AppConfig(t *testing.T) {
	m := NewUpdateManager(configNamespace, configmocks.NewStoreProvider())
	require.NotNil(t, m)

	require.NoError(t, m.Save(txID1, msp1App1ComponentsConfig))

	t.Run("No MspID -> error", func(t *testing.T) {
		_, err := m.query(&config.Criteria{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "field [MspID] is required")
	})

	t.Run("MspID not found -> empty result", func(t *testing.T) {
		results, err := m.Query(&config.Criteria{MspID: "xxx"})
		require.NoError(t, err)
		require.Empty(t, results)
	})

	t.Run("No Name with AppVersion -> error", func(t *testing.T) {
		results, err := m.Query(&config.Criteria{MspID: msp1, AppVersion: v1})
		require.Error(t, err)
		require.Contains(t, err.Error(), "field [Name] is required")
		require.Empty(t, results)
	})

	t.Run("No Name with Component -> error", func(t *testing.T) {
		results, err := m.Query(&config.Criteria{MspID: msp1, ComponentName: comp1})
		require.Error(t, err)
		require.Contains(t, err.Error(), "field [Name] is required")
		require.Empty(t, results)
	})

	t.Run("No ComponentName with ComponentVersion -> error", func(t *testing.T) {
		results, err := m.Query(&config.Criteria{MspID: msp1, AppName: app1, ComponentVersion: v1})
		require.Error(t, err)
		require.Contains(t, err.Error(), "field [ComponentName] is required")
		require.Empty(t, results)
	})

	t.Run("Unique key not found -> success", func(t *testing.T) {
		criteria := &config.Criteria{MspID: msp1, AppName: app2, AppVersion: v2, ComponentName: comp2, ComponentVersion: v2}
		results, err := m.Query(criteria)
		require.NoError(t, err)
		require.Empty(t, len(results))
	})

	t.Run("By component and version -> success", func(t *testing.T) {
		criteria := &config.Criteria{MspID: msp1, AppName: app1, AppVersion: v1, ComponentName: comp1, ComponentVersion: v1}
		results, err := m.Query(criteria)
		require.NoError(t, err)
		require.Equal(t, 1, len(results))
		requireEqualConfigData(t, results[0].Value, comp1, v1, txID1, comp1V1Config)
	})

	t.Run("By component no version -> success", func(t *testing.T) {
		criteria := &config.Criteria{MspID: msp1, AppName: app1, AppVersion: v1, ComponentName: comp1}
		results, err := m.Query(criteria)
		require.NoError(t, err)
		require.Equal(t, 2, len(results))
	})

	t.Run("Update with new component -> success", func(t *testing.T) {
		require.NoError(t, m.Save(txID1, msp1App1ComponentsConfigUpdate))
		criteria := &config.Criteria{MspID: msp1, AppName: app1, AppVersion: v1, ComponentName: comp3}
		results, err := m.Query(criteria)
		require.NoError(t, err)
		require.Equal(t, 1, len(results))
		requireEqualConfigData(t, results[0].Value, comp3, v1, txID1, comp3V1Config)

		// Make sure the previous components are still there
		criteria = &config.Criteria{MspID: msp1, AppName: app1, AppVersion: v1, ComponentName: comp1}
		results, err = m.Query(criteria)
		require.NoError(t, err)
		require.Equal(t, 2, len(results))
	})

	t.Run("Update with new app -> success", func(t *testing.T) {
		require.NoError(t, m.Save(txID1, msp1App2Config))
		criteria := &config.Criteria{MspID: msp1, AppName: app2, AppVersion: v1}
		results, err := m.Query(criteria)
		require.NoError(t, err)
		require.Equal(t, 1, len(results))
		requireEqualConfigData(t, results[0].Value, app2, v1, txID1, app2Config)
	})
}

func TestManager_Search_PeerConfig(t *testing.T) {
	m := NewUpdateManager(configNamespace, configmocks.NewStoreProvider())
	require.NotNil(t, m)
	require.NoError(t, m.Save(txID1, msp1PeerConfig))

	t.Run("No MSP -> error", func(t *testing.T) {
		_, err := m.Query(&config.Criteria{})
		require.EqualError(t, err, "MspID is required")
	})

	t.Run("MSP -> success", func(t *testing.T) {
		results, err := m.Query(&config.Criteria{MspID: msp1})
		require.NoError(t, err)
		require.Equal(t, 14, len(results))
	})

	t.Run("Peer-app-version -> success", func(t *testing.T) {
		criteria := &config.Criteria{MspID: msp1, PeerID: peer1, AppName: app3, AppVersion: v1}
		results, err := m.Query(criteria)
		require.NoError(t, err)
		require.Equal(t, 1, len(results))
		requireEqualConfigData(t, results[0].Value, app3, v1, txID1, msp1Peer1App3V1Config)

		criteria.AppVersion = v2
		results, err = m.Query(criteria)
		require.NoError(t, err)
		require.Equal(t, 1, len(results))
		requireEqualConfigData(t, results[0].Value, app3, v2, txID1, msp1Peer1App3V2Config)

		criteria = &config.Criteria{MspID: msp1, PeerID: peer1, AppName: app3, AppVersion: v1}
		results, err = m.Query(criteria)
		require.NoError(t, err)
		require.Equal(t, 1, len(results))
		requireEqualConfigData(t, results[0].Value, app3, v1, txID1, msp1Peer1App3V1Config)

		criteria.AppVersion = v2
		results, err = m.Query(criteria)
		require.NoError(t, err)
		require.Equal(t, 1, len(results))
		requireEqualConfigData(t, results[0].Value, app3, v2, txID1, msp1Peer1App3V2Config)
	})

	t.Run("Peer-app-component-version -> success", func(t *testing.T) {
		results, err := m.Query(&config.Criteria{MspID: msp1, PeerID: peer1, AppName: app5, AppVersion: v1, ComponentName: comp1})
		require.NoError(t, err)
		require.Equal(t, 2, len(results))

		results, err = m.Query(&config.Criteria{MspID: msp1, PeerID: peer1, AppName: app5, AppVersion: v1, ComponentName: comp1, ComponentVersion: v1})
		require.NoError(t, err)
		require.Equal(t, 1, len(results))
		requireEqualConfigData(t, results[0].Value, comp1, v1, txID1, msp1Peer1App5Comp1V1Config)

		results, err = m.Query(&config.Criteria{MspID: msp1, PeerID: peer1, AppName: app5, AppVersion: v1, ComponentName: comp1, ComponentVersion: v2})
		require.NoError(t, err)
		require.Equal(t, 1, len(results))
		requireEqualConfigData(t, results[0].Value, comp1, v2, txID1, msp1Peer1App5Comp1V2Config)

		results, err = m.Query(&config.Criteria{MspID: msp1, PeerID: peer1, AppName: app5, AppVersion: v1, ComponentName: comp2, ComponentVersion: v1})
		require.NoError(t, err)
		require.Equal(t, 1, len(results))
		requireEqualConfigData(t, results[0].Value, comp2, v1, txID1, msp1Peer1App5Comp2V1Config)
	})

	t.Run("Update peer config -> success", func(t *testing.T) {
		require.NoError(t, m.Save(txID1, msp1Peer1ConfigUpdate))
		results, err := m.Query(&config.Criteria{MspID: msp1, PeerID: peer1, AppName: app3, AppVersion: v1})
		require.NoError(t, err)
		require.Equal(t, 1, len(results))
		requireEqualConfigData(t, results[0].Value, app3, v1, txID1, msp1Peer1App3V1ConfigUpdated)

		results, err = m.Query(&config.Criteria{MspID: msp1, PeerID: peer1, AppName: "app6", AppVersion: v1})
		require.NoError(t, err)
		require.Equal(t, 1, len(results))
		requireEqualConfigData(t, results[0].Value, app6, v1, txID1, msp1Peer1App6V1Config)
	})

	t.Run("Update with new MSP -> success", func(t *testing.T) {
		require.NoError(t, m.Save(txID1, msp2Peer2Config))

		results, err := m.Query(&config.Criteria{MspID: msp2, PeerID: peer2, AppName: app3})
		require.NoError(t, err)
		require.Equal(t, 2, len(results))

		results, err = m.Query(&config.Criteria{MspID: msp2, PeerID: peer2, AppName: app4, AppVersion: v2})
		require.NoError(t, err)
		require.Equal(t, 1, len(results))
		requireEqualConfigData(t, results[0].Value, app4, v2, txID1, msp2Peer2App4V2Config)

		// MSP1 config should still be there
		results, err = m.Query(&config.Criteria{MspID: msp1, PeerID: peer2, AppName: app3, AppVersion: v1})
		require.NoError(t, err)
		require.Equal(t, 1, len(results))
		requireEqualConfigData(t, results[0].Value, app3, v1, txID1, msp1Peer2App3V1Config)
	})
}

func requireEqualConfigData(t *testing.T, app *config.Value, name string, version string, txID string, cfg string) {
	require.Equal(t, txID, app.TxID)
	require.Equal(t, cfg, app.Config)
}
