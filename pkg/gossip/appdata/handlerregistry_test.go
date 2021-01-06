/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package appdata

import (
	"testing"

	gproto "github.com/hyperledger/fabric-protos-go/gossip"
	"github.com/stretchr/testify/require"
)

func TestHandlerRegistry(t *testing.T) {
	r := NewHandlerRegistry()
	require.NotNil(t, r)

	const dataType = "dataType1"

	handler := func(channelID string, request *gproto.AppDataRequest, responder Responder) {}

	err := r.Register(dataType, handler)
	require.NoError(t, err)

	h, ok := r.HandlerForType(dataType)
	require.True(t, ok)
	require.NotNil(t, h)

	require.EqualError(t, r.Register(dataType, handler), "handler for data type [dataType1] already registered")
}
