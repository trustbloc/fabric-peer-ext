/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package storeprovider

import (
	extstoreprovider "github.com/trustbloc/fabric-peer-ext/pkg/collections/storeprovider"
)

// NewProviderFactory returns a new store provider factory
func NewProviderFactory() *extstoreprovider.StoreProvider {
	return extstoreprovider.New()
}
