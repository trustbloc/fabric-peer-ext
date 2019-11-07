/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package resource

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitialize(t *testing.T) {
	require.NoError(t, Initialize(&someProvider{}))
}

func TestChannelJoined(t *testing.T) {
	require.NotPanics(t, func() { ChannelJoined("testchannel") })
}

func TestClose(t *testing.T) {
	require.NotPanics(t, func() { Close() })
}

type someProvider struct {
}
