/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pvtdatastorage

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/ledger/util/couchdb"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/pvtdatastorage/common"
)

const (
	idField                       = "_id"
	revField                      = "_rev"
	dataField                     = "data"
	expiryField                   = "expiry"
	expiringBlkNumsField          = "expiringBlkNums"
	expiringBlockNumbersIndexName = "by_expiring_block_numbers"
	expiringBlockNumbersIndexDoc  = "indexExpiringBlockNumbers"
	blockKeyPrefix                = ""
	lastCommittedBlockID          = "lastCommittedBlock"
	lastCommittedBlockData        = "data"
)

type couchDB interface {
	ExistsWithRetry() (bool, error)
	IndexDesignDocExistsWithRetry(designDocs ...string) (bool, error)
	CreateNewIndexWithRetry(indexdefinition string, designDoc string) error
	ReadDoc(id string) (*couchdb.CouchDoc, string, error)
	BatchUpdateDocuments(documents []*couchdb.CouchDoc) ([]*couchdb.BatchUpdateResponse, error)
	QueryDocuments(query string) ([]*couchdb.QueryResult, string, error)
	DeleteDoc(id, rev string) error
}

type blockPvtDataResponse struct {
	ID              string            `json:"_id"`
	Rev             string            `json:"_rev"`
	Data            map[string][]byte `json:"data"`
	Expiry          map[string][]byte `json:"expiry"`
	Deleted         bool              `json:"_deleted"`
	ExpiringBlkNums []string          `json:"expiringBlkNums"`
}

type lastCommittedBlockResponse struct {
	ID   string `json:"_id"`
	Rev  string `json:"_rev"`
	Data string `json:"data"`
}

const expiringBlockNumbersIndexDef = `
	{
		"index": {
			"fields": ["` + expiringBlkNumsField + `"]
		},
		"name": "` + expiringBlockNumbersIndexName + `",
		"ddoc": "` + expiringBlockNumbersIndexDoc + `",
		"type": "json"
	}`

type jsonValue map[string]interface{}

func (v jsonValue) toBytes() ([]byte, error) {
	return json.Marshal(v)
}

func createPvtDataCouchDoc(storeEntries *common.StoreEntries, blockNumber uint64, rev string) (*couchdb.CouchDoc, error) {
	if len(storeEntries.DataEntries) <= 0 && len(storeEntries.ExpiryEntries) <= 0 {
		return nil, nil
	}
	jsonMap := make(jsonValue)
	jsonMap[idField] = blockNumberToKey(blockNumber)

	if rev != "" {
		jsonMap[revField] = rev
	}

	dataEntriesJSON, err := dataEntriesToJSONValue(storeEntries.DataEntries)
	if err != nil {
		return nil, err
	}
	jsonMap[dataField] = dataEntriesJSON

	expiryEntriesJSON, expiringBlkNums, err := expiryEntriesToJSONValue(storeEntries.ExpiryEntries)
	if err != nil {
		return nil, err
	}
	jsonMap[expiryField] = expiryEntriesJSON

	jsonMap[expiringBlkNumsField] = expiringBlkNums

	jsonBytes, err := jsonMap.toBytes()
	if err != nil {
		return nil, err
	}

	couchDoc := couchdb.CouchDoc{JSONValue: jsonBytes}

	return &couchDoc, nil

}

func createLastCommittedBlockDoc(committingBlockNum uint64, rev string) (*couchdb.CouchDoc, error) {
	jsonMap := make(jsonValue)
	jsonMap[idField] = lastCommittedBlockID
	if rev != "" {
		jsonMap[revField] = rev
	}
	jsonMap[lastCommittedBlockData] = strconv.FormatUint(committingBlockNum, 10)
	jsonBytes, err := jsonMap.toBytes()
	if err != nil {
		return nil, err
	}

	couchDoc := couchdb.CouchDoc{JSONValue: jsonBytes}

	return &couchDoc, nil

}

func lookupLastBlock(db couchDB) (uint64, string, error) {
	v, _, err := db.ReadDoc(lastCommittedBlockID)
	if err != nil {
		return 0, "", err
	}
	if v != nil {
		var lastBlockResponse lastCommittedBlockResponse
		if err = json.Unmarshal(v.JSONValue, &lastBlockResponse); err != nil {
			return 0, "", errors.Wrapf(err, "Unmarshal lastBlockResponse failed")
		}
		lastBlockNum, err := strconv.ParseInt(lastBlockResponse.Data, 10, 64)
		if err != nil {
			return 0, "", errors.Wrapf(err, "strconv.ParseInt lastBlockResponse.Data failed")
		}
		return uint64(lastBlockNum), lastBlockResponse.Rev, nil
	}
	return 0, "", nil
}

