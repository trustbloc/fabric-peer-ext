/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blkstorage

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/ledger/blkstorage"
	"github.com/hyperledger/fabric/common/metrics"
	"github.com/hyperledger/fabric/core/ledger"
	xledgerapi "github.com/hyperledger/fabric/extensions/ledger/api"

	"github.com/trustbloc/fabric-peer-ext/pkg/blkstorage/cdbblkstorage"
	"github.com/trustbloc/fabric-peer-ext/pkg/config"
)

var logger = flogging.MustGetLogger("ext_storage")

//NewProvider returns couchdb blockstorage provider
func NewProvider(conf *blkstorage.Conf, indexConfig *blkstorage.IndexConfig, ledgerconfig *ledger.Config, metricsProvider metrics.Provider) (xledgerapi.BlockStoreProvider, error) {
	if config.GetBlockStoreDBType() == config.CouchDBType {
		logger.Info("Using CouchDB block storage provider")

		return cdbblkstorage.NewProvider(indexConfig, ledgerconfig)
	}

	logger.Info("Using default block storage provider")

	return blkstorage.NewProvider(conf, indexConfig, metricsProvider)
}

//NewConf is returns file system based blockstorage conf
func NewConf(blockStorageDir string, maxBlockfileSize int) *blkstorage.Conf {
	return blkstorage.NewConf(blockStorageDir, maxBlockfileSize)
}
