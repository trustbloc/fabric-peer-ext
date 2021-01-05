/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric/core/ledger"
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
)

// MockBlockPublisher is a mock block publisher
type MockBlockPublisher struct {
	HandleUpgrade       gossipapi.ChaincodeUpgradeHandler
	HandleConfigUpdate  gossipapi.ConfigUpdateHandler
	HandleWrite         gossipapi.WriteHandler
	HandleRead          gossipapi.ReadHandler
	HandleCollHashWrite gossipapi.CollHashWriteHandler
	HandleCollHashRead  gossipapi.CollHashReadHandler
	HandleLSCCWrite     gossipapi.LSCCWriteHandler
	HandleCCEvent       gossipapi.ChaincodeEventHandler
	HandleBlock         gossipapi.PublishedBlockHandler
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

// AddCollHashWriteHandler adds a collection hash write handler
func (m *MockBlockPublisher) AddCollHashWriteHandler(handler gossipapi.CollHashWriteHandler) {
	m.HandleCollHashWrite = handler
}

// AddCollHashReadHandler adds a read handler
func (m *MockBlockPublisher) AddCollHashReadHandler(handler gossipapi.CollHashReadHandler) {
	m.HandleCollHashRead = handler
}

// AddLSCCWriteHandler adds a write handler
func (m *MockBlockPublisher) AddLSCCWriteHandler(handler gossipapi.LSCCWriteHandler) {
	m.HandleLSCCWrite = handler
}

// AddCCEventHandler adds a chaincode event handler
func (m *MockBlockPublisher) AddCCEventHandler(handler gossipapi.ChaincodeEventHandler) {
	m.HandleCCEvent = handler
}

// AddBlockHandler adds a block handler
func (m *MockBlockPublisher) AddBlockHandler(handler gossipapi.PublishedBlockHandler) {
	m.HandleBlock = handler
}

// Publish is not implemented and panics if invoked
func (m *MockBlockPublisher) Publish(_ *common.Block, _ ledger.TxPvtDataMap) {
	panic("not implemented")
}

// LedgerHeight is not implemented and panics if invoked
func (m *MockBlockPublisher) LedgerHeight() uint64 {
	panic("not implemented")
}

// MockBlockPublisherProvider is a mock block publisher provider
type MockBlockPublisherProvider struct {
	publisher gossipapi.BlockPublisher
}

// NewBlockPublisherProvider returns a mock block publisher provider
func NewBlockPublisherProvider() *MockBlockPublisherProvider {
	return &MockBlockPublisherProvider{
		publisher: NewBlockPublisher(),
	}
}

// WithBlockPublisher sets the block publisher
func (m *MockBlockPublisherProvider) WithBlockPublisher(publisher gossipapi.BlockPublisher) *MockBlockPublisherProvider {
	m.publisher = publisher
	return m
}

// ForChannel returns the mock block publisher
func (m *MockBlockPublisherProvider) ForChannel(channelID string) gossipapi.BlockPublisher {
	return m.publisher
}
