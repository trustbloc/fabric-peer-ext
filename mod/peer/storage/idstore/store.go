/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idstore

import (
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/idstore"
	s "github.com/trustbloc/fabric-peer-ext/pkg/idstore"
)

// OpenIDStore open idstore
func OpenIDStore(path string, ledgerconfig *ledger.Config) idstore.IDStore {
	store, err := s.OpenIDStore(ledgerconfig)
	if err != nil {
		panic(err.Error())
	}
	return store
}
