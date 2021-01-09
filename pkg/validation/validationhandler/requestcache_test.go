/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validationhandler

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

const (
	channelID = "channel1"
)

func TestRequestCache(t *testing.T) {
	rc := newRequestCache(channelID)
	require.NotNil(t, rc)
	require.Equal(t, rc.Size(), 0)

	block1 := mocks.NewBlockBuilder(channelID, 1000).Build()
	block2 := mocks.NewBlockBuilder(channelID, 1001).Build()
	block3 := mocks.NewBlockBuilder(channelID, 1001).Build()

	require.Nil(t, rc.Remove(block1.Header.Number))

	responder1 := &mockResponder{}
	responder2 := &mockResponder{}
	responder3 := &mockResponder{}

	rc.Add(block1, responder1)
	rc.Add(block2, responder2)
	rc.Add(block3, responder3)

	require.Equal(t, rc.Size(), 3)

	req := rc.Remove(block2.Header.Number)
	require.NotNil(t, req)
	require.Equal(t, block2.Header.Number, req.block.Header.Number)
	require.Equal(t, responder2, req.responder)
	require.Equal(t, rc.Size(), 1)

	req = rc.Remove(block3.Header.Number)
	require.NotNil(t, req)
	require.Equal(t, block3.Header.Number, req.block.Header.Number)
	require.Equal(t, responder3, req.responder)
	require.Equal(t, rc.Size(), 0)
}

type mockResponder struct {
	data []byte
}

func (m *mockResponder) Respond(data []byte) {
	m.data = data
}
