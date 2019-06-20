/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idstore

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric/core/ledger/util/couchdb"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/pkg/errors"
)

const (
	idField                    = "_id"
	underConstructionLedgerKey = "under_construction"
	ledgerKeyPrefix            = "ledger_"
	metadataKey                = "metadata"
	blockAttachmentName        = "genesis_block"
	inventoryTypeField         = "type"
	inventoryTypeIndexName     = "by_type"
	inventoryTypeIndexDoc      = "indexMetadataInventory"
	inventoryNameLedgerIDField = "ledger_id"
	typeLedgerName             = "ledger"
)

type couchDB interface {
	ExistsWithRetry() (bool, error)
	IndexDesignDocExistsWithRetry(designDocs ...string) (bool, error)
	CreateNewIndexWithRetry(indexdefinition string, designDoc string) error
	SaveDoc(id string, rev string, couchDoc *couchdb.CouchDoc) (string, error)
	ReadDoc(id string) (*couchdb.CouchDoc, string, error)
	BatchUpdateDocuments(documents []*couchdb.CouchDoc) ([]*couchdb.BatchUpdateResponse, error)
	QueryDocuments(query string) ([]*couchdb.QueryResult, string, error)
}

const inventoryTypeIndexDef = `
	{
		"index": {
			"fields": ["` + inventoryTypeField + `"]
		},
		"name": "` + inventoryTypeIndexName + `",
		"ddoc": "` + inventoryTypeIndexDoc + `",
		"type": "json"
	}`

type jsonValue map[string]interface{}

func (v jsonValue) toBytes() ([]byte, error) {
	return json.Marshal(v)
}

func ledgerToCouchDoc(ledgerID string, gb *common.Block) (*couchdb.CouchDoc, error) {
	jsonMap := make(jsonValue)

	jsonMap[idField] = ledgerIDToKey(ledgerID)
	jsonMap[inventoryTypeField] = typeLedgerName
	jsonMap[inventoryNameLedgerIDField] = ledgerID

	jsonBytes, err := jsonMap.toBytes()
	if err != nil {
		return nil, err
	}

	couchDoc := couchdb.CouchDoc{JSONValue: jsonBytes}

	attachment, err := blockToAttachment(gb)
	if err != nil {
		return nil, err
	}

	attachments := append([]*couchdb.AttachmentInfo{}, attachment)
	couchDoc.Attachments = attachments

	return &couchDoc, nil
}

func createMetadataDoc(constructionLedger string) (*couchdb.CouchDoc, error) {
	jsonMap := make(jsonValue)

	jsonMap[idField] = metadataKey
	jsonMap[underConstructionLedgerKey] = constructionLedger

	jsonBytes, err := jsonMap.toBytes()
	if err != nil {
		return nil, err
	}

	couchDoc := couchdb.CouchDoc{JSONValue: jsonBytes}

	return &couchDoc, nil
}

func ledgerIDToKey(ledgerID string) string {
	return fmt.Sprintf(ledgerKeyPrefix+"%s", ledgerID)
}

func blockToAttachment(block *common.Block) (*couchdb.AttachmentInfo, error) {
	blockBytes, err := proto.Marshal(block)
	if err != nil {
		return nil, errors.Wrap(err, "marshaling block failed")
	}

	attachment := &couchdb.AttachmentInfo{}
	attachment.AttachmentBytes = blockBytes
	attachment.ContentType = "application/octet-stream"
	attachment.Name = blockAttachmentName

	return attachment, nil
}

func couchDocToJSON(doc *couchdb.CouchDoc) (jsonValue, error) {
	return couchValueToJSON(doc.JSONValue)
}

func couchValueToJSON(value []byte) (jsonValue, error) {
	// create a generic map unmarshal the json
	jsonResult := make(map[string]interface{})
	decoder := json.NewDecoder(bytes.NewBuffer(value))
	decoder.UseNumber()

	err := decoder.Decode(&jsonResult)
	if err != nil {
		return nil, errors.Wrap(err, "result from DB is not JSON encoded")
	}

	return jsonResult, nil
}

func queryInventory(db couchDB, inventoryType string) ([]*couchdb.QueryResult, error) {
	const queryFmt = `
	{
		"selector": {
			"` + inventoryTypeField + `": {
				"$eq": "%s"
			}
		},
		"use_index": ["_design/` + inventoryTypeIndexDoc + `", "` + inventoryTypeIndexName + `"]
	}`

	results, _, err := db.QueryDocuments(fmt.Sprintf(queryFmt, inventoryType))
	if err != nil {
		return nil, err
	}
	return results, nil
}
