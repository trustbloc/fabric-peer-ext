/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package cachestore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStateCacheCompositeKey(t *testing.T) {
	require.Equal(t, "ab", ConstructCompositeKey("a", "b"))

	require.Equal(t, "b", ConstructCompositeKey("", "b"))
}
