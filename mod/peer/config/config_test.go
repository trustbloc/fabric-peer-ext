/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"testing"

	viper "github.com/spf13/viper2015"
	"github.com/stretchr/testify/require"
)

func TestIsSkipCheckForDupTxnID(t *testing.T) {
	require.False(t, IsSkipCheckForDupTxnID())

	viper.Set("peer.skipCheckForDupTxnID", true)
	require.True(t, IsSkipCheckForDupTxnID())
}
