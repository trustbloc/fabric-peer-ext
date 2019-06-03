/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cdbblkstorage

import (
	"strings"

	"github.com/hyperledger/fabric/core/ledger"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/ledger/blkstorage"
	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/core/ledger/util/couchdb"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

var logger = flogging.MustGetLogger("cdbblkstorage")

const (
	blockStoreName = "blocks"
	txnStoreName   = "transactions"
)

// CDBBlockstoreProvider provides block storage in CouchDB
type CDBBlockstoreProvider struct {
	couchInstance *couchdb.CouchInstance
	indexConfig   *blkstorage.IndexConfig
}

// NewProvider creates a new CouchDB BlockStoreProvider
func NewProvider(indexConfig *blkstorage.IndexConfig, ledgerconfig *ledger.Config) (blkstorage.BlockStoreProvider, error) {
	logger.Debugf("constructing CouchDB block storage provider")
	couchDBConfig := ledgerconfig.StateDB.CouchDB
	couchInstance, err := couchdb.CreateCouchInstance(couchDBConfig, &disabled.Provider{})
	if err != nil {
		return nil, errors.WithMessage(err, "obtaining CouchDB instance failed")
	}
	return &CDBBlockstoreProvider{couchInstance, indexConfig}, nil
}

// CreateBlockStore creates a block store instance for the given ledger ID
func (p *CDBBlockstoreProvider) CreateBlockStore(ledgerid string) (blkstorage.BlockStore, error) {
	return p.OpenBlockStore(ledgerid)
}

// OpenBlockStore opens the block store for the given ledger ID
func (p *CDBBlockstoreProvider) OpenBlockStore(ledgerid string) (blkstorage.BlockStore, error) {
	id := strings.ToLower(ledgerid)
	blockStoreDBName := couchdb.ConstructBlockchainDBName(id, blockStoreName)
	txnStoreDBName := couchdb.ConstructBlockchainDBName(id, txnStoreName)

	if roles.IsCommitter() {
		return p.createCommitterBlockStore(ledgerid, blockStoreDBName, txnStoreDBName)
	}

	return p.createNonCommitterBlockStore(ledgerid, blockStoreDBName, txnStoreDBName)
}

//createCommitterBlockStore creates new couch db with gievn db name if doesn't exists
func (p *CDBBlockstoreProvider) createCommitterBlockStore(ledgerid, blockStoreDBName, txnStoreDBName string) (blkstorage.BlockStore, error) {

	blockStoreDB, err := couchdb.CreateCouchDatabase(p.couchInstance, blockStoreDBName)
	if err != nil {
		return nil, err
	}

	txnStoreDB, err := couchdb.CreateCouchDatabase(p.couchInstance, txnStoreDBName)
	if err != nil {
		return nil, err
	}

	err = p.createBlockStoreIndices(blockStoreDB)
	if err != nil {
		return nil, err
	}

	return newCDBBlockStore(blockStoreDB, txnStoreDB, ledgerid), nil
}

//createBlockStore opens existing couch db with given db name with retry
func (p *CDBBlockstoreProvider) createNonCommitterBlockStore(ledgerid, blockStoreDBName, txnStoreDBName string) (blkstorage.BlockStore, error) {

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

	return newCDBBlockStore(blockStoreDB, txnStoreDB, ledgerid), nil
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

func (p *CDBBlockstoreProvider) createBlockStoreIndices(db *couchdb.CouchDatabase) error {
	_, err := db.CreateIndex(blockHashIndexDef)
	if err != nil {
		return errors.WithMessage(err, "creation of block hash index failed")
	}

	return nil
}

// Exists returns whether or not the given ledger ID exists
func (p *CDBBlockstoreProvider) Exists(ledgerid string) (bool, error) {
	id := strings.ToLower(ledgerid)
	blockStoreDBName := couchdb.ConstructBlockchainDBName(id, blockStoreName)
	blockStoreDB, err := couchdb.NewCouchDatabase(p.couchInstance, blockStoreDBName)
	if err != nil {
		return false, err
	}

	return blockStoreDB.Exists()
}

// List returns the available ledger IDs, not supported in couchdb block storage
func (p *CDBBlockstoreProvider) List() ([]string, error) {
	panic("not supported")
}

// Close cleans up the Provider
func (p *CDBBlockstoreProvider) Close() {
}
