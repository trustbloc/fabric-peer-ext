/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package couchdb

import (
	"github.com/hyperledger/fabric/common/flogging"
	couchdb "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"
	storageapi "github.com/hyperledger/fabric/extensions/storage/api"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("ext_couchdb")

// ReadOnlyProvider is a read-only CouchDB provider
type ReadOnlyProvider struct {
}

// NewReadOnlyProvider returns a read-only CouchDB provider
func NewReadOnlyProvider() *ReadOnlyProvider {
	logger.Infof("Creating couch database provider")

	return &ReadOnlyProvider{}
}

// ConnectToCouchDB connects to an existing Couch database
func (p *ReadOnlyProvider) ConnectToCouchDB(ci storageapi.CouchInstance, dbName string) (storageapi.CouchDatabase, error) {
	logger.Infof("Connecting to couch database [%s]", dbName)

	db, err := couchdb.NewCouchDatabase(ci.(*couchdb.CouchInstance), dbName)
	if err != nil {
		return nil, errors.WithMessagef(err, "unable to connect to couch DB [%s]", dbName)
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
