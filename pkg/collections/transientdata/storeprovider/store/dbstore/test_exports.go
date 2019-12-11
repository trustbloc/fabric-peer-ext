// +build testing

/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dbstore

import (
	"github.com/hyperledger/fabric/common/ledger/util/leveldbhelper"
)

// LevelDBCreator is a Level DB creator function
type LevelDBCreator func(dbPath string) (*leveldbhelper.Provider, error)

// SetLevelDBCreator sets the Level DB creator function and returns a function that will restore the
// creator to its original value.
func SetLevelDBCreator(creator LevelDBCreator) func() {
	previousCreator := newLevelDBProvider
	newLevelDBProvider = creator
	return func() { newLevelDBProvider = previousCreator }
}
