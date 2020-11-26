/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package statedb

import (
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	"github.com/hyperledger/fabric/extensions/gossip/blockpublisher"
	"github.com/hyperledger/fabric/extensions/roles"

	extstatedb "github.com/trustbloc/fabric-peer-ext/pkg/statedb"
)

//AddCCUpgradeHandler adds chaincode upgrade handler to blockpublisher
func AddCCUpgradeHandler(chainName string, handler gossipapi.ChaincodeUpgradeHandler) {
	if !roles.IsCommitter() {
		blockpublisher.ForChannel(chainName).AddCCUpgradeHandler(handler)
	}
}

// Register registers a state database for a given channel
func Register(channelID string, db statedb.VersionedDB) {
	extstatedb.GetProvider().Register(channelID, &stateDB{db: db})
}

type stateDB struct {
	db statedb.VersionedDB
}

func (s *stateDB) GetState(namespace string, key string) ([]byte, error) {
	value, err := s.db.GetState(namespace, key)
	if err != nil {
		return nil, err
	}

	if value == nil {
		return nil, nil
	}

	return value.Value, nil
}

func (s *stateDB) GetStateMultipleKeys(namespace string, keys []string) ([][]byte, error) {
	values, err := s.db.GetStateMultipleKeys(namespace, keys)
	if err != nil {
		return nil, err
	}

	if values == nil {
		return nil, nil
	}

	results := make([][]byte, len(values))
	for i, value := range values {
		if value != nil {
			results[i] = value.Value
		}
	}

	return results, nil
}

func (s *stateDB) GetStateRangeScanIterator(namespace string, startKey string, endKey string) (statedb.ResultsIterator, error) {
	return s.db.GetStateRangeScanIterator(namespace, startKey, endKey)
}

func (s *stateDB) GetStateRangeScanIteratorWithPagination(namespace string, startKey string, endKey string, pageSize int32) (statedb.QueryResultsIterator, error) {
	return s.db.GetStateRangeScanIteratorWithPagination(namespace, startKey, endKey, pageSize)
}

func (s *stateDB) ExecuteQuery(namespace, query string) (statedb.ResultsIterator, error) {
	return s.db.ExecuteQuery(namespace, query)
}

func (s *stateDB) ExecuteQueryWithPagination(namespace, query, bookmark string, pageSize int32) (statedb.QueryResultsIterator, error) {
	return s.db.ExecuteQueryWithPagination(namespace, query, bookmark, pageSize)
}

func (s *stateDB) BytesKeySupported() bool {
	return s.db.BytesKeySupported()
}

func (s *stateDB) UpdateCache(blockNum uint64, updates []byte) error {
	return s.db.UpdateCache(blockNum, updates)
}
