/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionSerialization(t *testing.T) {
	h1 := NewHeight(10, 100)
	b := h1.ToBytes()
	h2, n, err := NewHeightFromBytes(b)
	assert.NoError(t, err)
	assert.Equal(t, h1, h2)
	assert.Len(t, b, n)
}

func TestVersionComparison(t *testing.T) {
	assert.Equal(t, 1, NewHeight(10, 100).Compare(NewHeight(9, 1000)))
	assert.Equal(t, 1, NewHeight(10, 100).Compare(NewHeight(10, 90)))
	assert.Equal(t, -1, NewHeight(10, 100).Compare(NewHeight(11, 1)))
	assert.Equal(t, 0, NewHeight(10, 100).Compare(NewHeight(10, 100)))

	assert.True(t, AreSame(NewHeight(10, 100), NewHeight(10, 100)))
	assert.True(t, AreSame(nil, nil))
	assert.False(t, AreSame(NewHeight(10, 100), nil))
}

func TestVersionExtraBytes(t *testing.T) {
	extraBytes := []byte("junk")
	h1 := NewHeight(10, 100)
	b := h1.ToBytes()
	b1 := append(b, extraBytes...)
	h2, n, err := NewHeightFromBytes(b1)
	assert.NoError(t, err)
	assert.Equal(t, h1, h2)
	assert.Len(t, b, n)
	assert.Equal(t, extraBytes, b1[n:])
}
