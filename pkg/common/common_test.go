/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type iface interface {
}

type test struct {
}

func Test_IsNil(t *testing.T) {
	assert.True(t, IsNil(nil))

	var b []byte
	assert.True(t, IsNil(b))

	var v iface
	assert.True(t, IsNil(v))
	v = &test{}
	assert.False(t, IsNil(v))

	v = nil
	v2 := v
	assert.True(t, IsNil(v2))

	assert.False(t, IsNil(test{}))
}

func TestValues_IsEmpty(t *testing.T) {
	v1 := []byte("v1")
	v2 := []byte("v1")
	v3 := []byte("v1")

	values := Values{v1, v2, v3}
	assert.False(t, values.IsEmpty())

	values = Values{v1, nil, v3}
	assert.False(t, values.IsEmpty())

	values = Values{nil, nil, nil}
	assert.True(t, values.IsEmpty())
}

func TestValues_AllSet(t *testing.T) {
	v1 := []byte("v1")
	v2 := []byte("v1")
	v3 := []byte("v1")

	values := Values{v1, v2, v3}
	assert.True(t, values.AllSet())

	values = Values{v1, nil, v3}
	assert.False(t, values.AllSet())
}

func TestValues_Merge(t *testing.T) {
	v1 := []byte("v1")
	v2 := []byte("v1")
	v3 := []byte("v1")

	t.Run("Merge same size", func(t *testing.T) {
		values1 := Values{v1, nil, v3}
		values2 := Values{v3, v2, v1}

		values := values1.Merge(values2)
		assert.True(t, values.AllSet())
		assert.Equal(t, v1, values[0])
		assert.Equal(t, v2, values[1])
		assert.Equal(t, v3, values[2])
	})

	t.Run("Merge source size bigger", func(t *testing.T) {
		values1 := Values{v1, nil, v3}
		values2 := Values{v3, v2}

		values := values1.Merge(values2)
		assert.True(t, values.AllSet())
		assert.Equal(t, v1, values[0])
		assert.Equal(t, v2, values[1])
		assert.Equal(t, v3, values[2])
	})

	t.Run("Merge dest size bigger", func(t *testing.T) {
		values1 := Values{v1, nil}
		values2 := Values{v3, v2, v3}

		values := values1.Merge(values2)
		assert.True(t, values.AllSet())
		assert.Equal(t, v1, values[0])
		assert.Equal(t, v2, values[1])
		assert.Equal(t, v3, values[2])
	})
}

func TestTimestamp(t *testing.T) {
	tim := time.Now()
	tim2 := FromTimestamp(ToTimestamp(tim))
	assert.Equal(t, tim.Unix(), tim2.Unix())
}
