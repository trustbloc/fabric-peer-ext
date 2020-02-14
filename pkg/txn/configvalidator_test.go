/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package txn

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
)

func TestConfigValidator_Validate(t *testing.T) {
	v := newConfigValidator()

	txnKeyV1 := config.NewPeerComponentKey(msp1, peer1, configApp, configVersion, generalConfigComponent, generalConfigVersion)
	txnKeyV2 := config.NewPeerComponentKey(msp1, peer1, configApp, configVersion, generalConfigComponent, "v2")
	sdkKeyV1 := config.NewPeerComponentKey(msp1, peer1, configApp, configVersion, sdkConfigComponent, sdkConfigVersion)
	sdkKeyV2 := config.NewPeerComponentKey(msp1, peer1, configApp, configVersion, sdkConfigComponent, "v2")
	invalidCompKey := config.NewPeerComponentKey(msp1, peer1, configApp, configVersion, "some-component", "v1")

	t.Run("Not relevant -> success", func(t *testing.T) {
		kv := config.NewKeyValue(config.NewAppKey(msp1, "some-app", "v1"), &config.Value{Format: "json", Config: `{}`})
		require.NoError(t, v.Validate(kv))
	})

	t.Run("No PeerID -> error", func(t *testing.T) {
		kv := config.NewKeyValue(config.NewComponentKey(msp1, configApp, configVersion, generalConfigComponent, generalConfigVersion), &config.Value{Format: "yaml", Config: `{}`})
		err := v.Validate(kv)
		require.Error(t, err)
		require.Contains(t, err.Error(), "expecting PeerID")
	})

	t.Run("Unexpected component -> error", func(t *testing.T) {
		err := v.Validate(config.NewKeyValue(invalidCompKey, &config.Value{Format: "yaml", Config: `{}`}))
		require.Error(t, err)
		require.Contains(t, err.Error(), "unexpected component [some-component]")
	})

	t.Run("Txn config", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			require.NoError(t, v.Validate(config.NewKeyValue(txnKeyV1, &config.Value{Format: "json", Config: `{"User":"User1"}`})))
		})

		t.Run("Unsupported version", func(t *testing.T) {
			err := v.Validate(config.NewKeyValue(txnKeyV2, &config.Value{Format: "json", Config: `{}`}))
			require.Error(t, err)
			require.Contains(t, err.Error(), "unsupported component version")
		})

		t.Run("Invalid format", func(t *testing.T) {
			err := v.Validate(config.NewKeyValue(txnKeyV1, &config.Value{Format: "yaml", Config: `{}`}))
			require.Error(t, err)
			require.Contains(t, err.Error(), "expecting format [JSON] but got [yaml]")
		})

		t.Run("Invalid JSON", func(t *testing.T) {
			err := v.Validate(config.NewKeyValue(txnKeyV1, &config.Value{Format: "json", Config: `{xxx}`}))
			require.Error(t, err)
			require.Contains(t, err.Error(), "error unmarshalling value for TXN config")
		})

		t.Run("No UserID", func(t *testing.T) {
			err := v.Validate(config.NewKeyValue(txnKeyV1, &config.Value{Format: "json", Config: `{}`}))
			require.Error(t, err)
			require.Contains(t, err.Error(), "field 'UserID' was not specified for TXN config")
		})
	})

	t.Run("SDK config", func(t *testing.T) {
		sdkCfgBytes, err := ioutil.ReadFile("./client/testdata/sdk-config.yaml")
		require.NoError(t, err)

		sdkCfgValue := &config.Value{
			Format: "yaml",
			Config: string(sdkCfgBytes),
		}

		t.Run("Valid", func(t *testing.T) {
			require.NoError(t, v.Validate(config.NewKeyValue(sdkKeyV1, sdkCfgValue)))
		})

		t.Run("Unsupported version", func(t *testing.T) {
			err := v.Validate(config.NewKeyValue(sdkKeyV2, sdkCfgValue))
			require.Error(t, err)
			require.Contains(t, err.Error(), "unsupported component version")
		})

		t.Run("Invalid config", func(t *testing.T) {
			err := v.Validate(config.NewKeyValue(sdkKeyV1, &config.Value{Format: "json", Config: `{xxx}`}))
			require.Error(t, err)
			require.Contains(t, err.Error(), "invalid SDK config")
		})
	})
}
