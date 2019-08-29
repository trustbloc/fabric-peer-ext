/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package state

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/compositekey"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/state/api"
)

var logger = flogging.MustGetLogger("ledgerconfig")

// QERetrieverProvider is a RetrieverProvider using a QueryExecutor
type QERetrieverProvider struct {
	channelID  string
	qeProvider qeProvider
}

type qeProvider interface {
	GetQueryExecutorForLedger(cid string) (ledger.QueryExecutor, error)
}

// NewQERetrieverProvider returns a new state qeRetriever provider
func NewQERetrieverProvider(channelID string, qeProvider qeProvider) *QERetrieverProvider {
	return &QERetrieverProvider{
		channelID:  channelID,
		qeProvider: qeProvider,
	}
}

// GetStateRetriever returns the state qeRetriever
func (p *QERetrieverProvider) GetStateRetriever() (api.StateRetriever, error) {
	qe, err := p.qeProvider.GetQueryExecutorForLedger(p.channelID)
	if err != nil {
		return nil, err
	}
	return &qeRetriever{
		qe: qe,
	}, nil
}

// qeRetriever implements the StateRetriever interface
type qeRetriever struct {
	qe ledger.QueryExecutor
}

// GetState returns the value for the given key
func (s *qeRetriever) GetState(namespace, key string) ([]byte, error) {
	return s.qe.GetState(namespace, key)
}

// GetStateByPartialCompositeKey returns an iterator for the given index and attributes
func (s *qeRetriever) GetStateByPartialCompositeKey(namespace, objectType string, attributes []string) (api.ResultsIterator, error) {
	startKey, endKey := compositekey.CreateRangeKeysForPartialCompositeKey(objectType, attributes)
	logger.Debugf("Namespace [%s], StartKey [%s], EndKey [%s]", namespace, startKey, endKey)
	it, err := s.qe.GetStateRangeScanIterator(namespace, startKey, endKey)
	if err != nil {
		return nil, err
	}
	return NewKVResultsIter(it), nil
}

// Done releases resources occupied by the StateRetriever
func (s *qeRetriever) Done() {
	s.qe.Done()
}
