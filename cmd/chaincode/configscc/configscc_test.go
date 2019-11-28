/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package configscc

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
	configmocks "github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/service"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/state/api"
)

const (
	tx1     = "tx_1"
	org1MSP = "org1MSP"
)

func TestConfigSCC_New(t *testing.T) {
	cc := New()
	require.NotNil(t, cc)

	require.Equal(t, service.ConfigNS, cc.Name())
	require.Equal(t, cc, cc.Chaincode())
}

func TestConfigSCC_Init(t *testing.T) {
	cc := New()
	require.NotNil(t, cc)

	t.Run("System channel", func(t *testing.T) {
		stub := shimtest.NewMockStub("mock_stub", cc.Chaincode())
		r := stub.MockInit(tx1, nil)
		require.NotNil(t, r)
		require.Equal(t, shim.OK, int(r.Status))
		require.Nil(t, r.Payload)
		require.Empty(t, r.Message)
	})

	t.Run("With channel", func(t *testing.T) {
		const channelID = "testchannel"

		stub := shimtest.NewMockStub("mock_stub", cc.Chaincode())
		stub.ChannelID = channelID
		r := stub.MockInit(tx1, nil)
		require.NotNil(t, r)
		require.Equal(t, shim.OK, int(r.Status))
		require.Nil(t, r.Payload)
		require.Empty(t, r.Message)
	})
}

func TestConfigSCC_Invoke_Invalid(t *testing.T) {
	cc := New()
	require.NotNil(t, cc)

	t.Run("No func arg", func(t *testing.T) {
		r := shimtest.NewMockStub("mock_stub", cc.Chaincode()).MockInvoke(tx1, nil)
		require.NotNil(t, r)
		require.Equal(t, shim.ERROR, int(r.Status))
		require.Nil(t, r.Payload)
		require.Contains(t, r.Message, "Function not provided")
	})

	t.Run("Invalid func arg", func(t *testing.T) {
		r := shimtest.NewMockStub("mock_stub", cc.Chaincode()).MockInvoke(tx1, [][]byte{[]byte("invalid_func")})
		require.NotNil(t, r)
		require.Equal(t, shim.ERROR, int(r.Status))
		require.Nil(t, r.Payload)
		require.Contains(t, r.Message, "Invalid function")
	})
}

func TestConfigSCC_Invoke_Save(t *testing.T) {
	cc := New()
	require.NotNil(t, cc)

	t.Run("Empty config", func(t *testing.T) {
		r := shimtest.NewMockStub("mock_stub", cc.Chaincode()).MockInvoke(tx1, [][]byte{[]byte("save")})
		require.NotNil(t, r)
		require.Equal(t, shim.ERROR, int(r.Status))
		require.Nil(t, r.Payload)
		require.Equal(t, "config is empty - cannot be saved", r.Message)
	})

	t.Run("Valid number of args", func(t *testing.T) {
		prevProvider := getConfigMgr
		defer func() { getConfigMgr = prevProvider }()

		getConfigMgr = func(ns string, sp api.StoreProvider) configMgr {
			return configmocks.NewConfigMgr()
		}

		r := shimtest.NewMockStub("mock_stub", cc.Chaincode()).MockInvoke(tx1, [][]byte{[]byte("save"), []byte(`{}`)})
		require.NotNil(t, r)
		require.Equal(t, shim.OK, int(r.Status))
	})

	t.Run("Marshal error", func(t *testing.T) {
		prevProvider := getConfigMgr
		defer func() { getConfigMgr = prevProvider }()

		getConfigMgr = func(ns string, sp api.StoreProvider) configMgr {
			return configmocks.NewConfigMgr()
		}

		r := shimtest.NewMockStub("mock_stub", cc.Chaincode()).MockInvoke(tx1, [][]byte{[]byte("save"), []byte("xxx")})
		require.NotNil(t, r)
		require.Equal(t, shim.ERROR, int(r.Status))
		require.Contains(t, r.Message, "Error unmarshalling config")
	})

	t.Run("Config manager error", func(t *testing.T) {
		prevProvider := getConfigMgr
		defer func() { getConfigMgr = prevProvider }()

		errExpected := errors.New("config mgr error")
		getConfigMgr = func(ns string, sp api.StoreProvider) configMgr {
			return configmocks.NewConfigMgr().WithError(errExpected)
		}

		r := shimtest.NewMockStub("mock_stub", cc.Chaincode()).MockInvoke(tx1, [][]byte{[]byte("save"), []byte(`{}`)})
		require.NotNil(t, r)
		require.Equal(t, shim.ERROR, int(r.Status))
		require.Contains(t, r.Message, errExpected.Error())
	})
}

