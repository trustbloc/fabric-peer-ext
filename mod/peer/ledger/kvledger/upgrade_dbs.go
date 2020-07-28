/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvledger

import (
	"github.com/pkg/errors"

	"github.com/hyperledger/fabric/common/ledger/util/leveldbhelper"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger"
	"github.com/trustbloc/fabric-peer-ext/pkg/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/idstore"
)

// UpgradeDBs upgrades existing ledger databases to the latest formats.
// It checks the format of idStore and does not drop any databases
// if the format is already the latest version. Otherwise, it drops
// ledger databases and upgrades the idStore format.
func UpgradeDBs(ledgerconfig *ledger.Config) error {
	if config.GetBlockStoreDBType() == config.CouchDBType {
		rootFSPath := ledgerconfig.RootFSPath
		fileLockPath := fileLockPath(rootFSPath)
		fileLock := leveldbhelper.NewFileLock(fileLockPath)
		if err := fileLock.Lock(); err != nil {
			return errors.Wrap(err, "as another peer node command is executing,"+
				" wait for that command to complete its execution or terminate it before retrying")
		}
		defer fileLock.Unlock()

		if err := dropDBs(ledgerconfig); err != nil {
			return err
		}

		store, err := idstore.OpenIDStore(ledgerconfig)
		if err != nil {
			return err
		}

		return store.UpgradeFormat()
	}
	return kvledger.UpgradeDBs(ledgerconfig)
}
