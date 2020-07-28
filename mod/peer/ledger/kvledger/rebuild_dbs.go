/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvledger

import (
	"github.com/hyperledger/fabric/common/ledger/util/leveldbhelper"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/config"
)

// RebuildDBs drops existing ledger databases.
// Dropped database will be rebuilt upon server restart
func RebuildDBs(ledgerconfig *ledger.Config) error {
	if config.GetBlockStoreDBType() == config.CouchDBType {
		rootFSPath := ledgerconfig.RootFSPath
		fileLockPath := fileLockPath(rootFSPath)
		fileLock := leveldbhelper.NewFileLock(fileLockPath)
		if err := fileLock.Lock(); err != nil {
			return errors.Wrap(err, "as another peer node command is executing,"+
				" wait for that command to complete its execution or terminate it before retrying")
		}
		defer fileLock.Unlock()

		return dropDBs(ledgerconfig)
	}

	return kvledger.RebuildDBs(ledgerconfig)
}
