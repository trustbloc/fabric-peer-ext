/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pvtdatastorage

import (
	"github.com/hyperledger/fabric/core/ledger/pvtdatastorage"
	cdbpvtdatastore "github.com/trustbloc/fabric-peer-ext/pkg/pvtdatastorage/cdbpvtdatastore"
)

// NewProvider instantiates a StoreProvider
func NewProvider() pvtdatastorage.Provider {
	provider, err := cdbpvtdatastore.NewProvider()
	if err != nil {
		panic(err)
	}
	return provider
}
