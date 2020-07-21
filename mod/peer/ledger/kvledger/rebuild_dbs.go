/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvledger

import (
	"fmt"

	"github.com/hyperledger/fabric/core/ledger"
)

// RebuildDBs drops existing ledger databases.
// Dropped database will be rebuilt upon server restart
func RebuildDBs(config *ledger.Config) error {
	// https://github.com/trustbloc/fabric-peer-ext/issues/472
	return fmt.Errorf("not supported")
}
