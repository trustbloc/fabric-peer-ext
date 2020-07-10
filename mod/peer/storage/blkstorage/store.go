/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blkstorage

import (
	"github.com/hyperledger/fabric/common/ledger/blkstorage"
	"github.com/hyperledger/fabric/common/metrics"
	"github.com/hyperledger/fabric/core/ledger"
	xledgerapi "github.com/hyperledger/fabric/extensions/ledger/api"

	"github.com/trustbloc/fabric-peer-ext/pkg/blkstorage/cdbblkstorage"
)

//NewProvider returns couchdb blockstorage provider
func NewProvider(_ *blkstorage.Conf, indexConfig *blkstorage.IndexConfig, ledgerconfig *ledger.Config, metricsProvider metrics.Provider) (xledgerapi.BlockStoreProvider, error) {
	return cdbblkstorage.NewProvider(indexConfig, ledgerconfig)
}

//NewConf is returns file system based blockstorage conf
func NewConf(blockStorageDir string, maxBlockfileSize int) *blkstorage.Conf {
	return blkstorage.NewConf(blockStorageDir, maxBlockfileSize)
}
