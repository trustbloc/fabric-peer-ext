/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chaincode

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetUCC(t *testing.T) {
	cc, ok := GetUCC("ccid")
	require.False(t, ok)
	require.Nil(t, cc)
}

func TestChaincodes(t *testing.T) {
	require.Empty(t, Chaincodes())
}

func TestWaitForReady(t *testing.T) {
	require.NotPanics(t, WaitForReady)
}
