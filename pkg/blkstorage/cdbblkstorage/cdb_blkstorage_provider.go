/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cdbblkstorage

import (
	"strings"

	"github.com/hyperledger/fabric/common/ledger/snapshot"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/extensions/ledger/api"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/ledger/blkstorage"
	"github.com/hyperledger/fabric/common/metrics/disabled"
	couchdb "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"
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
func NewProvider(indexConfig *blkstorage.IndexConfig, ledgerconfig *ledger.Config) (api.BlockStoreProvider, error) {
	logger.Debugf("constructing CouchDB block storage provider")
	couchDBConfig := ledgerconfig.StateDBConfig.CouchDB
	couchInstance, err := couchdb.CreateCouchInstance(couchDBConfig, &disabled.Provider{})
	if err != nil {
		return nil, errors.WithMessage(err, "obtaining CouchDB instance failed")
	}
	return &CDBBlockstoreProvider{couchInstance, indexConfig}, nil
}

// Open opens the block store for the given ledger ID
func (p *CDBBlockstoreProvider) Open(ledgerid string) (api.BlockStore, error) {
	id := strings.ToLower(ledgerid)
	blockStoreDBName := couchdb.ConstructBlockchainDBName(id, blockStoreName)
	txnStoreDBName := couchdb.ConstructBlockchainDBName(id, txnStoreName)

	if roles.IsCommitter() {
		return p.createCommitterBlockStore(ledgerid, blockStoreDBName, txnStoreDBName)
	}

	return p.createNonCommitterBlockStore(ledgerid, blockStoreDBName, txnStoreDBName)
}

//createCommitterBlockStore creates new couch db with gievn db name if doesn't exists
func (p *CDBBlockstoreProvider) createCommitterBlockStore(ledgerid, blockStoreDBName, txnStoreDBName string) (api.BlockStore, error) {

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
func (p *CDBBlockstoreProvider) createNonCommitterBlockStore(ledgerid, blockStoreDBName, txnStoreDBName string) (api.BlockStore, error) {

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

// BootstrapFromSnapshottedTxIDs initializes blockstore from a previously generated snapshot
// Any failure during bootstrapping the blockstore may leave the partial loaded data
// on disk. The consumer, such as peer is expected to keep track of failures and cleanup the
// data explicitly.
func (p *CDBBlockstoreProvider) BootstrapFromSnapshottedTxIDs(snapshotDir string, snapshotInfo *api.SnapshotInfo) (api.BlockStore, error) {
	panic("not implemented")
}

// ExportTxIds creates two files in the specified dir and returns a map that contains
// the mapping between the names of the files and their hashes.
// Technically, the TxIDs appear in the sort order of radix-sort/shortlex. However,
// since practically all the TxIDs are of same length, so the sort order would be the lexical sort order
func (p *CDBBlockstoreProvider) ExportTxIds(dir string, newHashFunc snapshot.NewHashFunc) (map[string][]byte, error) {
	panic("not implemented")
}

// Remove block index and blocks for the given ledgerid (channelID). It is not an error if the channel does not exist.
// This function is not error safe. If this function returns an error or a crash takes place, it is highly likely
// that the data for this ledger is left in an inconsistent state. Opening the ledger again or reusing the previously
// opened ledger can show unknown behavior.
func (p *CDBBlockstoreProvider) Remove(ledgerid string) error {
	panic("not implemented")
}

// Close cleans up the Provider
func (p *CDBBlockstoreProvider) Close() {
}
