/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package simplequeryexecutor

import (
	"encoding/base64"
	"strings"

	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	"github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/util"

	extstatedb "github.com/trustbloc/fabric-peer-ext/pkg/statedb"
)

// QueryExecutor is a simple query executor that reads data directly from the provided state database
// without acquiring a ledger read lock.
type QueryExecutor struct {
	stateDB extstatedb.StateDB
}

// New returns a new query executor that reads data directly from the provided state database
func New(db extstatedb.StateDB) *QueryExecutor {
	return &QueryExecutor{stateDB: db}
}

// GetState gets the value for given namespace and key. For a chaincode, the namespace corresponds to the chaincodeId
func (sqe *QueryExecutor) GetState(ns string, key string) ([]byte, error) {
	return sqe.stateDB.GetState(ns, key)
}

// GetStateRangeScanIterator returns an iterator that contains all the key-values between given key ranges.
func (sqe *QueryExecutor) GetStateRangeScanIterator(ns string, startKey string, endKey string) (ledger.ResultsIterator, error) {
	dbItr, err := sqe.stateDB.GetStateRangeScanIterator(ns, startKey, endKey)
	if err != nil {
		return nil, err
	}

	return &resultsItr{ns: ns, dbItr: dbItr}, nil
}

// GetPrivateDataHash gets the hash of the value of a private data item identified by a tuple <namespace, collection, key>
func (sqe *QueryExecutor) GetPrivateDataHash(ns, coll, key string) ([]byte, error) {
	keyHash := util.ComputeStringHash(key)
	keyHashStr := string(keyHash)

	if !sqe.stateDB.BytesKeySupported() {
		keyHashStr = base64.StdEncoding.EncodeToString(keyHash)
	}

	return sqe.stateDB.GetState(hashedDataNs(ns, coll), keyHashStr)
}

type resultsItr struct {
	ns    string
	dbItr statedb.ResultsIterator
}

// Next implements method in interface ledger.ResultsIterator
func (itr *resultsItr) Next() (ledger.QueryResult, error) {
	queryResult, err := itr.dbItr.Next()
	if err != nil {
		return nil, err
	}

	if queryResult == nil {
		return nil, nil
	}

	versionedKV := queryResult.(*statedb.VersionedKV)
	return &queryresult.KV{Namespace: versionedKV.Namespace, Key: versionedKV.Key, Value: versionedKV.Value}, nil
}

// Close implements method in interface ledger.ResultsIterator
func (itr *resultsItr) Close() {
	itr.dbItr.Close()
}

func hashedDataNs(namespace, collection string) string {
	return strings.Join([]string{namespace, collection}, "$$h")
}
