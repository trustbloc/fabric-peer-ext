/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package compositekey

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	objectType = "type1"
	attr1      = "attr1"
	attr2      = "attr2"
)

func TestCreateAndSplit(t *testing.T) {
	attributes := []string{attr1, attr2}
	key := Create(objectType, attributes)
	require.NotEmpty(t, key)

	objType, attribs := Split(key)
	require.Equal(t, objectType, objType)
	require.Equal(t, attributes, attribs)
}

func TestCreateRangeKeysForPartialCompositeKey(t *testing.T) {
	startKey, endKey := CreateRangeKeysForPartialCompositeKey(objectType, []string{attr1, attr2})
	require.Equal(t, "\x00type1\x00attr1\x00attr2\x00", startKey)
	require.Equal(t, "\x00type1\x00attr1\x00attr2\x00\U0010ffff", endKey)
}
