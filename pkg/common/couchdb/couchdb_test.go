/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package couchdb

import (
	"fmt"
	"testing"

	"github.com/hyperledger/fabric/common/metrics/disabled"
	couchdb "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"
	"github.com/stretchr/testify/require"
	xtestutil "github.com/trustbloc/fabric-peer-ext/pkg/testutil"
)

func TestConnectToCouchDB(t *testing.T) {
	_, _, destroy := xtestutil.SetupExtTestEnv()
	defer destroy()

	config := xtestutil.TestLedgerConf().StateDBConfig.CouchDB

	fmt.Printf("Config: %+v\n", config)

	cdbInstance, err := couchdb.CreateCouchInstance(config, &disabled.Provider{})
	require.NoError(t, err)

	p := NewReadOnlyProvider()
	require.NotNil(t, p)

	t.Run("Invalid DB name", func(t *testing.T) {
		db, err := p.ConnectToCouchDB(cdbInstance, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "unable to connect to couch DB")
		require.Nil(t, db)
	})

	t.Run("DB not found", func(t *testing.T) {
		db, err := p.ConnectToCouchDB(cdbInstance, "sample-1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "DB not found")
		require.Nil(t, db)
	})

	t.Run("DB exists", func(t *testing.T) {
		const dbName = "sample-2"
		cdb, err := couchdb.CreateCouchDatabase(cdbInstance, dbName)
		require.NoError(t, err)
		require.NotNil(t, cdb)

		db, err := p.ConnectToCouchDB(cdbInstance, dbName)
		require.NoError(t, err)
		require.NotNil(t, db)
	})
}
