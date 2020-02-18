/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/mocks"
)

const (
	msp1 = "msp1"
	app1 = "app1"
	v1   = "v1"
)

func TestKeyValueValidation(t *testing.T) {
	k1 := config.NewAppKey(msp1, app1, v1)

	r := NewRegistry()
	require.NotNil(t, r)

	t.Run("No MspID", func(t *testing.T) {
		kv1 := config.NewKeyValue(&config.Key{}, &config.Value{})
		require.EqualError(t, r.Validate(kv1), "field [MspID] is required")
	})

	t.Run("No Format", func(t *testing.T) {
		kv := config.NewKeyValue(k1, &config.Value{})
		require.EqualError(t, r.Validate(kv), "field 'Format' must not be empty")
	})

	t.Run("No Config", func(t *testing.T) {
		kv := config.NewKeyValue(k1, &config.Value{Format: config.FormatJSON})
		require.EqualError(t, r.Validate(kv), "field 'Config' must not be empty")
	})

	t.Run("Valid Config", func(t *testing.T) {
		kv := config.NewKeyValue(k1, &config.Value{Format: config.FormatJSON, Config: "{}"})
		require.NoError(t, r.Validate(kv))
	})
}

func TestAppValidation(t *testing.T) {
	r := NewRegistry()
	require.NotNil(t, r)

	val1 := &mocks.Validator{}
	errExpected := errors.New("injected validation error")
	val1.ValidateReturns(errExpected)
	r.Register(val1)

	kv := config.NewKeyValue(config.NewAppKey(msp1, app1, v1), &config.Value{Format: config.FormatJSON, Config: "{}"})
	require.EqualError(t, r.Validate(kv), errExpected.Error())
}
