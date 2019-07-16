/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockpublisher

import (
	"sync"

	extblockpublisher "github.com/trustbloc/fabric-peer-ext/pkg/gossip/blockpublisher"
)

var blockPublisherProvider *extblockpublisher.Provider
var once sync.Once

// GetProvider returns block publisher provider
func GetProvider() *extblockpublisher.Provider {
	once.Do(func() {
		blockPublisherProvider = extblockpublisher.NewProvider()
	})
	return blockPublisherProvider
}
