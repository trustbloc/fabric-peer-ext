/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transientstore

import "github.com/hyperledger/fabric/core/transientstore"

// NewStoreProvider return new store provider
func NewStoreProvider() transientstore.StoreProvider {
	return transientstore.NewStoreProvider()
}
