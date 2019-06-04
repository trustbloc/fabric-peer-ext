/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package storeprovider

import (
	"github.com/hyperledger/fabric/core/ledger"
	extstoreprovider "github.com/trustbloc/fabric-peer-ext/pkg/collections/storeprovider"
)

// NewProviderFactory returns a new store provider factory
func NewProviderFactory(ledgerconfig *ledger.Config) *extstoreprovider.StoreProvider {
	return extstoreprovider.New(ledgerconfig)
}
