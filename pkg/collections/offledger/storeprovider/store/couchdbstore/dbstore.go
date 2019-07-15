/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package couchdbstore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger/util/couchdb"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/storeprovider/store/api"
)

var logger = flogging.MustGetLogger("ext_offledger")

const (
	fieldsField = "fields"
)

type docType int32

const (
	typeDelete docType = iota
	typeJSON
	typeAttachment
)

const (
	fetchExpiryDataQuery = `
	{
		"selector": {
			"` + expiryField + `": {
				"$lt": %v
			}
		},
		"fields": [
			"` + idField + `",
			"` + revField + `"
		],
		"use_index": ["_design/` + expiryIndexDoc + `", "` + expiryIndexName + `"]
	}`
)

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
	var docs []*couchdb.CouchDoc
	for _, kv := range keyVal {
		dataDoc, err := s.createCouchDoc(kv.Key, kv.Value)
		if err != nil {
			return err
		}
		if dataDoc != nil {
			docs = append(docs, dataDoc)
		}
	}

	if len(docs) == 0 {
		logger.Debugf("[%s] Nothing to do", s.dbName)
		return nil
	}

	_, err := s.db.BatchUpdateDocuments(docs)
	if nil != err {
		return errors.WithMessage(err, fmt.Sprintf("BatchUpdateDocuments failed for [%d] documents", len(docs)))
	}

	return nil
}

// GetKey get dataModel based on key from db
func (s *dbstore) Get(key string) (*api.Value, error) {
	data, err := s.fetchData(s.db, key)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to load key [%s] from db", key)
	}
	return data, nil
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

// Query executes a query against the CouchDB and returns the key/value result set
func (s *dbstore) Query(query string) ([]*api.KeyValue, error) {
	query, err := decorateQuery(query)
	if err != nil {
		return nil, err
	}

	results, _, err := s.db.QueryDocuments(query)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		logger.Debugf("No results for query [%s]", query)
		return nil, nil
	}

	var responses []*api.KeyValue
	for _, result := range results {
		value, err := unmarshalData(result.Value, result.Attachments)
		if err != nil {
			return nil, err
		}
		responses = append(responses, &api.KeyValue{
			Key:   result.ID,
			Value: value,
		})
		logger.Debugf("Added result for query [%s]: Key [%s], Revision [%s], TxID [%s], Expiry [%s], Value [%s]", query, result.ID, value.Revision, value.TxID, value.ExpiryTime, value.Value)
	}

	return responses, nil
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
		updateDoc := &expiryData{ID: doc.ID, Rev: doc.Rev, Deleted: true}
		jsonBytes, err := jsonMarshal(updateDoc)
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

func (s *dbstore) fetchData(db *couchdb.CouchDatabase, key string) (*api.Value, error) {
	doc, _, err := db.ReadDoc(key)
	if err != nil || doc == nil {
		return nil, err
	}
	return unmarshalData(doc.JSONValue, doc.Attachments)
}

func unmarshalData(jsonValue []byte, attachments []*couchdb.AttachmentInfo) (*api.Value, error) {
	// create a generic map unmarshal the json
	jsonResult := make(jsonMap)
	decoder := json.NewDecoder(bytes.NewBuffer(jsonValue))
	decoder.UseNumber()
	if err := decoder.Decode(&jsonResult); err != nil {
		return nil, err
	}

	data := &api.Value{
		Revision: jsonResult[revField].(string),
		TxID:     jsonResult[txnIDField].(string),
	}

	var err error
	data.ExpiryTime, err = getExpiry(jsonResult)
	if err != nil {
		return nil, err
	}

	// Delete the meta-data fields so that they're not returned as part of the value
	delete(jsonResult, idField)
	delete(jsonResult, revField)
	delete(jsonResult, txnIDField)
	delete(jsonResult, expiryField)

	// handle binary or json data
	// nolint : S1031: unnecessary nil check around range (gosimple) -- here actual logic is implemnted for
	// attachement == nil in else block
	if attachments != nil { // binary attachment
		// get binary data from attachment
		for _, attachment := range attachments {
			if attachment.Name == binaryWrapperField {
				data.Value = attachment.AttachmentBytes
			}
		}
	} else {
		// marshal the returned JSON data.
		if data.Value, err = jsonMarshal(jsonResult); err != nil {
			return nil, err
		}
	}
	return data, nil
}

func (s *dbstore) fetchRevision(key string) (string, error) {
	// Get the revision on the current doc
	current, err := s.fetchData(s.db, key)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to load key [%s] from db", key)
	}
	if current != nil {
		return current.Revision, nil
	}
	return "", nil
}

func (s *dbstore) createCouchDoc(key string, value *api.Value) (*couchdb.CouchDoc, error) {
	var err error
	var revision string
	if value != nil && value.Revision != "" {
		revision = value.Revision
	} else {
		revision, err = s.fetchRevision(key)
		if err != nil {
			return nil, err
		}
	}

	jsonMapVal, docTypeVal, err := newJSONMap(key, revision, value)
	if err != nil {
		return nil, err
	}
	if docTypeVal == typeDelete && jsonMapVal == nil {
		logger.Debugf("[%s] Current key/revision not found to delete [%s]", s.dbName, key)
		return nil, nil
	}

	jsonBytes, err := jsonMapVal.toBytes()
	if err != nil {
		return nil, err
	}

	couchDoc := couchdb.CouchDoc{JSONValue: jsonBytes}
	if value != nil && docTypeVal == typeAttachment {
		couchDoc.Attachments = append([]*couchdb.AttachmentInfo{}, asAttachment(value.Value))
	}

	return &couchDoc, nil
}

