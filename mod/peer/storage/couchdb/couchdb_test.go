/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package couchdb

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"

	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/core/ledger/util/couchdb"
	xtestutil "github.com/trustbloc/fabric-peer-ext/pkg/testutil"
)

var cdbInstance *couchdb.CouchInstance

func TestMain(m *testing.M) {

	//setup extension test environment
	_, _, destroy := xtestutil.SetupExtTestEnv()

	couchDBConfig := xtestutil.TestLedgerConf().StateDB.CouchDB
	var err error
	cdbInstance, err = couchdb.CreateCouchInstance(couchDBConfig, &disabled.Provider{})
	if err != nil {
		panic(err.Error())
	}

	code := m.Run()

	destroy()

	os.Exit(code)
}

func TestCreateCouchDatabaseHandleByCommitter(t *testing.T) {

	sampleDB := &couchdb.CouchDatabase{DBName: "sample-test-run-db"}
	handle := func(couchInstance *couchdb.CouchInstance, dbName string) (*couchdb.CouchDatabase, error) {
		return sampleDB, nil
	}

	//make sure roles is committer not endorser
	if roles.IsEndorser() {
		rolesValue := make(map[roles.Role]struct{})
		rolesValue[roles.CommitterRole] = struct{}{}
		roles.SetRoles(rolesValue)
		defer func() { roles.SetRoles(nil) }()
	}
	require.True(t, roles.IsCommitter())
	require.False(t, roles.IsEndorser())

	db, err := HandleCreateCouchDatabase(handle)(nil, "")
	require.Equal(t, sampleDB, db)
	require.NoError(t, err)

}

func TestCreateCouchDatabaseByCommitter(t *testing.T) {

	const dbName = "sampledb-test-c"

	//make sure roles is committer not endorser
	if roles.IsEndorser() {
		rolesValue := make(map[roles.Role]struct{})
		rolesValue[roles.CommitterRole] = struct{}{}
		roles.SetRoles(rolesValue)
		defer func() { roles.SetRoles(nil) }()
	}
	require.True(t, roles.IsCommitter())
	require.False(t, roles.IsEndorser())

	db, err := HandleCreateCouchDatabase(couchdb.CreateCouchDatabase)(cdbInstance, dbName)
	require.NoError(t, err)
	require.NotNil(t, db)
	require.Equal(t, dbName, db.DBName)

}

func TestCreateCouchDatabaseByEndorser(t *testing.T) {

	const dbName = "sampledb-test-e"

	//make sure roles is committer not endorser
	if roles.IsCommitter() {
		rolesValue := make(map[roles.Role]struct{})
		rolesValue[roles.EndorserRole] = struct{}{}
		roles.SetRoles(rolesValue)
		defer func() { roles.SetRoles(nil) }()
	}
	require.False(t, roles.IsCommitter())
	require.True(t, roles.IsEndorser())

	//for non committer, db is returned only if it already exists
	db, err := HandleCreateCouchDatabase(couchdb.CreateCouchDatabase)(cdbInstance, dbName)
	require.Error(t, err)
	require.Nil(t, db)
	require.Contains(t, err.Error(), "DB not found")

	//create db manually
	db, err = couchdb.CreateCouchDatabase(cdbInstance, dbName)
	require.NoError(t, err)
	require.NotNil(t, db)

	//now try again when couchdb with given db name already exists
	edb, err := HandleCreateCouchDatabase(couchdb.CreateCouchDatabase)(cdbInstance, dbName)
	require.NoError(t, err)
	require.NotNil(t, db)
	require.Equal(t, db, edb)

}
