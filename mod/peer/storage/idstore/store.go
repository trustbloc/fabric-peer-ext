/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idstore

import (
	"github.com/hyperledger/fabric/core/ledger"
	storeapi "github.com/hyperledger/fabric/extensions/storage/api"
	s "github.com/trustbloc/fabric-peer-ext/pkg/idstore"
)

// OpenIDStore open idstore
func OpenIDStore(path string, ledgerconfig *ledger.Config) (storeapi.IDStore, error) {
	store, err := s.OpenIDStore(ledgerconfig)
	if err != nil {
		return nil, err
	}
	return store, nil
}
