/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idstore

import (
	"github.com/hyperledger/fabric/core/ledger"
	xstorageapi "github.com/hyperledger/fabric/extensions/storage/api"
	xidstore "github.com/trustbloc/fabric-peer-ext/pkg/idstore"
)

type OpenIDStoreHandler func(path string, ledgerconfig *ledger.Config) (xstorageapi.IDStore, error)

// OpenIDStore open idstore
func OpenIDStore(path string, ledgerconfig *ledger.Config, _ OpenIDStoreHandler) (xstorageapi.IDStore, error) {
	return xidstore.OpenIDStore(ledgerconfig)
}
