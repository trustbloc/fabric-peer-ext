/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ledgerstorage

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/extensions/roles"
)

var logger = flogging.MustGetLogger("extension-recover")

//SyncPvtdataStoreWithBlockStoreHandler provides extension for syncing private data store with block store
func SyncPvtdataStoreWithBlockStoreHandler(handle func() error) func() error {

	if roles.IsCommitter() {
		return handle
	}

	return func() error {
		logger.Debug("Not a committer, so no need to sync private data store with block store")
		return nil
	}
}