//-----------------helper functions--------------------//

type expiryData struct {
	ID      string `json:"_id"`
	Rev     string `json:"_rev"`
	Deleted bool   `json:"_deleted"`
}

func fetchExpiryData(db *couchdb.CouchDatabase, expiry time.Time) ([]*expiryData, error) {
	results, _, err := db.QueryDocuments(fmt.Sprintf(fetchExpiryDataQuery, expiry.UnixNano()/int64(time.Millisecond)))
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, nil
	}

	var responses []*expiryData
	for _, result := range results {
		var data expiryData
		err = jsonUnmarshal(result.Value, &data)
		if err != nil {
			return nil, errors.Wrapf(err, "result from DB is not JSON encoded")
		}
		responses = append(responses, &data)
	}

	return responses, nil
}

func getExpiry(jsonResult jsonMap) (time.Time, error) {
	jnExpiry, ok := jsonResult[expiryField].(json.Number)
	if !ok {
		return time.Time{}, errors.Errorf("expiry [%+v] is not a valid JSON number. Type: %s", jsonResult[expiryField], reflect.TypeOf(jsonResult[expiryField]))
	}

	nExpiry, err := jnExpiry.Int64()
	if err != nil {
		return time.Time{}, err
	}
	if nExpiry == 0 {
		return time.Time{}, nil
	}

	return time.Unix(0, nExpiry*int64(time.Millisecond)), nil
}

func getUnixExpiry(expiry time.Time) int64 {
	if expiry.IsZero() {
		return 0
	}
	return expiry.UnixNano() / int64(time.Millisecond)
}

func decorateQuery(query string) (string, error) {
	// create a generic map unmarshal the json
	jsonQuery := make(jsonMap)
	decoder := json.NewDecoder(bytes.NewBuffer([]byte(query)))
	decoder.UseNumber()
	err := decoder.Decode(&jsonQuery)
	if err != nil {
		return "", err
	}

	var fields []interface{}

	// Get the fields specified in the query
	fieldsJSONArray, ok := jsonQuery[fieldsField]
	if ok {
		switch fieldsJSONArray.(type) {
		case []interface{}:
			fields = fieldsJSONArray.([]interface{})
		default:
			return "", errors.New("fields definition must be an array")
		}
	}

	// Append the internal fields
	jsonQuery[fieldsField] = append(fields, idField, revField, txnIDField, expiryField)

	decoratedQuery, err := jsonMarshal(jsonQuery)
	if err != nil {
		return "", err
	}

	logger.Debugf("Decorated query: [%s]", decoratedQuery)
	return string(decoratedQuery), nil
}

func newJSONMap(key string, revision string, value *api.Value) (jsonMap, docType, error) {
	m, docTypeVal, err := jsonMapFromValue(value)
	if err != nil {
		return nil, docTypeVal, err
	}

	if docTypeVal == typeDelete {
		if revision == "" {
			return nil, typeDelete, nil
		}
		m[deletedField] = true
	}

	m[idField] = key
	if value != nil {
		m[txnIDField] = value.TxID
		m[expiryField] = getUnixExpiry(value.ExpiryTime)
	}

	if revision != "" {
		m[revField] = revision
	}

	return m, docTypeVal, nil
}

func jsonMapFromValue(value *api.Value) (jsonMap, docType, error) {
	m := make(jsonMap)

	switch {
	case value == nil || value.Value == nil:
		return m, typeDelete, nil
	case jsonUnmarshal(value.Value, &m) == nil && m != nil:
		// Value is a JSON value. Ensure that it doesn't contain any reserved fields
		if err := m.checkReservedFieldsNotPresent(); err != nil {
			return nil, typeJSON, err
		}
		return m, typeJSON, nil
	default:
		return m, typeAttachment, nil
	}
}

func asAttachment(value []byte) *couchdb.AttachmentInfo {
	attachment := &couchdb.AttachmentInfo{}
	attachment.AttachmentBytes = value
	attachment.ContentType = "application/octet-stream"
	attachment.Name = binaryWrapperField
	return attachment
}

type jsonMap map[string]interface{}

func (v jsonMap) toBytes() ([]byte, error) {
	jsonBytes, err := jsonMarshal(v)
	err = errors.Wrap(err, "error marshalling json data")
	return jsonBytes, err
}

func (v jsonMap) checkReservedFieldsNotPresent() error {
	for fieldName := range v {
		if strings.HasPrefix(fieldName, "~") || strings.HasPrefix(fieldName, "_") {
			return errors.Errorf("field [%s] is not valid for the CouchDB state database", fieldName)
		}
	}
	return nil
}

var jsonUnmarshal = func(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

var jsonMarshal = func(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
