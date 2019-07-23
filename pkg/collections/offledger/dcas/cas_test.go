/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dcas

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCASKeyAndValue(t *testing.T) {
	t.Run("Binary value", func(t *testing.T) {
		k, v, err := GetCASKeyAndValue([]byte("value1"))
		require.NoError(t, err)
		assert.NotNil(t, k)
		assert.NotNil(t, v)
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
	})

	t.Run("Marshal error", func(t *testing.T) {
		value1 := []byte(`{"field1":"value1","field2":"value2"}`)

		reset := SetJSONMarshaller(func(m map[string]interface{}) (bytes []byte, e error) {
			return nil, errors.New("injected marshal error")
		})
		defer reset()

		_, _, err := GetCASKeyAndValue(value1)
		require.Error(t, err)
	})
}
