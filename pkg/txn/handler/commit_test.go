/*
   Copyright SecureKey Technologies Inc.
   This file contains software code that is the intellectual property of SecureKey.
   SecureKey reserves all rights in the code and you may not use it without
	 written permission from SecureKey.
*/

package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	sdkmocks "github.com/hyperledger/fabric-sdk-go/pkg/fab/mocks"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/handler/mocks"
)

//go:generate counterfeiter -o ./mocks/transactor.gen.go -fake-name Transactor github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab.Transactor

func TestCommitHandler(t *testing.T) {
	t.Run("Async - no next handler", func(t *testing.T) {
		reqCtx := &invoke.RequestContext{
			Request:  invoke.Request{},
			Opts:     invoke.Opts{},
			Response: invoke.Response{},
			Ctx:      context.Background(),
		}

		clientCtx := &invoke.ClientContext{
			Transactor:   &mocks.Transactor{},
			EventService: sdkmocks.NewMockEventService(),
		}

		h := NewCommitHandler(true)
		require.NotNil(t, h)

		h.Handle(reqCtx, clientCtx)
		require.NoError(t, reqCtx.Error)
	})

	t.Run("Async - with next handler", func(t *testing.T) {
		reqCtx := &invoke.RequestContext{
			Request:  invoke.Request{},
			Opts:     invoke.Opts{},
			Response: invoke.Response{},
			Ctx:      context.Background(),
		}

		clientCtx := &invoke.ClientContext{
			Transactor:   &mocks.Transactor{},
			EventService: sdkmocks.NewMockEventService(),
		}

		h := NewCommitHandler(true, &mocks.InvokeHandler{})
		require.NotNil(t, h)

		h.Handle(reqCtx, clientCtx)
		require.NoError(t, reqCtx.Error)
	})

	t.Run("Sync - no next handler", func(t *testing.T) {
		reqCtx := &invoke.RequestContext{
			Request:  invoke.Request{},
			Opts:     invoke.Opts{},
			Response: invoke.Response{},
			Ctx:      context.Background(),
		}

		clientCtx := &invoke.ClientContext{
			Transactor:   &mocks.Transactor{},
			EventService: sdkmocks.NewMockEventService(),
		}

		h := NewCommitHandler(false)
		require.NotNil(t, h)

		h.Handle(reqCtx, clientCtx)
		require.NoError(t, reqCtx.Error)
	})

	t.Run("Sync - with next handler", func(t *testing.T) {
		reqCtx := &invoke.RequestContext{
			Request:  invoke.Request{},
			Opts:     invoke.Opts{},
			Response: invoke.Response{},
			Ctx:      context.Background(),
		}

		clientCtx := &invoke.ClientContext{
			Transactor:   &mocks.Transactor{},
			EventService: sdkmocks.NewMockEventService(),
		}

		h := NewCommitHandler(false, &mocks.InvokeHandler{})
		require.NotNil(t, h)

		h.Handle(reqCtx, clientCtx)
		require.NoError(t, reqCtx.Error)
	})

	t.Run("CreateTransaction error", func(t *testing.T) {
		reqCtx := &invoke.RequestContext{
			Request:  invoke.Request{},
			Opts:     invoke.Opts{},
			Response: invoke.Response{},
			Ctx:      context.Background(),
		}

		errExpected := errors.New("injected transactor error")

		transactor := &mocks.Transactor{}
		transactor.CreateTransactionReturns(nil, errExpected)

		clientCtx := &invoke.ClientContext{
			Transactor:   transactor,
			EventService: sdkmocks.NewMockEventService(),
		}

		h := NewCommitHandler(false, &mocks.InvokeHandler{})
		require.NotNil(t, h)

		h.Handle(reqCtx, clientCtx)
		require.Error(t, reqCtx.Error)
		require.Contains(t, reqCtx.Error.Error(), errExpected.Error())
	})

	t.Run("SendTransaction error", func(t *testing.T) {
		reqCtx := &invoke.RequestContext{
			Request:  invoke.Request{},
			Opts:     invoke.Opts{},
			Response: invoke.Response{},
			Ctx:      context.Background(),
		}

		errExpected := errors.New("injected transactor error")

		transactor := &mocks.Transactor{}
		transactor.SendTransactionReturns(nil, errExpected)

		clientCtx := &invoke.ClientContext{
			Transactor:   transactor,
			EventService: sdkmocks.NewMockEventService(),
		}

		h := NewCommitHandler(false, &mocks.InvokeHandler{})
		require.NotNil(t, h)

		h.Handle(reqCtx, clientCtx)
		require.Error(t, reqCtx.Error)
		require.Contains(t, reqCtx.Error.Error(), errExpected.Error())
	})

	t.Run("TxValidation code error", func(t *testing.T) {
		reqCtx := &invoke.RequestContext{
			Request:  invoke.Request{},
			Opts:     invoke.Opts{},
			Response: invoke.Response{},
			Ctx:      context.Background(),
		}

		eventService := sdkmocks.NewMockEventService()
		eventService.TxValidationCode = pb.TxValidationCode_MVCC_READ_CONFLICT

		clientCtx := &invoke.ClientContext{
			Transactor:   &mocks.Transactor{},
			EventService: eventService,
		}

		h := NewCommitHandler(false, &mocks.InvokeHandler{})
		require.NotNil(t, h)

		h.Handle(reqCtx, clientCtx)
		require.Error(t, reqCtx.Error)
		require.Contains(t, reqCtx.Error.Error(), "MVCC_READ_CONFLICT")
	})

	t.Run("Event timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()

		reqCtx := &invoke.RequestContext{
			Request:  invoke.Request{},
			Opts:     invoke.Opts{},
			Response: invoke.Response{},
			Ctx:      ctx,
		}

		eventService := sdkmocks.NewMockEventService()
		eventService.Timeout = true

		clientCtx := &invoke.ClientContext{
			Transactor:   &mocks.Transactor{},
			EventService: eventService,
		}

		h := NewCommitHandler(false, &mocks.InvokeHandler{})
		require.NotNil(t, h)

		h.Handle(reqCtx, clientCtx)
		require.Error(t, reqCtx.Error)
		require.Contains(t, reqCtx.Error.Error(), "TIMEOUT")
	})
}
