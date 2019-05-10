/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transientstore

import (
	"github.com/hyperledger/fabric/core/transientstore"
	ts "github.com/trustbloc/fabric-peer-ext/pkg/transientstore"
)

// NewStoreProvider return new store provider
func NewStoreProvider() transientstore.StoreProvider {
	return ts.NewStoreProvider()
}
