/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idstore

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger"
	xstorageapi "github.com/hyperledger/fabric/extensions/storage/api"

	"github.com/trustbloc/fabric-peer-ext/pkg/config"
	xidstore "github.com/trustbloc/fabric-peer-ext/pkg/idstore"
)

var logger = flogging.MustGetLogger("ext_storage")

type OpenIDStoreHandler func(path string, ledgerconfig *ledger.Config) (xstorageapi.IDStore, error)

// OpenIDStore open idstore
func OpenIDStore(path string, ledgerconfig *ledger.Config, openDefaultStore OpenIDStoreHandler) (xstorageapi.IDStore, error) {
	if config.GetIDStoreDBType() == config.CouchDBType {
		logger.Info("Opening CouchDB ID store")

		return xidstore.OpenIDStore(ledgerconfig)
	}

	logger.Info("Opening default ID store")

	return openDefaultStore(path, ledgerconfig)
}
