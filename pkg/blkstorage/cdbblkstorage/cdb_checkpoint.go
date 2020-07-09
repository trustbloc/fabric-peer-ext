/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cdbblkstorage

import (
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/util/retry"
	couchdb "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"
	"github.com/pkg/errors"
	viper "github.com/spf13/viper2015"
)

const blkMgrInfoKey = "blkMgrInfo"

type checkpoint struct {
	db         couchDB
	maxRetries int
}

// checkpointInfo
type checkpointInfo struct {
	isChainEmpty    bool
	lastBlockNumber uint64
	currentHash     []byte
}

type couchDB interface {
	ReadDoc(id string) (*couchdb.CouchDoc, string, error)
	SaveDoc(id string, rev string, couchDoc *couchdb.CouchDoc) (string, error)
}

func newCheckpoint(db couchDB) *checkpoint {
	return &checkpoint{
		db:         db,
		maxRetries: viper.GetInt("ledger.state.couchDBConfig.maxRetries"),
	}
}

func (cp *checkpoint) getCheckpointInfo() *checkpointInfo {
	cpInfo, err := cp.loadCurrentInfo()
	if err != nil {
		panic(fmt.Sprintf("Could not get block file info for current block file from db: %s", err))
	}
	if cpInfo == nil {
		cpInfo = &checkpointInfo{
			isChainEmpty:    true,
			lastBlockNumber: 0}
	}
	return cpInfo
}

//Get the current checkpoint information that is stored in the database
func (cp *checkpoint) loadCurrentInfo() (*checkpointInfo, error) {
	doc, err := cp.readCheckpointInfo()
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("retrieval of checkpointInfo from couchDB failed [%s]", blkMgrInfoKey))
	}
	if doc == nil {
		return nil, nil
	}
	checkpointInfo, err := couchDocToCheckpointInfo(doc)
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("unmarshal of checkpointInfo from couchDB failed [%s]", blkMgrInfoKey))
	}
	logger.Debugf("loaded checkpointInfo:%s", checkpointInfo)
	return checkpointInfo, nil
}

func (cp *checkpoint) readCheckpointInfo() (*couchdb.CouchDoc, error) {
	doc, err := retry.Invoke(
		func() (interface{}, error) {
			doc, _, err := cp.db.ReadDoc(blkMgrInfoKey)
			return doc, err
		},
		retry.WithMaxAttempts(cp.maxRetries+1),
		retry.WithBeforeRetry(func(err error, attempt int, backoff time.Duration) bool {
			logger.Warnf("Error reading checkpoint info on attempt %d. Will retry in %s: %s", attempt, backoff, err)
			return true
		}),
	)

	if err != nil {
		return nil, err
	}

	return doc.(*couchdb.CouchDoc), nil
}

func (cp *checkpoint) saveCurrentInfo(i *checkpointInfo) error {
	doc, err := checkpointInfoToCouchDoc(i)
	if err != nil {
		return errors.WithMessage(err, "converting checkpointInfo to couchDB document failed")
	}
	_, err = cp.db.SaveDoc(blkMgrInfoKey, "", doc)
	if err != nil {
		return errors.WithMessage(err, "adding checkpointInfo to couchDB failed")
	}
	return nil
}

func (i *checkpointInfo) marshal() ([]byte, error) {
	buffer := proto.NewBuffer([]byte{})
	var err error
	if err = buffer.EncodeVarint(i.lastBlockNumber); err != nil {
		return nil, err
	}

	if err = buffer.EncodeRawBytes(i.currentHash); err != nil {
		return nil, err
	}

	var chainEmptyMarker uint64
	if i.isChainEmpty {
		chainEmptyMarker = 1
	}
	if err = buffer.EncodeVarint(chainEmptyMarker); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (i *checkpointInfo) unmarshal(b []byte) error {
	buffer := proto.NewBuffer(b)
	var chainEmptyMarker uint64
	var err error

	if i.lastBlockNumber, err = buffer.DecodeVarint(); err != nil {
		return err
	}

	if i.currentHash, err = buffer.DecodeRawBytes(false); err != nil {
		return err
	}

	if len(i.currentHash) == 0 {
		i.currentHash = nil
	}

	if chainEmptyMarker, err = buffer.DecodeVarint(); err != nil {
		return err
	}
	i.isChainEmpty = chainEmptyMarker == 1

	return nil
}
