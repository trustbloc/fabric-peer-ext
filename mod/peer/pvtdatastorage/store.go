/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pvtdatastorage

import (
	"github.com/hyperledger/fabric/core/ledger/pvtdatastorage"
	s "github.com/trustbloc/fabric-peer-ext/pkg/pvtdatastorage"
)

// NewProvider instantiates a StoreProvider
func NewProvider() pvtdatastorage.Provider {
	return s.NewProvider()
}
