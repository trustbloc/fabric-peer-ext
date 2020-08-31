/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package state

import (
	"github.com/hyperledger/fabric/common/flogging"

	"github.com/trustbloc/fabric-peer-ext/pkg/common/compositekey"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/state/api"
	extstatedb "github.com/trustbloc/fabric-peer-ext/pkg/statedb"
)

var logger = flogging.MustGetLogger("ledgerconfig")

// QERetrieverProvider is a RetrieverProvider using a QueryExecutor
type QERetrieverProvider struct {
	qeRetriever *qeRetriever
}

// NewQERetrieverProvider returns a new state qeRetriever provider
func NewQERetrieverProvider(stateDB extstatedb.StateDB) *QERetrieverProvider {
	return &QERetrieverProvider{
		qeRetriever: &qeRetriever{stateDB: stateDB},
	}
}

// GetStateRetriever returns the state qeRetriever
func (p *QERetrieverProvider) GetStateRetriever() (api.StateRetriever, error) {
	return p.qeRetriever, nil
}

// qeRetriever implements the StateRetriever interface
type qeRetriever struct {
	stateDB extstatedb.StateDB
}

// GetState returns the value for the given key
func (s *qeRetriever) GetState(namespace, key string) ([]byte, error) {
	return s.stateDB.GetState(namespace, key)
}

// GetStateByPartialCompositeKey returns an iterator for the given index and attributes
func (s *qeRetriever) GetStateByPartialCompositeKey(namespace, objectType string, attributes []string) (api.ResultsIterator, error) {
	startKey, endKey := compositekey.CreateRangeKeysForPartialCompositeKey(objectType, attributes)
	logger.Debugf("Namespace [%s], StartKey [%s], EndKey [%s]", namespace, startKey, endKey)

	it, err := s.stateDB.GetStateRangeScanIterator(namespace, startKey, endKey)
	if err != nil {
		return nil, err
	}

	return NewKVResultsIter(it), nil
}

func (s *qeRetriever) Done() {
	// Nothing to do
}
