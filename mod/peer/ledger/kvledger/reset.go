/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvledger

import (
	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/core/ledger"
	couchdb "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/blkstorage/cdbblkstorage"
)

// LoadPreResetHeight returns the prereset height for the specified ledgers.
func LoadPreResetHeight(ledgerconfig *ledger.Config, ledgerIDs []string) (map[string]uint64, error) {
	stateDBCouchInstance, err := couchdb.CreateCouchInstance(ledgerconfig.StateDBConfig.CouchDB, &disabled.Provider{})
	if err != nil {
		return nil, errors.WithMessage(err, "obtaining CouchDB instance failed")
	}

	return cdbblkstorage.LoadPreResetHeight(stateDBCouchInstance, ledgerIDs)
}
