/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idstore

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/ledger/dataformat"
	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/msgs"
	"github.com/hyperledger/fabric/core/ledger/util/couchdb"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

var logger = flogging.MustGetLogger("idstore")
var systemID = "fabric_system_"
var inventoryName = "inventory"

//Store contain couchdb instance
type Store struct {
	db               couchDB
	couchMetadataRev string
}

//OpenIDStore return id store
func OpenIDStore(ledgerconfig *ledger.Config) (*Store, error) {
	couchInstance, err := createCouchInstance(ledgerconfig)
	if err != nil {
		return nil, errors.Wrapf(err, "create couchdb instance failed ")
	}

	dbName := couchdb.ConstructBlockchainDBName(systemID, inventoryName)

	// check if it committer role
	if roles.IsCommitter() {
		db, dbErr := couchdb.CreateCouchDatabase(couchInstance, dbName)
		if dbErr != nil {
			return nil, errors.Wrapf(dbErr, "create new couchdb database failed ")
		}
		return newCommitterStore(db)
	}

	db, err := couchdb.NewCouchDatabase(couchInstance, dbName)
	if err != nil {
		return nil, errors.WithMessagef(err, "new couchdb database [%s] failed", dbName)
	}
	return newStore(db, dbName)
}

func newStore(db couchDB, dbName string) (*Store, error) {
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
		return nil, errors.New(fmt.Sprintf("DB index not found: [%s]", dbName))
	}

	s := Store{db, ""}
	return &s, nil
}

func newCommitterStore(db couchDB) (*Store, error) {
	err := createIndices(db)
	if err != nil {
		return nil, errors.Wrapf(err, "create couchdb index failed")
	}

	s := Store{db, ""}

	return &s, nil
}

func createIndices(db couchDB) error {
	err := db.CreateNewIndexWithRetry(inventoryTypeIndexDef, inventoryTypeIndexDoc)
	if err != nil {
		return errors.WithMessagef(err, "creation of inventory metadata index failed")
	}
	return nil
}

func createCouchInstance(ledgerconfig *ledger.Config) (*couchdb.CouchInstance, error) {
	logger.Debugf("constructing CouchDB block storage provider")
	if ledgerconfig == nil {
		return nil, errors.New("ledgerconfig is nil")
	}
	couchDBConfig := ledgerconfig.StateDBConfig.CouchDB
	couchInstance, err := couchdb.CreateCouchInstance(couchDBConfig, &disabled.Provider{})
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
func (s *Store) getLedgerIDValue(ledgerID string) ([]byte, error) {
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

// GetActiveLedgerIDs returns the active ledger IDs
// TODO: Should only return ledger IDs that are active once ledger status is implemented
func (s *Store) GetActiveLedgerIDs() ([]string, error) {
	results, err := queryInventory(s.db, typeLedgerName)
	if err != nil {
		return nil, err
	}

	ledgers := make([]string, 0)
	for _, r := range results {
		ledgerJSON, err := couchValueToJSON(r.Value)
		if err != nil {
			return nil, errors.Wrapf(err, "couchValueToJSON failed")
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

// UpdateLedgerStatus sets the status of the given ledger
func (s *Store) UpdateLedgerStatus(ledgerID string, newStatus msgs.Status) error {
	panic("not implemented")
}

// GetFormat returns the format of the database
func (s *Store) GetFormat() ([]byte, error) {
	// TODO: Should read format from meta data. For now return a hard-coded format.
	return []byte(dataformat.Version20), nil
}

// UpgradeFormat upgrades the database format
func (s *Store) UpgradeFormat() error {
	panic("not implemented")
}

// LedgerIDActive tests whether a ledger ID exists and is active
// TODO: Should return the correct status once ledger status is implemented
func (s *Store) LedgerIDActive(ledgerID string) (active bool, exists bool, err error) {
	// TODO: This is a temporary implementation. Need to implement ledger state (ACTIVE/INACTIVE)
	ids, err := s.GetActiveLedgerIDs()
	if err != nil {
		return false, false, err
	}
	for _, id := range ids {
		if ledgerID == id {
			return true, true, nil
		}
	}
	return false, false, nil
}

// GetGenesisBlock returns the genesis block for the given ledger ID
func (s *Store) GetGenesisBlock(ledgerID string) (*common.Block, error) {
	bytes, err := s.getLedgerIDValue(ledgerID)
	if err != nil {
		return nil, err
	}

	b := &common.Block{}
	err = proto.Unmarshal(bytes, b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

//Close the store
func (s *Store) Close() {
}
