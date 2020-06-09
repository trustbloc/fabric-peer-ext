/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package handler

import (
	"testing"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/handler/mocks"
)

func TestPreEndorsedHandler(t *testing.T) {
	resp := &channel.Response{}
	reqCtx := &invoke.RequestContext{
		Request:  invoke.Request{},
		Opts:     invoke.Opts{},
		Response: invoke.Response{},
	}

	clientCtx := &invoke.ClientContext{}

	t.Run("No next handler", func(t *testing.T) {
		h := NewPreEndorsedHandler(resp)
		require.NotNil(t, h)

		h.Handle(reqCtx, clientCtx)
	})

	t.Run("With next handler", func(t *testing.T) {
		h := NewPreEndorsedHandler(resp, &mocks.InvokeHandler{})
		require.NotNil(t, h)

		h.Handle(reqCtx, clientCtx)
	})
}
