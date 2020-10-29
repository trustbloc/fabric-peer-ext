/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package peer

import (
	"fmt"

	cfg "github.com/trustbloc/fabric-peer-ext/pkg/config"
)

type dcasConfig struct {
	maxLinksPerBlock int
	rawLeaves        bool
	blockSize        int64
	layout           string
}

func newDCASConfig() *dcasConfig {
	cfg := &dcasConfig{
		maxLinksPerBlock: cfg.GetDCASMaxLinksPerBlock(),
		rawLeaves:        cfg.IsDCASRawLeaves(),
		blockSize:        cfg.GetDCASMaxBlockSize(),
		layout:           cfg.GetDCASBlockLayout(),
	}

	logger.Infof("Created DCAS config: %s", cfg)

	return cfg
}

func (c *dcasConfig) String() string {
	return fmt.Sprintf("blockSize: %d, maxLinksPerBlock: %d, rawLeaves: %t, blockLayout: %s", c.blockSize, c.maxLinksPerBlock, c.rawLeaves, c.layout)
}

// GetDCASMaxLinksPerBlock specifies the maximum number of links there will be per block in a Merkle DAG.
func (c *dcasConfig) GetDCASMaxLinksPerBlock() int {
	return c.maxLinksPerBlock
}

// IsDCASRawLeaves indicates whether or not to use raw leaf nodes in a Merkle DAG.
func (c *dcasConfig) IsDCASRawLeaves() bool {
	return c.rawLeaves
}

// GetDCASMaxBlockSize specifies the maximum size of a block in a Merkle DAG.
func (c *dcasConfig) GetDCASMaxBlockSize() int64 {
	return c.blockSize
}

// GetDCASBlockLayout returns the block layout strategy when creating a Merkle DAG.
// Supported values are "balanced" and "trickle". Leave empty to use the default.
func (c *dcasConfig) GetDCASBlockLayout() string {
	return c.layout
}