func dataEntriesToJSONValue(dataEntries []*common.DataEntry) (jsonValue, error) {
	data := make(jsonValue)

	for _, dataEntry := range dataEntries {
		keyBytes := common.EncodeDataKey(dataEntry.Key)
		valBytes, err := common.EncodeDataValue(dataEntry.Value)
		if err != nil {
			return nil, err
		}

		keyBytesHex := hex.EncodeToString(keyBytes)
		data[keyBytesHex] = valBytes
	}

	return data, nil
}

func expiryEntriesToJSONValue(expiryEntries []*common.ExpiryEntry) (jsonValue, []string, error) {
	data := make(jsonValue)
	expiringBlkNums := make([]string, 0)

	for _, expEntry := range expiryEntries {
		keyBytes := common.EncodeExpiryKey(expEntry.Key)
		valBytes, err := common.EncodeExpiryValue(expEntry.Value)
		if err != nil {
			return nil, nil, err
		}
		expiringBlkNums = append(expiringBlkNums, blockNumberToKey(expEntry.Key.ExpiringBlk))
		keyBytesHex := hex.EncodeToString(keyBytes)
		data[keyBytesHex] = valBytes
	}

	return data, expiringBlkNums, nil
}

func createPvtDataCouchDB(db couchDB) error {
	err := db.CreateNewIndexWithRetry(expiringBlockNumbersIndexDef, expiringBlockNumbersIndexDoc)
	if err != nil {
		return errors.WithMessage(err, "creation of purge block number index failed")
	}
	return err
}

func getPvtDataCouchInstance(db couchDB, dbName string) error {
	dbExists, err := db.ExistsWithRetry()
	if err != nil {
		return err
	}
	if !dbExists {
		return errors.Errorf("DB not found: [%s]", dbName)
	}

	indexExists, err := db.IndexDesignDocExistsWithRetry(expiringBlockNumbersIndexDoc)
	if err != nil {
		return err
	}
	if !indexExists {
		return errors.Errorf("DB index not found: [%s]", dbName)
	}
	return nil
}

func retrieveBlockPvtData(db couchDB, id string) (*blockPvtDataResponse, error) {
	doc, _, err := db.ReadDoc(id)
	if err != nil {
		return nil, err
	}

	if doc == nil {
		return nil, NewErrNotFoundInIndex()
	}

	var blockPvtData blockPvtDataResponse
	err = json.Unmarshal(doc.JSONValue, &blockPvtData)
	if err != nil {
		return nil, errors.Wrapf(err, "result from DB is not JSON encoded")
	}

	return &blockPvtData, nil
}

func retrieveBlockExpiryData(db couchDB, id string) ([]*blockPvtDataResponse, error) {
	const queryFmt = `
	{
		"selector": {
			"` + expiringBlkNumsField + `": {
				"$elemMatch": {
					"$lte": "%s"
				}
			}
		},
		"use_index": ["_design/` + expiringBlockNumbersIndexDoc + `", "` + expiringBlockNumbersIndexName + `"]
	}`

	results, _, err := db.QueryDocuments(fmt.Sprintf(queryFmt, id))
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, nil
	}

	var responses []*blockPvtDataResponse
	for _, result := range results {
		var blockPvtData blockPvtDataResponse
		err = json.Unmarshal(result.Value, &blockPvtData)
		if err != nil {
			return nil, errors.Wrapf(err, "result from DB is not JSON encoded")
		}
		responses = append(responses, &blockPvtData)
	}

	return responses, nil
}

func blockNumberToKey(blockNum uint64) string {
	return fmt.Sprintf("%064s", blockKeyPrefix+strconv.FormatUint(blockNum, 10))
}

// NotFoundInIndexErr is used to indicate missing entry in the index
type NotFoundInIndexErr struct {
}

// NewErrNotFoundInIndex creates an missing entry in the index error
func NewErrNotFoundInIndex() *NotFoundInIndexErr {
	return &NotFoundInIndexErr{}
}

func (err *NotFoundInIndexErr) Error() string {
	return "Entry not found in index"
}
