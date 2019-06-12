/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package couchdb

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger/util/couchdb"
	"github.com/hyperledger/fabric/extensions/roles"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("extension-couchdb")

//CreateCouchDatabase is a handle function type for create couch db
type CreateCouchDatabase func(couchInstance *couchdb.CouchInstance, dbName string) (*couchdb.CouchDatabase, error)

//HandleCreateCouchDatabase can be used to extend create couch db feature
func HandleCreateCouchDatabase(handle CreateCouchDatabase) CreateCouchDatabase {
	if roles.IsCommitter() {
		return handle
	}
	logger.Debugf("Not a committer, getting couchdb instance of existing db")
	return createCouchDatabase
}

//createCouchDatabase returns db instance of existing db with given name
func createCouchDatabase(couchInstance *couchdb.CouchInstance, dbName string) (*couchdb.CouchDatabase, error) {
	db, err := couchdb.NewCouchDatabase(couchInstance, dbName)
	if err != nil {
		return nil, err
	}

	dbExists, err := db.ExistsWithRetry()
	if err != nil {
		return nil, err
	}

	if !dbExists {
		return nil, errors.Errorf("DB not found: [%s]", db.DBName)
	}

	return db, nil
}
