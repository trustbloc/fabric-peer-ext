/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pvtdatastorage

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/pvtdatastorage"
	xstorageapi "github.com/hyperledger/fabric/extensions/storage/api"

	"github.com/trustbloc/fabric-peer-ext/pkg/config"
	xpvtdatastorage "github.com/trustbloc/fabric-peer-ext/pkg/pvtdatastorage"
)

var logger = flogging.MustGetLogger("ext_storage")

// NewProvider instantiates a StoreProvider
func NewProvider(conf *pvtdatastorage.PrivateDataConfig, ledgerconfig *ledger.Config) (xstorageapi.PrivateDataProvider, error) {
	if config.GetPrivateDataStoreDBType() == config.CouchDBType {
		logger.Info("Using CouchDB private data storage provider")

		return xpvtdatastorage.NewProvider(conf, ledgerconfig)
	}

	logger.Info("Using default private data storage provider")

	return pvtdatastorage.NewProvider(conf)
}
