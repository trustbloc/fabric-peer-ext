/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idstore

import (
	"fmt"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/core/ledger/kvledger/idstore"
	"github.com/hyperledger/fabric/core/ledger/util/couchdb"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

var logger = flogging.MustGetLogger("idstore")

const (
	systemID      = "fabric_system_"
	inventoryName = "inventory"
)

//Store contain couchdb instance
type Store struct {
	db               *couchdb.CouchDatabase
	couchMetadataRev string
}

//OpenIDStore return id store
func OpenIDStore(path string) idstore.IDStore {
	couchInstance, err := createCouchInstance()
	if err != nil {
		logger.Errorf("create couchdb instance failed %s", err.Error())
		return nil
	}

	inventoryDBName := couchdb.ConstructBlockchainDBName(systemID, inventoryName)
	if roles.IsCommitter() {
		return newCommitterStore(couchInstance, inventoryDBName)
	}
	s, err := newStore(couchInstance, inventoryDBName)
	if err != nil {
		logger.Error(err.Error())
		return nil
	}
	return s
}

func newStore(couchInstance *couchdb.CouchInstance, dbName string) (idstore.IDStore, error) {
	db, err := couchdb.NewCouchDatabase(couchInstance, dbName)
	if err != nil {
		return nil, errors.WithMessagef(err, "create new couchdb database called [%s] failed", dbName)
	}

	dbExists, err := db.ExistsWithRetry()
	if err != nil {
		return nil, errors.WithMessagef(err, "check couchdb [%s] exist failed", dbName)
	}
	if !dbExists {
		return nil, errors.New(fmt.Sprintf("DB not found: [%s]", dbName))
	}

	indexExists, err := db.IndexDesignDocExistsWithRetry(inventoryTypeIndexDoc)
	if err != nil {
		return nil, errors.WithMessagef(err, "check couchdb [%s] index exist failed", dbName)

	}
	if !indexExists {
		return nil, errors.New(fmt.Sprintf("DB index not found: [%s]", db.DBName))
	}

	s := Store{db, ""}
	return &s, nil
}

func newCommitterStore(couchInstance *couchdb.CouchInstance, dbName string) idstore.IDStore {
	db, err := couchdb.CreateCouchDatabase(couchInstance, dbName)
	if err != nil {
		logger.Errorf("create new couchdb database failed %s", err.Error())
		return nil
	}

	err = createIndices(db)
	if err != nil {
		logger.Errorf("create couchdb index failed %s", err.Error())
		return nil
	}

	s := Store{db, ""}

	return &s
}

func createIndices(db *couchdb.CouchDatabase) error {
	err := db.CreateNewIndexWithRetry(inventoryTypeIndexDef, inventoryTypeIndexDoc)
	if err != nil {
		return errors.WithMessagef(err, "creation of inventory metadata index failed for [%s]", db.DBName)
	}
	return nil
}

func createCouchInstance() (*couchdb.CouchInstance, error) {
	logger.Debugf("constructing CouchDB block storage provider")
	couchDBDef := couchdb.GetCouchDBDefinition()
	couchInstance, err := couchdb.CreateCouchInstance(couchDBDef.URL, couchDBDef.Username, couchDBDef.Password,
		couchDBDef.MaxRetries, couchDBDef.MaxRetriesOnStartup, couchDBDef.RequestTimeout, couchDBDef.CreateGlobalChangesDB, &disabled.Provider{})
	if err != nil {
		return nil, errors.WithMessage(err, "obtaining CouchDB instance failed")
	}

	return couchInstance, nil
}

//SetUnderConstructionFlag set under construction flag
func (s *Store) SetUnderConstructionFlag(ledgerID string) error {
	doc, err := createMetadataDoc(ledgerID)
	if err != nil {
		return err
	}

	rev, err := s.db.SaveDoc(metadataKey, s.couchMetadataRev, doc)
	if err != nil {
		return errors.WithMessage(err, "update of metadata in CouchDB failed")
	}

	s.couchMetadataRev = rev

	logger.Debugf("updated metadata in CouchDB inventory [%s]", rev)
	return nil
}

//UnsetUnderConstructionFlag unset under construction flag
func (s *Store) UnsetUnderConstructionFlag() error {
	doc, err := createMetadataDoc("")
	if err != nil {
		return err
	}

	rev, err := s.db.SaveDoc(metadataKey, s.couchMetadataRev, doc)
	if err != nil {
		return errors.WithMessage(err, "update of metadata in CouchDB failed")
	}

	s.couchMetadataRev = rev

	logger.Debugf("updated metadata in CouchDB inventory [%s]", rev)
	return nil
}

//GetUnderConstructionFlag get under construction flag
func (s *Store) GetUnderConstructionFlag() (string, error) {
	doc, _, err := s.db.ReadDoc(metadataKey)
	if err != nil {
		return "", errors.WithMessage(err, "retrieval of metadata from CouchDB inventory failed")
	}

	// if metadata does not exist, assume that there is nothing under construction.
	if doc == nil {
		return "", nil
	}

	metadata, err := couchDocToJSON(doc)
	if err != nil {
		return "", errors.WithMessage(err, "metadata in CouchDB inventory is invalid")
	}

	constructionLedgerUT := metadata[underConstructionLedgerKey]
	constructionLedger, ok := constructionLedgerUT.(string)
	if !ok {
		return "", errors.New("metadata under construction key in CouchDB inventory is invalid")
	}

	return constructionLedger, nil
}

//CreateLedgerID create ledger id
func (s *Store) CreateLedgerID(ledgerID string, gb *common.Block) error {
	exists, err := s.LedgerIDExists(ledgerID)
	if err != nil {
		return err
	}

	if exists {
		return errors.Errorf("ledger already exists [%s]", ledgerID)
	}

	doc, err := ledgerToCouchDoc(ledgerID, gb)
	if err != nil {
		return err
	}

	rev, err := s.db.BatchUpdateDocuments([]*couchdb.CouchDoc{doc})
	if err != nil {
		return errors.WithMessagef(err, "creation of ledger failed [%s]", ledgerID)
	}

	err = s.UnsetUnderConstructionFlag()
	if err != nil {
		return err
	}

	logger.Debugf("created ledger in CouchDB inventory [%s, %s]", ledgerID, rev)
	return nil
}

//LedgerIDExists check ledger id exists
func (s *Store) LedgerIDExists(ledgerID string) (bool, error) {
	doc, _, err := s.db.ReadDoc(ledgerIDToKey(ledgerID))
	if err != nil {
		return false, err
	}

	exists := doc != nil
	return exists, nil
}

//GetLedgeIDValue get ledger id value
func (s *Store) GetLedgeIDValue(ledgerID string) ([]byte, error) {
	doc, _, err := s.db.ReadDoc(ledgerIDToKey(ledgerID))
	if err != nil {
		return nil, err
	}
	for _, v := range doc.Attachments {
		if v.Name == blockAttachmentName {
			return v.AttachmentBytes, nil
		}
	}
	return nil, nil
}

//GetAllLedgerIds get all ledger ids
func (s *Store) GetAllLedgerIds() ([]string, error) {
	results, err := queryInventory(s.db, typeLedgerName)
	if err != nil {
		return nil, err
	}

	ledgers := make([]string, 0)
	for _, r := range results {
		ledgerJSON, err := couchValueToJSON(r.Value)
		if err != nil {
			return nil, err
		}

		ledgerIDUT, ok := ledgerJSON[inventoryNameLedgerIDField]
		if !ok {
			return nil, errors.Errorf("ledger inventory document is invalid [%s]", r.ID)
		}

		ledgerID, ok := ledgerIDUT.(string)
		if !ok {
			return nil, errors.Errorf("ledger inventory document value is invalid [%s]", r.ID)
		}

		ledgers = append(ledgers, ledgerID)
	}

	return ledgers, nil
}

//Close the store
func (s *Store) Close() {
}
