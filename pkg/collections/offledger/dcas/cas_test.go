/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dcas

import (
	"errors"
	"testing"

	"github.com/btcsuite/btcutil/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCASKeyAndValue(t *testing.T) {
	t.Run("Binary value", func(t *testing.T) {
		v1 := []byte("value1")
		k64, v, err := GetCASKeyAndValue(v1)
		require.NoError(t, err)
		assert.NotNil(t, k64)
		assert.Equal(t, v1, v)

		k58, v, err := GetCASKeyAndValueBase58(v1)
		require.NoError(t, err)
		require.NotNil(t, k58)
		require.Equal(t, v1, v)
		require.Equal(t, k64, string(base58.Decode(k58)))
	})

	t.Run("JSON value", func(t *testing.T) {
		value1 := []byte(`{"field1":"value1","field2":"value2"}`)
		value2 := []byte(`{"field2":"value2","field1":"value1"}`)

		k1, v1, err := GetCASKeyAndValue(value1)
		require.NoError(t, err)
		assert.NotNil(t, k1)
		assert.NotNil(t, v1)

		k2, v2, err := GetCASKeyAndValue(value2)
		require.NoError(t, err)
		assert.Equal(t, k1, k2)
		assert.Equal(t, v1, v2)

		k1_58, v3, err := GetCASKeyAndValueBase58(value1)
		require.NoError(t, err)
		require.NotNil(t, k1_58)
		require.Equal(t, v1, v3)
		require.Equal(t, k1, string(base58.Decode(k1_58)))
	})

	t.Run("Marshal error", func(t *testing.T) {
		value1 := []byte(`{"field1":"value1","field2":"value2"}`)

		reset := SetJSONMarshaller(func(m map[string]interface{}) (bytes []byte, e error) {
			return nil, errors.New("injected marshal error")
		})
		defer reset()

		_, _, err := GetCASKeyAndValue(value1)
		require.Error(t, err)

		_, _, err = GetCASKeyAndValueBase58(value1)
		require.Error(t, err)
	})
}
