/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blkstorage

import (
	"github.com/hyperledger/fabric/common/ledger/blkstorage"
	"github.com/hyperledger/fabric/common/ledger/blkstorage/fsblkstorage"
	"github.com/hyperledger/fabric/common/metrics"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/trustbloc/fabric-peer-ext/pkg/blkstorage/cdbblkstorage"
)

//NewProvider returns couchdb blockstorage provider
func NewProvider(conf *fsblkstorage.Conf, indexConfig *blkstorage.IndexConfig, ledgerconfig *ledger.Config, metricsProvider metrics.Provider) (blkstorage.BlockStoreProvider, error) {
	return cdbblkstorage.NewProvider(indexConfig, ledgerconfig)
}

//NewConf is returns file system based blockstorage conf
func NewConf(blockStorageDir string, maxBlockfileSize int) *fsblkstorage.Conf {
	return fsblkstorage.NewConf(blockStorageDir, maxBlockfileSize)
}
