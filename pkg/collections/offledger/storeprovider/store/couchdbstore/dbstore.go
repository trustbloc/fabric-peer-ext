/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package couchdbstore

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger/util/couchdb"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/storeprovider/store/api"
)

var (
	logger          = flogging.MustGetLogger("ext_offledger")
	compositeKeySep = "!"
)

const (
	fetchExpiryDataQuery = `
	{
		"selector": {
			"` + expiryField + `": {
				"$lt": %v
			}
		},
		"use_index": ["_design/` + expiryIndexDoc + `", "` + expiryIndexName + `"]
	}`
)

// dataModel couch doc dataModel
type dataModel struct {
	ID      string `json:"_id"`
	Rev     string `json:"_rev,omitempty"`
	Data    string `json:"dataModel"`
	TxnID   string `json:"txnID"`
	Expiry  int64  `json:"expiry"`
	Deleted bool   `json:"_deleted"`
}

type dbstore struct {
	dbName string
	db     *couchdb.CouchDatabase
}

// newDBStore constructs an instance of db store
func newDBStore(db *couchdb.CouchDatabase, dbName string) *dbstore {
	return &dbstore{dbName, db}
}

//-----------------Interface implementation functions--------------------//
// AddKey adds dataModel to db
func (s *dbstore) Put(keyVal ...*api.KeyValue) error {
	docs := make([]*couchdb.CouchDoc, 0)
	for _, kv := range keyVal {

		dataDoc, err := createCouchDoc(string(encodeKey(kv.Key, time.Time{})), kv.Value)
		if err != nil {
			return err
		}
		if dataDoc != nil {
			docs = append(docs, dataDoc)
		}
	}

	_, err := s.db.BatchUpdateDocuments(docs)
	if nil != err {
		return errors.WithMessage(err, fmt.Sprintf("BatchUpdateDocuments failed for [%d] documents", len(docs)))
	}

	return nil
}

// GetKey get dataModel based on key from db
func (s *dbstore) Get(key string) (*api.Value, error) {
	data, err := fetchData(s.db, string(encodeKey(key, time.Time{})))
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to load key [%s] from db", key)
	}

	if data != nil {
		val := &api.Value{Value: []byte(data.Data), TxID: data.TxnID, ExpiryTime: time.Unix(0, data.Expiry)}
		return val, nil
	}

	return nil, nil
}

// GetMultiple retrieves values for multiple keys at once
func (s *dbstore) GetMultiple(keys ...string) ([]*api.Value, error) {
	values := make([]*api.Value, len(keys))
	for i, k := range keys {
		v, err := s.Get(k)
		if err != nil {
			return nil, err
		}
		values[i] = v
	}

	return values, nil
}

// DeleteExpiredKeys delete expired keys from db
func (s *dbstore) DeleteExpiredKeys() error {
	data, err := fetchExpiryData(s.db, time.Now())
	if err != nil {
		return err
	}
	if len(data) == 0 {
		logger.Debugf("No keys to delete from db")
		return nil
	}

	docs := make([]*couchdb.CouchDoc, 0)
	docIDs := make([]string, 0)
	for _, doc := range data {
		updateDoc := &dataModel{ID: doc.ID, Data: doc.Data, TxnID: doc.TxnID, Expiry: doc.Expiry, Rev: doc.Rev, Deleted: true}
		jsonBytes, err := json.Marshal(updateDoc)
		if err != nil {
			return err
		}
		couchDoc := couchdb.CouchDoc{JSONValue: jsonBytes}
		docs = append(docs, &couchDoc)
		docIDs = append(docIDs, updateDoc.ID)
	}

	if len(docs) > 0 {
		_, err := s.db.BatchUpdateDocuments(docs)
		if err != nil {
			return errors.WithMessage(err, fmt.Sprintf("BatchUpdateDocuments failed for [%d] documents", len(docs)))
		}
		logger.Debugf("Deleted expired keys %s from db", docIDs)
	}

	return nil
}

// Close db
func (s *dbstore) Close() {
}

//-----------------helper functions--------------------//
func encodeKey(key string, expiryTime time.Time) []byte {
	var compositeKey []byte
	if !expiryTime.IsZero() {
		compositeKey = append(compositeKey, []byte(fmt.Sprintf("%d", expiryTime.UnixNano()/int64(time.Millisecond)))...)
		compositeKey = append(compositeKey, compositeKeySep...)
	}
	compositeKey = append(compositeKey, []byte(key)...)
	return compositeKey
}

//-----------------database helper functions--------------------//
func fetchData(db *couchdb.CouchDatabase, key string) (*dataModel, error) {
	doc, _, err := db.ReadDoc(key)
	if err != nil {
		return nil, err
	}

	if doc == nil {
		return nil, nil
	}

	var data dataModel
	err = json.Unmarshal(doc.JSONValue, &data)
	if err != nil {
		return nil, errors.Wrapf(err, "Result from DB is not JSON encoded")
	}

	return &data, nil
}

func createCouchDoc(key string, value *api.Value) (*couchdb.CouchDoc, error) {
	data := &dataModel{ID: key, Data: string(value.Value), TxnID: value.TxID, Expiry: value.ExpiryTime.UnixNano() / int64(time.Millisecond)}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrapf(err, "result from DB is not JSON encoded")
	}

	couchDoc := couchdb.CouchDoc{JSONValue: jsonBytes}

	return &couchDoc, nil
}

func fetchExpiryData(db *couchdb.CouchDatabase, expiry time.Time) ([]*dataModel, error) {
	results, _, err := db.QueryDocuments(fmt.Sprintf(fetchExpiryDataQuery, expiry.UnixNano()/int64(time.Millisecond)))
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, nil
	}

	var responses []*dataModel
	for _, result := range results {
		var data dataModel
		err = json.Unmarshal(result.Value, &data)
		if err != nil {
			return nil, errors.Wrapf(err, "result from DB is not JSON encoded")
		}
		responses = append(responses, &data)
	}

	return responses, nil
}
