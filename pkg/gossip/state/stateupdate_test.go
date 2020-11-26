/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package state

import (
	"errors"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/common/blockvisitor"
	"github.com/trustbloc/fabric-peer-ext/pkg/gossip/blockpublisher"
	statemocks "github.com/trustbloc/fabric-peer-ext/pkg/gossip/state/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

//go:generate counterfeiter -o ./mocks/ccevtmgrprovider.gen.go --fake-name CCEventMgrProvider . ccEventMgrProvider
//go:generate counterfeiter -o ./mocks/ccevtmgr.gen.go --fake-name CCEventMgr github.com/hyperledger/fabric/extensions/chaincode/api.EventMgr
//go:generate counterfeiter -o ./mocks/appdatahandlerregistry.gen.go --fake-name AppDataHandlerRegistry . appDataHandlerRegistry

const (
	channel1 = "channel1"
	txID1    = "tx1"
	ccID1    = "cc1"
)

func TestNewUpdateHandler(t *testing.T) {
	bpp := blockpublisher.NewProvider()

	b := mocks.NewBlockBuilder(channel1, 1000)

	ccData := &ccprovider.ChaincodeData{
		Name: ccID1,
	}
	ccDataBytes, err := proto.Marshal(ccData)
	require.NoError(t, err)

	b.Transaction(txID1, pb.TxValidationCode_VALID).
		ChaincodeAction(blockvisitor.LsccID).
		Write(ccID1, ccDataBytes)

	t.Run("Committer", func(t *testing.T) {
		ccEvtMgr := &statemocks.CCEventMgr{}
		ccEvtMgrProvider := &statemocks.CCEventMgrProvider{}
		ccEvtMgrProvider.GetMgrReturns(ccEvtMgr)
		handlerRegistry := &statemocks.AppDataHandlerRegistry{}

		h := NewUpdateHandler(&providers{BPProvider: bpp, MgrProvider: ccEvtMgrProvider, HandlerRegistry: handlerRegistry})
		require.NotPanics(t, func() { h.ChannelJoined(channel1) })

		bpp.ForChannel(channel1).Publish(b.Build(), nil)
		time.Sleep(200 * time.Millisecond)
	})

	t.Run("Endorser", func(t *testing.T) {
		reset := initRoles(roles.EndorserRole)
		defer reset()

		ccEvtMgr := &statemocks.CCEventMgr{}
		ccEvtMgrProvider := &statemocks.CCEventMgrProvider{}
		ccEvtMgrProvider.GetMgrReturns(ccEvtMgr)
		handlerRegistry := &statemocks.AppDataHandlerRegistry{}

		h := NewUpdateHandler(&providers{BPProvider: bpp, MgrProvider: ccEvtMgrProvider, HandlerRegistry: handlerRegistry})
		require.NotPanics(t, func() { h.ChannelJoined(channel1) })

		bpp.ForChannel(channel1).Publish(b.Build(), nil)
		time.Sleep(200 * time.Millisecond)
	})

	t.Run("Handler error", func(t *testing.T) {
		reset := initRoles(roles.EndorserRole)
		defer reset()

		ccEvtMgr := &statemocks.CCEventMgr{}
		ccEvtMgrProvider := &statemocks.CCEventMgrProvider{}
		ccEvtMgrProvider.GetMgrReturns(ccEvtMgr)
		handlerRegistry := &statemocks.AppDataHandlerRegistry{}

		h := NewUpdateHandler(&providers{BPProvider: bpp, MgrProvider: ccEvtMgrProvider, HandlerRegistry: handlerRegistry})
		require.NotPanics(t, func() { h.ChannelJoined(channel1) })

		errExpected := errors.New("handler error")
		ccEvtMgr.HandleChaincodeDeployReturns(errExpected)

		bpp.ForChannel(channel1).Publish(b.Build(), nil)
		time.Sleep(200 * time.Millisecond)
	})
}

func initRoles(r ...roles.Role) (reset func()) {
	rolesValue := make(map[roles.Role]struct{})
	for _, r := range r {
		rolesValue[r] = struct{}{}
	}
	roles.SetRoles(rolesValue)
	return func() { roles.SetRoles(nil) }
}
