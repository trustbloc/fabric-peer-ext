/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package peer

import (
	"testing"

	viper "github.com/spf13/viper2015"
	"github.com/stretchr/testify/require"
)

func TestDCASConfigConfig(t *testing.T) {
	const (
		maxLinksPerBlock = 33
		rawLeaves        = false
		maxBlockSize     = int64(44)
		blockLayout      = "trickle"
	)

	viper.Set("coll.dcas.maxLinksPerBlock", maxLinksPerBlock)
	viper.Set("coll.dcas.rawLeaves", rawLeaves)
	viper.Set("coll.dcas.maxBlockSize", maxBlockSize)
	viper.Set("coll.dcas.blockLayout", blockLayout)

	c := newDCASConfig()
	require.Equal(t, maxLinksPerBlock, c.GetDCASMaxLinksPerBlock())
	require.Equal(t, rawLeaves, c.IsDCASRawLeaves())
	require.Equal(t, maxBlockSize, c.GetDCASMaxBlockSize())
	require.Equal(t, blockLayout, c.GetDCASBlockLayout())
}
