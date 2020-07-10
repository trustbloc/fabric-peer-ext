/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package couchdb

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/extensions/roles"
	storageapi "github.com/hyperledger/fabric/extensions/storage/api"
)

var logger = flogging.MustGetLogger("ext_couchdb")

var handler *Handler

// Handler is responsible for creating a CouchDB. If the peer has the committer role
// then it simply defers to the default handler, otherwise it creates a client that connects
// to an existing CouchDB.
type Handler struct {
	provider readOnlyCouchDBProvider
}

type readOnlyCouchDBProvider interface {
	ConnectToCouchDB(ci storageapi.CouchInstance, dbName string) (storageapi.CouchDatabase, error)
}

// NewHandler returns a new Handler
func NewHandler(couchDBProvider readOnlyCouchDBProvider) *Handler {
	logger.Info("Creating CouchDB handler")

	handler = &Handler{provider: couchDBProvider}

	return handler
}

// CreateCouchDatabase is a handle function type for create couch db
type CreateCouchDatabase func(couchInstance storageapi.CouchInstance, dbName string) (storageapi.CouchDatabase, error)

// HandleCreateCouchDatabase can be used to extend create couch db feature
func HandleCreateCouchDatabase(handle CreateCouchDatabase) CreateCouchDatabase {
	if roles.IsCommitter() {
		logger.Info("This peer is a committer. Returning default handler")
		return handle
	}

	logger.Infof("Not a committer, getting couchdb handler of existing db")

	return handler.provider.ConnectToCouchDB
}
