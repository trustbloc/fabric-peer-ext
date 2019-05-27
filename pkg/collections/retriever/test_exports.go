// +build testing

/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package retriever

import (
	supportapi "github.com/hyperledger/fabric/extensions/collections/api/support"
	olapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/api"
	tdataapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/api"
)

// SetTransientDataProvider sets the transient data Retriever provider for unit tests
func SetTransientDataProvider(provider func(storeProvider func(channelID string) tdataapi.Store, support Support, gossipProvider func() supportapi.GossipAdapter) tdataapi.Provider) {
	getTransientDataProvider = provider
}

// SetOffLedgerProvider sets the off-ledger Retriever provider for unit tests
func SetOffLedgerProvider(provider func(storeProvider func(channelID string) olapi.Store, support Support, gossipProvider func() supportapi.GossipAdapter) olapi.Provider) {
	getOffLedgerProvider = provider
}
