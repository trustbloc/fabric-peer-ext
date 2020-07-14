/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idstore

import (
	"bytes"
	"encoding/json"
	"fmt"

	couchdb "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"

	"github.com/pkg/errors"
)

const (
	idField                    = "_id"
	underConstructionLedgerKey = "under_construction"
	ledgerKeyPrefix            = "ledger_"
	metadataID                 = "metadata"
	blockAttachmentName        = "genesis_block"
	inventoryTypeField         = "type"
	inventoryTypeIndexName     = "by_type"
	inventoryTypeIndexDoc      = "indexMetadataInventory"
	inventoryNameLedgerIDField = "ledger_id"
	statusField                = "status"
	formatField                = "format"
	typeLedgerName             = "ledger"
)

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

func ledgerToCouchDoc(ledgerID string, blockBytes []byte, status string) (*couchdb.CouchDoc, error) {
	jsonMap := make(jsonValue)

	jsonMap[idField] = ledgerIDToKey(ledgerID)
	jsonMap[inventoryTypeField] = typeLedgerName
	jsonMap[inventoryNameLedgerIDField] = ledgerID
	jsonMap[statusField] = status

	jsonBytes, err := jsonMap.toBytes()
	if err != nil {
		return nil, err
	}

	return &couchdb.CouchDoc{
		JSONValue:   jsonBytes,
		Attachments: []*couchdb.AttachmentInfo{blockToAttachment(blockBytes)},
	}, nil
}

func createMetadataDoc(constructionLedger, format string) ([]byte, error) {
	jsonMap := make(jsonValue)

	jsonMap[idField] = metadataID
	jsonMap[underConstructionLedgerKey] = constructionLedger
	jsonMap[formatField] = format

	return jsonMap.toBytes()
}

func ledgerIDToKey(ledgerID string) string {
	return fmt.Sprintf(ledgerKeyPrefix+"%s", ledgerID)
}

func blockToAttachment(blockBytes []byte) *couchdb.AttachmentInfo {
	attachment := &couchdb.AttachmentInfo{}
	attachment.AttachmentBytes = blockBytes
	attachment.ContentType = "application/octet-stream"
	attachment.Name = blockAttachmentName

	return attachment
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

func queryInventory(db couchDatabase, inventoryType string) ([]*couchdb.QueryResult, error) {
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
