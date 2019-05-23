/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	"github.com/hyperledger/fabric/protos/common"
)

// MockBlockPublisher is a mock block publisher
type MockBlockPublisher struct {
	HandleUpgrade      gossipapi.ChaincodeUpgradeHandler
	HandleConfigUpdate gossipapi.ConfigUpdateHandler
	HandleWrite        gossipapi.WriteHandler
	HandleRead         gossipapi.ReadHandler
	HandleCCEvent      gossipapi.ChaincodeEventHandler
}

// NewBlockPublisher returns a mock block publisher
func NewBlockPublisher() *MockBlockPublisher {
	return &MockBlockPublisher{}
}

// AddCCUpgradeHandler adds a chaincode upgrade handler
func (m *MockBlockPublisher) AddCCUpgradeHandler(handler gossipapi.ChaincodeUpgradeHandler) {
	m.HandleUpgrade = handler
}

// AddConfigUpdateHandler adds a config update handler
func (m *MockBlockPublisher) AddConfigUpdateHandler(handler gossipapi.ConfigUpdateHandler) {
	m.HandleConfigUpdate = handler
}

// AddWriteHandler adds a write handler
func (m *MockBlockPublisher) AddWriteHandler(handler gossipapi.WriteHandler) {
	m.HandleWrite = handler
}

// AddReadHandler adds a read handler
func (m *MockBlockPublisher) AddReadHandler(handler gossipapi.ReadHandler) {
	m.HandleRead = handler
}

// AddCCEventHandler adds a chaincode event handler
func (m *MockBlockPublisher) AddCCEventHandler(handler gossipapi.ChaincodeEventHandler) {
	m.HandleCCEvent = handler
}

// Publish is not implemented and panics if invoked
func (m *MockBlockPublisher) Publish(block *common.Block) {
	panic("not implemented")
}
