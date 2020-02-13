/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
)

const (
	msp1 = "msp1"
	app1 = "app1"
	v1   = "v1"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	require.NotNil(t, r)

	val1 := &mockValidator{
		canValidate: false,
	}
	val2 := &mockValidator{
		canValidate: true,
	}

	r.Register(val1)

	k1 := config.NewAppKey(msp1, app1, v1)

	t.Run("Key-Value validation", func(t *testing.T) {
		k := &config.Key{}
		v1 := &config.Value{}
		v := r.ValidatorForKey(k)
		require.NotNil(t, v)
		require.EqualError(t, v.Validate(k, v1), "field [MspID] is required")

		v = r.ValidatorForKey(k1)
		require.NotNil(t, v)
		require.EqualError(t, v.Validate(k1, &config.Value{}), "field 'Format' must not be empty")
		require.EqualError(t, v.Validate(k1, &config.Value{Format: config.FormatJSON}), "field 'Config' must not be empty")
		require.NoError(t, v.Validate(k1, &config.Value{Format: config.FormatJSON, Config: "{}"}))
	})

	t.Run("Without app validator", func(t *testing.T) {
		v := r.ValidatorForKey(k1)
		require.NotNil(t, v)
		kv, ok := v.(*keyValueValidator)
		require.True(t, ok)
		require.Nil(t, kv.appValidator)
	})

	t.Run("With app validator", func(t *testing.T) {
		r.Register(val2)
		v := r.ValidatorForKey(k1)
		require.NotNil(t, v)
		kv, ok := v.(*keyValueValidator)
		require.True(t, ok)
		require.Equal(t, val2, kv.appValidator)
		require.NoError(t, v.Validate(k1, &config.Value{Format: config.FormatJSON, Config: "{}"}))
	})
}

func TestKeyValueValidator(t *testing.T) {
	v := &keyValueValidator{}

	k1 := config.NewAppKey(msp1, app1, v1)
	v1 := &config.Value{}

	require.True(t, v.CanValidate(k1))
	require.Error(t, v.Validate(k1, v1))
}

type mockValidator struct {
	canValidate bool
}

func (v *mockValidator) Validate(key *config.Key, value *config.Value) error {
	return nil
}

func (v *mockValidator) CanValidate(key *config.Key) bool {
	return v.canValidate
}
