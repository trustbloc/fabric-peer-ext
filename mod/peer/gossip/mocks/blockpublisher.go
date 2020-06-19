/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	cb "github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric/extensions/gossip/api"
)

// BlockPublisher is a mock block publisher
type BlockPublisher struct {
}

// NewBlockPublisher returns a new mock block publisher
func NewBlockPublisher() *BlockPublisher {
	return &BlockPublisher{}
}

// AddCCUpgradeHandler adds a handler for chaincode upgrades
func (m *BlockPublisher) AddCCUpgradeHandler(handler api.ChaincodeUpgradeHandler) {
	// Not implemented
}

// AddConfigUpdateHandler adds a handler for config updates
func (m *BlockPublisher) AddConfigUpdateHandler(handler api.ConfigUpdateHandler) {
	// Not implemented
}

// AddWriteHandler adds a handler for KV writes
func (m *BlockPublisher) AddWriteHandler(handler api.WriteHandler) {
	// Not implemented
}

// AddLSCCWriteHandler adds a handler for KV writes
func (m *BlockPublisher) AddLSCCWriteHandler(handler api.LSCCWriteHandler) {
	// Not implemented
}

// AddReadHandler adds a handler for KV reads
func (m *BlockPublisher) AddReadHandler(handler api.ReadHandler) {
	// Not implemented
}

// AddCollHashWriteHandler adds a handler for KV collection hash writes
func (m *BlockPublisher) AddCollHashWriteHandler(handler api.CollHashWriteHandler) {
	// Not implemented
}

// AddCollHashReadHandler adds a handler for KV collection hash reads
func (m *BlockPublisher) AddCollHashReadHandler(handler api.CollHashReadHandler) {
	// Not implemented
}

// AddCCEventHandler adds a handler for chaincode events
func (m *BlockPublisher) AddCCEventHandler(handler api.ChaincodeEventHandler) {
	// Not implemented
}

// Publish traverses the block and invokes all applicable handlers
func (m *BlockPublisher) Publish(block *cb.Block) {
	// Not implemented
}

// LedgerHeight returns ledger height based on last block published
func (m *BlockPublisher) LedgerHeight() uint64 {
	// Not implemented
	return 0
}
