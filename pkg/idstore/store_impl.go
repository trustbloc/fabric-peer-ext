/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idstore

import (
	"bytes"
	"fmt"
	"sync/atomic"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/ledger/dataformat"
	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/msgs"
	couchdb "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"
	"github.com/pkg/errors"

	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

var logger = flogging.MustGetLogger("ext_idstore")

const (
	fabricInternalDBName = "fabric__internal"
)

var (
	systemID      = "fabric_system_"
	inventoryName = "inventory"
)

type dbProvider = func(couchInstance *couchdb.CouchInstance, dbName string) (couchDatabase, error)

type couchDatabase interface {
	ExistsWithRetry() (bool, error)
	IndexDesignDocExistsWithRetry(designDocs ...string) (bool, error)
	CreateNewIndexWithRetry(indexdefinition string, designDoc string) error
	SaveDoc(id string, rev string, couchDoc *couchdb.CouchDoc) (string, error)
	ReadDoc(id string) (*couchdb.CouchDoc, string, error)
	BatchUpdateDocuments(documents []*couchdb.CouchDoc) ([]*couchdb.BatchUpdateResponse, error)
	QueryDocuments(query string) ([]*couchdb.QueryResult, string, error)
}

//Store contain couchdb instance
type Store struct {
	db                couchDatabase
	couchInstance     *couchdb.CouchInstance
	closed            uint32
	createMetadataDoc metadataProvider
}

//OpenIDStore return id store
func OpenIDStore(ledgerconfig *ledger.Config) (*Store, error) {
	logger.Info("Opening ID store")

	couchInstance, err := createCouchInstance(ledgerconfig)
	if err != nil {
		return nil, errors.Wrapf(err, "create couchdb instance failed ")
	}

	dbName := couchdb.ConstructBlockchainDBName(systemID, inventoryName)

	// check if it committer role
	if roles.IsCommitter() {
		return newCommitterStore(dbName, couchInstance,
			func(couchInstance *couchdb.CouchInstance, dbName string) (couchDatabase, error) {
				return couchdb.CreateCouchDatabase(couchInstance, dbName)
			},
			createMetadataDoc,
		)
	}

	return newStore(dbName, couchInstance,
		func(couchInstance *couchdb.CouchInstance, dbName string) (couchDatabase, error) {
			return couchdb.NewCouchDatabase(couchInstance, dbName)
		},
	)
}

func newStore(dbName string, couchInstance *couchdb.CouchInstance, newCouchDatabase dbProvider) (*Store, error) {
	db, err := newCouchDatabase(couchInstance, dbName)
	if err != nil {
		return nil, errors.WithMessagef(err, "new couchdb database [%s] failed", dbName)
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
		return nil, errors.New(fmt.Sprintf("DB index not found: [%s]", dbName))
	}

	return &Store{
		db:                db,
		couchInstance:     couchInstance,
		createMetadataDoc: createMetadataDoc,
	}, nil
}

type metadataProvider = func(constructionLedger, format string) ([]byte, error)

func newCommitterStore(dbName string, couchInstance *couchdb.CouchInstance, createCouchDatabase dbProvider, createMetadataDoc metadataProvider) (*Store, error) {
	emptyDB, err := isEmpty(couchInstance)
	if err != nil {
		return nil, err
	}

	logger.Debugf("ID store DB is empty: %t", emptyDB)

	db, dbErr := createCouchDatabase(couchInstance, dbName)
	if dbErr != nil {
		return nil, errors.Wrapf(dbErr, "create new couchdb database failed ")
	}

	if emptyDB {
		// add format key to a new db
		jsonBytes, e := createMetadataDoc("", dataformat.CurrentFormat)
		if e != nil {
			return nil, e
		}

		_, e = db.SaveDoc(metadataID, "", &couchdb.CouchDoc{JSONValue: jsonBytes})
		if e != nil {
			return nil, errors.WithMessage(e, "update of metadata in CouchDB failed")
		}

		logger.Infof("Created metadata in CouchDB inventory")
	}

	err = createIndices(db)
	if err != nil {
		return nil, errors.Wrapf(err, "create couchdb index failed")
	}

	return &Store{
		db:                db,
		couchInstance:     couchInstance,
		createMetadataDoc: createMetadataDoc,
	}, nil
}

func createIndices(db couchDatabase) error {
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
	if s.isClosed() {
		return errors.New("ID store is closed")
	}

	metadata, rev, err := s.getMetadata()
	if err != nil {
		return err
	}

	var format string

	f, ok := metadata[formatField]
	if ok {
		format = f.(string)
	}

	jsonBytes, err := s.createMetadataDoc(ledgerID, format)
	if err != nil {
		return err
	}

	_, err = s.db.SaveDoc(metadataID, rev, &couchdb.CouchDoc{JSONValue: jsonBytes})
	if err != nil {
		return errors.WithMessage(err, "update of metadata in CouchDB failed")
	}

	logger.Debugf("updated metadata in CouchDB inventory [%s]", rev)

	return nil
}

//UnsetUnderConstructionFlag unset under construction flag
func (s *Store) UnsetUnderConstructionFlag() error {
	if s.isClosed() {
		return errors.New("ID store is closed")
	}

	metadata, rev, err := s.getMetadata()
	if err != nil {
		return err
	}

	var format string

	f, ok := metadata[formatField]
	if ok {
		format = f.(string)
	}

	jsonBytes, err := s.createMetadataDoc("", format)
	if err != nil {
		return err
	}

	_, err = s.db.SaveDoc(metadataID, rev, &couchdb.CouchDoc{JSONValue: jsonBytes})
	if err != nil {
		return errors.WithMessage(err, "update of metadata in CouchDB failed")
	}

	logger.Debugf("updated metadata in CouchDB inventory [%s]", rev)

	return nil
}

//GetUnderConstructionFlag get under construction flag
func (s *Store) GetUnderConstructionFlag() (string, error) {
	if s.isClosed() {
		return "", errors.New("ID store is closed")
	}

	metadata, _, err := s.getMetadata()
	if err != nil {
		return "", err
	}

	// if metadata does not exist, assume that there is nothing under construction.
	if metadata == nil {
		return "", nil
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
	if s.isClosed() {
		return errors.New("ID store is closed")
	}

	exists, err := s.LedgerIDExists(ledgerID)
	if err != nil {
		return err
	}

	if exists {
		return errors.Errorf("ledger already exists [%s]", ledgerID)
	}

	blockBytes, err := proto.Marshal(gb)
	if err != nil {
		return errors.Wrap(err, "marshaling block failed")
	}

	doc, err := ledgerToCouchDoc(ledgerID, blockBytes, msgs.Status_ACTIVE.String())
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
	if s.isClosed() {
		return false, errors.New("ID store is closed")
	}

	doc, _, err := s.db.ReadDoc(ledgerIDToKey(ledgerID))
	if err != nil {
		return false, err
	}

	exists := doc != nil
	return exists, nil
}

//GetLedgeIDValue get ledger id value
func (s *Store) getLedgerIDValue(ledgerID string) (jsonValue, []byte, string, error) {
	doc, rev, err := s.db.ReadDoc(ledgerIDToKey(ledgerID))
	if err != nil {
		return nil, nil, "", err
	}

	if doc == nil {
		return nil, nil, rev, nil
	}

	var value jsonValue
	var genesysBlock []byte

	if len(doc.JSONValue) > 0 {
		value, err = couchValueToJSON(doc.JSONValue)
		if err != nil {
			return nil, nil, "", errors.WithMessage(err, "unable to unmarshal JSON value")
		}
	}

	for _, v := range doc.Attachments {
		if v.Name == blockAttachmentName {
			genesysBlock = v.AttachmentBytes
			break
		}
	}

	return value, genesysBlock, rev, nil
}

// GetActiveLedgerIDs returns the active ledger IDs
func (s *Store) GetActiveLedgerIDs() ([]string, error) {
	if s.isClosed() {
		return nil, errors.New("ID store is closed")
	}

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

		status := ledgerJSON[statusField]

		logger.Debugf("Status of ledger [%s] is [%s]", ledgerID, status)

		if status == nil || status.(string) != msgs.Status_ACTIVE.String() {
			logger.Debugf("Skipping ledger [%s] since its status is [%s]", ledgerID, status)
			continue
		}

		ledgers = append(ledgers, ledgerID)
	}

	return ledgers, nil
}

// UpdateLedgerStatus sets the status of the given ledger
func (s *Store) UpdateLedgerStatus(ledgerID string, newStatus msgs.Status) error {
	if s.isClosed() {
		return errors.New("ID store is closed")
	}

	ledgerMetadata, blockBytes, rev, err := s.getLedgerIDValue(ledgerID)
	if err != nil {
		return err
	}

	if ledgerMetadata == nil {
		logger.Errorf("Ledger [%s] does not exist", ledgerID)
		return errors.New("LedgerID does not exist")
	}

	if ledgerMetadata[statusField] == newStatus.String() {
		logger.Debugf("Ledger [%s] is already in [%s] status, nothing to do", ledgerID, newStatus)
		return nil
	}

	logger.Infof("Updating ledger [%s] status to [%s]", ledgerID, newStatus)

	doc, err := ledgerToCouchDoc(ledgerID, blockBytes, newStatus.String())
	if err != nil {
		return err
	}

	_, err = s.db.SaveDoc(ledgerIDToKey(ledgerID), rev, doc)
	if err != nil {
		logger.Errorf("Error updating ledger [%s] status to [%s]: %s", ledgerID, newStatus, err)
		return err
	}

	return nil
}

// GetFormat returns the format of the database
func (s *Store) GetFormat() ([]byte, error) {
	if s.isClosed() {
		return nil, errors.New("ID store is closed")
	}

	metadata, _, err := s.getMetadata()
	if err != nil {
		return nil, errors.WithMessage(err, "unable to read metadata")
	}

	format, ok := metadata[formatField]
	if !ok {
		return []byte(dataformat.PreviousFormat), nil
	}

	return []byte(format.(string)), nil
}

// UpgradeFormat upgrades the database format
func (s *Store) UpgradeFormat() error {
	if s.isClosed() {
		return errors.New("ID store is closed")
	}

	eligible, err := s.checkUpgradeEligibility()
	if err != nil {
		return err
	}

	if !eligible {
		return nil
	}

	logger.Infof("Upgrading ledgerProvider database to the new format %s", dataformat.CurrentFormat)

	metadata, rev, err := s.getMetadata()
	if err != nil {
		return nil
	}

	jsonBytes, err := s.createMetadataDoc(getUnderConstructionID(metadata), dataformat.CurrentFormat)
	if err != nil {
		return err
	}

	rev, err = s.db.SaveDoc(metadataID, rev, &couchdb.CouchDoc{JSONValue: jsonBytes})
	if err != nil {
		return errors.WithMessage(err, "update of metadata in CouchDB failed")
	}

	results, err := queryInventory(s.db, typeLedgerName)
	if err != nil {
		return err
	}

	for _, r := range results {
		if e := s.activateLedger(r, rev); e != nil {
			return e
		}
	}

	return nil
}

func (s *Store) activateLedger(r *couchdb.QueryResult, rev string) error {
	ledgerJSON, err := couchValueToJSON(r.Value)
	if err != nil {
		return errors.Wrapf(err, "couchValueToJSON failed")
	}

	ledgerJSON[statusField] = msgs.Status_ACTIVE.String()

	ledgerID, ok := ledgerJSON[inventoryNameLedgerIDField].(string)
	if !ok {
		return errors.Errorf("ledger inventory document value is invalid [%s]", r.ID)
	}

	var blockBytes = getGenesysBlock(r.Attachments)
	if len(blockBytes) == 0 {
		return errors.Errorf("ledger inventory document value is invalid [%s]", r.ID)
	}

	doc, err := ledgerToCouchDoc(ledgerID, blockBytes, msgs.Status_ACTIVE.String())
	if err != nil {
		return err
	}

	_, err = s.db.SaveDoc(ledgerID, rev, doc)
	if err != nil {
		return errors.WithMessage(err, "update of metadata in CouchDB failed")
	}

	return nil
}

// LedgerIDActive tests whether a ledger ID exists and is active
func (s *Store) LedgerIDActive(ledgerID string) (active bool, exists bool, err error) {
	if s.isClosed() {
		return false, false, errors.New("ID store is closed")
	}

	value, _, _, err := s.getLedgerIDValue(ledgerID)
	if err != nil {
		return false, false, err
	}

	if value == nil {
		logger.Debugf("Ledger [%s] not found", ledgerID)
		return false, false, nil
	}

	status, ok := value[statusField]
	return ok && (status.(string) == msgs.Status_ACTIVE.String()), true, nil
}

// GetGenesisBlock returns the genesis block for the given ledger ID
func (s *Store) GetGenesisBlock(ledgerID string) (*common.Block, error) {
	if s.isClosed() {
		return nil, errors.New("ID store is closed")
	}

	_, bytes, _, err := s.getLedgerIDValue(ledgerID)
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
	if atomic.CompareAndSwapUint32(&s.closed, 0, 1) {
		logger.Infof("Store was successfully closed")
	} else {
		logger.Debugf("Store is already closed")
	}
}

func (s *Store) isClosed() bool {
	return atomic.LoadUint32(&s.closed) == 1
}

// checkUpgradeEligibility checks if the format is eligible to upgrade.
// It returns true if the format is eligible to upgrade to the current format.
// It returns false if either the format is the current format or the db is empty.
// Otherwise, an ErrFormatMismatch is returned.
func (s *Store) checkUpgradeEligibility() (bool, error) {
	emptydb, err := s.couchInstance.IsEmpty([]string{fabricInternalDBName})
	if err != nil {
		return false, err
	}

	if emptydb {
		logger.Warnf("Ledger database %s is empty, nothing to upgrade")
		return false, nil
	}

	format, err := s.GetFormat()
	if err != nil {
		return false, err
	}

	if bytes.Equal(format, []byte(dataformat.CurrentFormat)) {
		logger.Debugf("Ledger database has current data format, nothing to upgrade")
		return false, nil
	}

	if !bytes.Equal(format, []byte(dataformat.PreviousFormat)) {
		err = &dataformat.ErrFormatMismatch{
			ExpectedFormat: dataformat.PreviousFormat,
			Format:         string(format),
			DBInfo:         fmt.Sprintf("CouchDB for channel-IDs"),
		}
		return false, err
	}

	return true, nil
}

func (s *Store) getMetadata() (jsonValue, string, error) {
	doc, rev, err := s.db.ReadDoc(metadataID)
	if err != nil {
		return nil, "", errors.WithMessage(err, "retrieval of metadata from CouchDB inventory failed")
	}

	var metadata jsonValue
	if doc != nil {
		metadata, err = couchDocToJSON(doc)
		if err != nil {
			return nil, "", errors.WithMessage(err, "metadata in CouchDB inventory is invalid")
		}
	}

	return metadata, rev, nil
}

var isEmpty = func(couchInstance *couchdb.CouchInstance) (bool, error) {
	return couchInstance.IsEmpty([]string{fabricInternalDBName})
}

func getGenesysBlock(attachments []*couchdb.AttachmentInfo) []byte {
	for _, v := range attachments {
		if v.Name == blockAttachmentName {
			return v.AttachmentBytes
		}
	}

	return nil
}

func getUnderConstructionID(metadata jsonValue) string {
	if metadata == nil {
		return ""
	}

	id, ok := metadata[underConstructionLedgerKey]
	if !ok {
		return ""
	}

	return id.(string)
}
