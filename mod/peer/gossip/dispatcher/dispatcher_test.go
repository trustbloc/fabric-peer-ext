/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dispatcher

import (
	"testing"

	gproto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

func TestProvider(t *testing.T) {
	const channelID = "testchannel"

	dispatcher := New(
		channelID,
		&mocks.DataStore{},
		mocks.NewMockGossipAdapter(),
		&mocks.Ledger{QueryExecutor: mocks.NewQueryExecutor(nil)},
		mocks.NewBlockPublisher(),
	)

	var response *gproto.GossipMessage
	msg := &mocks.MockReceivedMessage{
		Message: mocks.NewDataMsg(channelID),
		RespondTo: func(msg *gproto.GossipMessage) {
			response = msg
		},
	}
	assert.False(t, dispatcher.Dispatch(msg))
	require.Nil(t, response)
}
