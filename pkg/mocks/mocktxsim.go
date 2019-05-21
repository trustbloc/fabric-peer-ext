/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	ledgermocks "github.com/hyperledger/fabric/common/mocks/ledger"
	"github.com/hyperledger/fabric/core/ledger"
)

// TxSimulator implements a mock transaction simulator
type TxSimulator struct {
	ledgermocks.MockQueryExecutor
	SimulationResults *ledger.TxSimulationResults
	Error             error
	SimError          error
}

// SetState is not currently implemented and will panic if called
func (m *TxSimulator) SetState(namespace string, key string, value []byte) error {
	panic("not implemented")
}

// DeleteState is not currently implemented and will panic if called
func (m *TxSimulator) DeleteState(namespace string, key string) error {
	panic("not implemented")
}

// SetStateMultipleKeys is not currently implemented and will panic if called
func (m *TxSimulator) SetStateMultipleKeys(namespace string, kvs map[string][]byte) error {
	panic("not implemented")
}

// ExecuteUpdate is not currently implemented and will panic if called
func (m *TxSimulator) ExecuteUpdate(query string) error {
	panic("not implemented")
}

// GetTxSimulationResults returns the mock simulation results
func (m *TxSimulator) GetTxSimulationResults() (*ledger.TxSimulationResults, error) {
	return m.SimulationResults, m.SimError
}

// DeletePrivateData is not currently implemented and will panic if called
func (m *TxSimulator) DeletePrivateData(namespace, collection, key string) error {
	panic("not implemented")
}

// SetPrivateData is not currently implemented and will panic if called
func (m *TxSimulator) SetPrivateData(namespace, collection, key string, value []byte) error {
	panic("not implemented")
}

// SetPrivateDataMultipleKeys currently does nothing except return the mock error (if any)
func (m *TxSimulator) SetPrivateDataMultipleKeys(namespace, collection string, kvs map[string][]byte) error {
	return m.Error
}

// SetStateMetadata is not currently implemented and will panic if called
func (m *TxSimulator) SetStateMetadata(namespace, key string, metadata map[string][]byte) error {
	panic("not implemented")
}

// DeleteStateMetadata is not currently implemented and will panic if called
func (m *TxSimulator) DeleteStateMetadata(namespace, key string) error {
	panic("not implemented")
}

// SetPrivateDataMetadata is not currently implemented and will panic if called
func (m *TxSimulator) SetPrivateDataMetadata(namespace, collection, key string, metadata map[string][]byte) error {
	panic("not implemented")
}

// DeletePrivateDataMetadata is not currently implemented and will panic if called
func (m *TxSimulator) DeletePrivateDataMetadata(namespace, collection, key string) error {
	panic("not implemented")
}