func TestConfigSCC_Invoke_Get(t *testing.T) {
	cc := New()
	require.NotNil(t, cc)

	t.Run("No criteria", func(t *testing.T) {
		r := shimtest.NewMockStub("mock_stub", cc.Chaincode()).MockInvoke(tx1, [][]byte{[]byte("get")})
		require.NotNil(t, r)
		require.Equal(t, shim.ERROR, int(r.Status))
		require.Nil(t, r.Payload)
		require.Equal(t, "criteria not provided", r.Message)
	})

	t.Run("Invalid criteria", func(t *testing.T) {
		prevProvider := getConfigMgr
		defer func() { getConfigMgr = prevProvider }()

		getConfigMgr = func(ns string, sp api.StoreProvider) configMgr {
			return configmocks.NewConfigMgr()
		}
		r := shimtest.NewMockStub("mock_stub", cc.Chaincode()).MockInvoke(tx1, [][]byte{[]byte("get"), {}})
		require.NotNil(t, r)
		require.Equal(t, shim.ERROR, int(r.Status))
		require.Contains(t, r.Message, "error unmarshalling criteria")
	})

	t.Run("Valid args", func(t *testing.T) {
		prevProvider := getConfigMgr
		defer func() { getConfigMgr = prevProvider }()

		criteria := &config.Criteria{MspID: org1MSP}
		result := &config.KeyValue{
			Key:   &config.Key{},
			Value: config.NewValue(tx1, "config_value", config.FormatOther),
		}
		getConfigMgr = func(ns string, sp api.StoreProvider) configMgr {
			return configmocks.NewConfigMgr().WithQueryResults(criteria, []*config.KeyValue{result})
		}

		jsonKey, err := json.Marshal(criteria)
		require.NoError(t, err)

		r := shimtest.NewMockStub("mock_stub", cc.Chaincode()).MockInvoke(tx1, [][]byte{[]byte("get"), jsonKey})
		require.NotNil(t, r)
		require.Equal(t, shim.OK, int(r.Status))
		require.NotEmpty(t, r.Payload)

		var results []*config.KeyValue
		err = json.Unmarshal(r.Payload, &results)
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, result, results[0])
	})

	t.Run("Marshal error", func(t *testing.T) {
		prevProvider := getConfigMgr
		defer func() { getConfigMgr = prevProvider }()

		criteria := &config.Criteria{MspID: org1MSP}
		result := config.NewKeyValue(&config.Key{}, config.NewValue(tx1, "config_value", config.FormatOther))
		getConfigMgr = func(ns string, sp api.StoreProvider) configMgr {
			return configmocks.NewConfigMgr().WithQueryResults(criteria, []*config.KeyValue{result})
		}

		jsonKey, err := json.Marshal(criteria)
		require.NoError(t, err)

		errExpected := errors.New("marshal error")
		marshalJSON = func(v interface{}) (bytes []byte, e error) {
			return nil, errExpected
		}
		r := shimtest.NewMockStub("mock_stub", cc.Chaincode()).MockInvoke(tx1, [][]byte{[]byte("get"), jsonKey})
		require.NotNil(t, r)
		require.Equal(t, shim.ERROR, int(r.Status))
		require.Contains(t, r.Message, errExpected.Error())
	})

	t.Run("Query error", func(t *testing.T) {
		prevProvider := getConfigMgr
		defer func() { getConfigMgr = prevProvider }()

		errExpected := errors.New("config mgr error")
		getConfigMgr = func(ns string, sp api.StoreProvider) configMgr {
			return configmocks.NewConfigMgr().WithError(errExpected)
		}

		jsonKey, err := json.Marshal(&config.Key{MspID: org1MSP})
		require.NoError(t, err)

		r := shimtest.NewMockStub("mock_stub", cc.Chaincode()).MockInvoke(tx1, [][]byte{[]byte("get"), jsonKey})
		require.NotNil(t, r)
		require.Equal(t, shim.ERROR, int(r.Status))
		require.Contains(t, r.Message, errExpected.Error())
	})
}

func TestConfigSCC_Invoke_Delete(t *testing.T) {
	cc := New()
	require.NotNil(t, cc)

	t.Run("No criteria", func(t *testing.T) {
		r := shimtest.NewMockStub("mock_stub", cc.Chaincode()).MockInvoke(tx1, [][]byte{[]byte("delete")})
		require.NotNil(t, r)
		require.Equal(t, shim.ERROR, int(r.Status))
		require.Nil(t, r.Payload)
		require.Equal(t, "criteria not provided", r.Message)
	})

	t.Run("Invalid key", func(t *testing.T) {
		prevProvider := getConfigMgr
		defer func() { getConfigMgr = prevProvider }()

		getConfigMgr = func(ns string, sp api.StoreProvider) configMgr {
			return configmocks.NewConfigMgr()
		}
		r := shimtest.NewMockStub("mock_stub", cc.Chaincode()).MockInvoke(tx1, [][]byte{[]byte("delete"), {}})
		require.NotNil(t, r)
		require.Equal(t, shim.ERROR, int(r.Status))
		require.Contains(t, r.Message, "error unmarshalling criteria")
	})

	t.Run("Valid args", func(t *testing.T) {
		prevProvider := getConfigMgr
		defer func() { getConfigMgr = prevProvider }()

		getConfigMgr = func(ns string, sp api.StoreProvider) configMgr {
			return configmocks.NewConfigMgr()
		}

		jsonKey, err := json.Marshal(&config.Key{MspID: org1MSP})
		require.NoError(t, err)

		r := shimtest.NewMockStub("mock_stub", cc.Chaincode()).MockInvoke(tx1, [][]byte{[]byte("delete"), jsonKey})
		require.NotNil(t, r)
		require.Equal(t, shim.OK, int(r.Status))
		require.Empty(t, r.Payload)
	})

	t.Run("Query error", func(t *testing.T) {
		prevProvider := getConfigMgr
		defer func() { getConfigMgr = prevProvider }()

		errExpected := errors.New("config mgr error")
		getConfigMgr = func(ns string, sp api.StoreProvider) configMgr {
			return configmocks.NewConfigMgr().WithError(errExpected)
		}

		jsonKey, err := json.Marshal(&config.Key{MspID: org1MSP})
		require.NoError(t, err)

		r := shimtest.NewMockStub("mock_stub", cc.Chaincode()).MockInvoke(tx1, [][]byte{[]byte("delete"), jsonKey})
		require.NotNil(t, r)
		require.Equal(t, shim.ERROR, int(r.Status))
		require.Contains(t, r.Message, errExpected.Error())
	})
}
