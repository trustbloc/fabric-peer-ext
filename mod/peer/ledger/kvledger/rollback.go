/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvledger

import (
	"fmt"
)

func RollbackKVLedger(rootFSPath, ledgerID string, blockNum uint64) error {
	// TODO add RollbackKVLedger implementation for couchdb
	return fmt.Errorf("not supported")
}
