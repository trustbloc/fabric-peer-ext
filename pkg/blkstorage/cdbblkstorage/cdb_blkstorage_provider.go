/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cdbblkstorage

import (
	"strings"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/ledger/blkstorage"
	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/core/ledger"
	couchdb "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"
	"github.com/hyperledger/fabric/extensions/ledger/api"
	"github.com/pkg/errors"

	cfg "github.com/trustbloc/fabric-peer-ext/pkg/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

var logger = flogging.MustGetLogger("ext_blkstorage")

const (
	blockStoreName = "blocks"
	txnStoreName   = "transactions"
)

// CDBBlockstoreProvider provides block storage in CouchDB
type CDBBlockstoreProvider struct {
	options       []option
	couchInstance *couchdb.CouchInstance
	indexConfig   *blkstorage.IndexConfig
}

// NewProvider creates a new CouchDB BlockStoreProvider
func NewProvider(indexConfig *blkstorage.IndexConfig, ledgerconfig *ledger.Config) (api.BlockStoreProvider, error) {
	logger.Debugf("constructing CouchDB block storage provider")
	couchDBConfig := ledgerconfig.StateDBConfig.CouchDB
	couchInstance, err := couchdb.CreateCouchInstance(couchDBConfig, &disabled.Provider{})
	if err != nil {
		return nil, errors.WithMessage(err, "obtaining CouchDB instance failed")
	}

	return &CDBBlockstoreProvider{
		options: []option{
			withBlockByNumCacheSize(cfg.GetBlockStoreBlockByNumCacheSize()),
			withBlockByHashCacheSize(cfg.GetBlockStoreBlockByHashCacheSize()),
		},
		couchInstance: couchInstance,
		indexConfig:   indexConfig,
	}, nil
}

// Open opens the block store for the given ledger ID
func (p *CDBBlockstoreProvider) Open(ledgerid string) (api.BlockStore, error) {
	id := strings.ToLower(ledgerid)
	blockStoreDBName := couchdb.ConstructBlockchainDBName(id, blockStoreName)
	txnStoreDBName := couchdb.ConstructBlockchainDBName(id, txnStoreName)

	var err error
	var s api.BlockStore

	if roles.IsCommitter() {
		s, err = createCommitterBlockStore(p.couchInstance, ledgerid, blockStoreDBName, txnStoreDBName, p.options...)
	} else {
		s, err = p.createNonCommitterBlockStore(ledgerid, blockStoreDBName, txnStoreDBName, p.options...)
	}

	if err != nil {
		return nil, err
	}

	return s, nil
}

// createCommitterBlockStore creates new couch db with given db name if doesn't exists
func createCommitterBlockStore(couchInstance *couchdb.CouchInstance, ledgerid, blockStoreDBName, txnStoreDBName string, opts ...option) (api.BlockStore, error) {
	blockStoreDB, err := couchdb.CreateCouchDatabase(couchInstance, blockStoreDBName)
	if err != nil {
		return nil, err
	}

	txnStoreDB, err := couchdb.CreateCouchDatabase(couchInstance, txnStoreDBName)
	if err != nil {
		return nil, err
	}

	err = createBlockStoreIndices(blockStoreDB)
	if err != nil {
		return nil, err
	}

	return newCDBBlockStore(blockStoreDB, txnStoreDB, ledgerid, opts...), nil
}

// createNonCommitterBlockStore opens existing couch db with given db name with retry
func (p *CDBBlockstoreProvider) createNonCommitterBlockStore(ledgerid, blockStoreDBName, txnStoreDBName string, opts ...option) (api.BlockStore, error) {
	//create new block store db
	blockStoreDB, err := p.openCouchDB(blockStoreDBName)
	if err != nil {
		return nil, err
	}

	//check if indexes exists
	indexExists, err := blockStoreDB.IndexDesignDocExistsWithRetry(blockHashIndexDoc)
	if err != nil {
		return nil, err
	}
	if !indexExists {
		return nil, errors.Errorf("DB index not found: [%s]", blockStoreDBName)
	}

	//create new txn store db
	txnStoreDB, err := p.openCouchDB(txnStoreDBName)
	if err != nil {
		return nil, err
	}

	return newCDBBlockStore(blockStoreDB, txnStoreDB, ledgerid, opts...), nil
}

func (p *CDBBlockstoreProvider) openCouchDB(dbName string) (*couchdb.CouchDatabase, error) {
	//create new store db
	cdb, err := couchdb.NewCouchDatabase(p.couchInstance, dbName)
	if err != nil {
		return nil, err
	}

	//check if store db exists
	dbExists, err := cdb.ExistsWithRetry()
	if err != nil {
		return nil, err
	}
	if !dbExists {
		return nil, errors.Errorf("DB not found: [%s]", dbName)
	}

	return cdb, nil
}

func createBlockStoreIndices(db *couchdb.CouchDatabase) error {
	_, err := db.CreateIndex(blockHashIndexDef)
	if err != nil {
		return errors.WithMessage(err, "creation of block hash index failed")
	}

	return nil
}

// Close cleans up the Provider
func (p *CDBBlockstoreProvider) Close() {
}
