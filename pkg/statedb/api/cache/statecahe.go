/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package statecahe

import (
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
)

// StateCache state cache interface
type StateCache interface {
	// Get gets state data
	Get(ns, key string) *statedb.VersionedValue

	// Add adds state data
	Add(ns, key string, vval *statedb.VersionedValue)

	// Remove removes state data
	Remove(ns string, key string) bool

	// GetVersion version of state data
	GetVersion(ns, key string) *version.Height
}
