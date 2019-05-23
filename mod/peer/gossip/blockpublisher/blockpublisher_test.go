/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockpublisher

import (
	"testing"
	"time"

	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/magiconair/properties/assert"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

const (
	channelID = "testchannel"
	txID1     = "tx1"
	ccID1     = "cc1"
	key1      = "key1"
	ccEvent1  = "ccevent1"
)

func TestProvider(t *testing.T) {
	var (
		value1 = []byte("value1")

		v1 = &kvrwset.Version{
			BlockNum: 1000,
			TxNum:    0,
		}
	)

	p := NewProvider()
	require.NotNil(t, p)

	publisher := p.ForChannel(channelID)
	require.NotNil(t, publisher)

	handler := mocks.NewMockBlockHandler()

	publisher.AddWriteHandler(handler.HandleWrite)
	publisher.AddReadHandler(handler.HandleRead)
	publisher.AddCCEventHandler(handler.HandleChaincodeEvent)

	b := mocks.NewBlockBuilder(channelID, 1100)
	defer p.Close()

	b.Transaction(txID1, pb.TxValidationCode_VALID).
		ChaincodeAction(ccID1).
		Write(key1, value1).
		Read(key1, v1).
		ChaincodeEvent(ccEvent1, []byte("ccpayload"))

	publisher.Publish(b.Build())

	// Wait a bit for the events to be published
	time.Sleep(500 * time.Millisecond)

	assert.Equal(t, handler.NumReads(), 1)
	assert.Equal(t, handler.NumWrites(), 1)
	assert.Equal(t, handler.NumCCEvents(), 1)
}
