/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockpublisher

import (
	extblockpublisher "github.com/trustbloc/fabric-peer-ext/pkg/gossip/blockpublisher"
)

// NewProvider returns a new block publisher provider
func NewProvider() *extblockpublisher.Provider {
	return extblockpublisher.NewProvider()
}
