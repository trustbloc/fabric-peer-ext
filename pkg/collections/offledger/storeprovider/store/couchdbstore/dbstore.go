/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package couchdbstore

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb/msgs"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/core/ledger/util/couchdb"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/storeprovider/store/api"
)

var (
	logger          = flogging.MustGetLogger("ext_offledger")
	compositeKeySep = "!"
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
		dataDoc, err := s.createCouchDoc(string(encodeKey(kv.Key, time.Time{})), kv.Value)
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
	data, err := s.fetchData(s.db, string(encodeKey(key, time.Time{})))
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

func (s *dbstore) fetchData(db *couchdb.CouchDatabase, key string) (*api.Value, error) {
	doc, _, err := db.ReadDoc(key)
	if err != nil || doc == nil {
		return nil, err
	}

	// create a generic map unmarshal the json
	jsonResult := make(jsonMap)
	decoder := json.NewDecoder(bytes.NewBuffer(doc.JSONValue))
	decoder.UseNumber()
	if err = decoder.Decode(&jsonResult); err != nil {
		return nil, err
	}

	data := &api.Value{
		Revision: jsonResult[revField].(string),
		TxID:     jsonResult[txnIDField].(string),
	}

	data.ExpiryTime, err = getExpiry(jsonResult)
	if err != nil {
		return nil, err
	}

	// Delete the meta-data fields so that they're not returned as part of the value
	delete(jsonResult, idField)
	delete(jsonResult, revField)
	delete(jsonResult, txnIDField)
	delete(jsonResult, expiryField)
	delete(jsonResult, versionField)

	// handle binary or json data
	if doc.Attachments != nil { // binary attachment
		// get binary data from attachment
		for _, attachment := range doc.Attachments {
			if attachment.Name == binaryWrapperField {
				data.Value = attachment.AttachmentBytes
			}
		}
	} else {
		// marshal the returned JSON data.
		if data.Value, err = json.Marshal(jsonResult); err != nil {
			return nil, err
		}
	}
	return data, nil
}

func (s *dbstore) fetchRevision(key string) (string, error) {
	// Get the revision on the current doc
	current, err := s.fetchData(s.db, string(encodeKey(key, time.Time{})))
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

	jsonMap, docType, err := newJSONMap(key, revision, value)
	if err != nil {
		return nil, err
	}
	if docType == typeDelete && jsonMap == nil {
		logger.Debugf("[%s] Current key/revision not found to delete [%s]", s.dbName, key)
		return nil, nil
	}

	jsonBytes, err := jsonMap.toBytes()
	if err != nil {
		return nil, err
	}

	couchDoc := couchdb.CouchDoc{JSONValue: jsonBytes}
	if value != nil && docType == typeAttachment {
		couchDoc.Attachments = append([]*couchdb.AttachmentInfo{}, asAttachment(value.Value))
	}

	return &couchDoc, nil
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
		err = json.Unmarshal(result.Value, &data)
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
	return time.Unix(0, nExpiry), nil
}

func newJSONMap(key string, revision string, value *api.Value) (jsonMap, docType, error) {
	m, docType, err := jsonMapFromValue(value)
	if err != nil {
		return nil, docType, err
	}

	if docType == typeDelete {
		if revision == "" {
			return nil, typeDelete, nil
		}
		m[deletedField] = true
	}

	m[idField] = key
	if value != nil {
		m[txnIDField] = value.TxID
		m[expiryField] = value.ExpiryTime.UnixNano() / int64(time.Millisecond)
	}

	if revision != "" {
		m[revField] = revision
	}

	// Add a dummy version field so that Fabric doesn't complain when retrieving current using rich queries
	verAndMetadata, err := encodeVersionAndMetadata()
	if err != nil {
		return nil, docType, err
	}
	m[versionField] = verAndMetadata

	return m, docType, nil
}

func jsonMapFromValue(value *api.Value) (jsonMap, docType, error) {
	m := make(jsonMap)

	switch {
	case value == nil || value.Value == nil:
		return m, typeDelete, nil
	case json.Unmarshal(value.Value, &m) == nil && m != nil:
		// Value is a JSON value. Ensure that it doesn't contain any reserved fields
		if err := m.checkReservedFieldsNotPresent(); err != nil {
			return nil, typeJSON, err
		}
		return m, typeJSON, nil
	default:
		return m, typeAttachment, nil
	}
}

func encodeVersionAndMetadata() (string, error) {
	version := &version.Height{
		BlockNum: 1000,
		TxNum:    0,
	}

	msg := &msgs.VersionFieldProto{
		VersionBytes: version.ToBytes(),
	}
	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		return "", err
	}
	msgBase64 := base64.StdEncoding.EncodeToString(msgBytes)
	encodedVersionField := append([]byte{byte(0)}, []byte(msgBase64)...)
	return string(encodedVersionField), nil
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
	jsonBytes, err := json.Marshal(v)
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
