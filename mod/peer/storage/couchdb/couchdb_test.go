/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package couchdb

import (
	"testing"

	storageapi "github.com/hyperledger/fabric/extensions/storage/api"
	"github.com/hyperledger/fabric/extensions/storage/couchdb/mocks"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

//go:generate counterfeiter -o ./mocks/couchdbprovider.gen.go --fake-name CouchDBProvider . couchDBProvider
//go:generate counterfeiter -o ./mocks/couchdatabase.gen.go --fake-name CouchDatabase ../api CouchDatabase

var _ = roles.GetRoles()

func TestHandler(t *testing.T) {
	defaultDB := &mocks.CouchDatabase{}
	extDB := &mocks.CouchDatabase{}

	cdbProvider := &mocks.CouchDBProvider{}
	cdbProvider.CreateCouchDBReturns(extDB, nil)

	h := NewHandler(cdbProvider)
	require.NotNil(t, h)
	require.NotNil(t, handler)

	defaultHandler := func(couchInstance storageapi.CouchInstance, dbName string) (storageapi.CouchDatabase, error) {
		return defaultDB, nil
	}

	t.Run("As committer", func(t *testing.T) {
		createDB := HandleCreateCouchDatabase(defaultHandler)
		require.NotNil(t, createDB)
		db, err := createDB(nil, "")
		require.NoError(t, err)
		require.True(t, db == defaultDB)
	})

	t.Run("As non-committer", func(t *testing.T) {
		roles.SetRoles(map[roles.Role]struct{}{"endorser": {}})
		defer roles.SetRoles(nil)

		createDB := HandleCreateCouchDatabase(defaultHandler)
		require.NotNil(t, createDB)
		db, err := createDB(nil, "")
		require.NoError(t, err)
		require.True(t, db == extDB)
	})
}
