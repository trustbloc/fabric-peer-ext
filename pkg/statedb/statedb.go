/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package statedb

import (
	"sync"

	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
)

var provider = newProvider()

// StateDB declares functions that allow for retrieving and querying from a state database
type StateDB interface {
	// GetState gets the value for given namespace and key. For a chaincode, the namespace corresponds to the chaincodeId
	GetState(namespace string, key string) ([]byte, error)
	// GetStateMultipleKeys gets the values for multiple keys in a single call
	GetStateMultipleKeys(namespace string, keys []string) ([][]byte, error)
	// GetStateRangeScanIterator returns an iterator that contains all the key-values between given key ranges.
	// startKey is inclusive
	// endKey is exclusive
	// The returned ResultsIterator contains results of type *VersionedKV
	GetStateRangeScanIterator(namespace string, startKey string, endKey string) (statedb.ResultsIterator, error)
	// GetStateRangeScanIteratorWithPagination returns an iterator that contains all the key-values between given key ranges.
	// startKey is inclusive
	// endKey is exclusive
	// pageSize parameter limits the number of returned results
	// The returned ResultsIterator contains results of type *VersionedKV
	GetStateRangeScanIteratorWithPagination(namespace string, startKey string, endKey string, pageSize int32) (statedb.QueryResultsIterator, error)
	// ExecuteQuery executes the given query and returns an iterator that contains results of type *VersionedKV.
	ExecuteQuery(namespace, query string) (statedb.ResultsIterator, error)
	// ExecuteQueryWithPagination executes the given query and
	// returns an iterator that contains results of type *VersionedKV.
	// The bookmark and page size parameters are associated with the pagination query.
	ExecuteQueryWithPagination(namespace, query, bookmark string, pageSize int32) (statedb.QueryResultsIterator, error)
	// BytesKeySupported returns true if the implementation (underlying db) supports the any bytes to be used as key.
	// For instance, leveldb supports any bytes for the key while the couchdb supports only valid utf-8 string
	BytesKeySupported() bool
	// UpdateCache updates the state cache with the given updates. The format of the updates depends on the database implementation.
	UpdateCache(blockNum uint64, updates []byte) error
}

// Provider is a state database Provider
type Provider struct {
	channelDBs map[string]StateDB
	mutex      sync.RWMutex
}

// GetProvider returns the state database provider
func GetProvider() *Provider {
	return provider
}

func newProvider() *Provider {
	return &Provider{
		channelDBs: make(map[string]StateDB),
	}
}

// StateDBForChannel returns the state database for the given channel
func (p *Provider) StateDBForChannel(channelID string) StateDB {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.channelDBs[channelID]
}

// Register registers a state database for the given channel
func (p *Provider) Register(channelID string, db StateDB) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.channelDBs[channelID] = db
}
