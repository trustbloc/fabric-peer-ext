/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package statedb

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"

	"github.com/hyperledger/fabric/extensions/gossip/blockpublisher"

	"github.com/golang/protobuf/proto"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

const (
	lsccID       = "lscc"
	upgradeEvent = "upgrade"

	txID = "tx123"
	ccID = "cc123"

	channelID = "sample-chain-name"
)

func TestAddCCUpgradeHandlerAsEndorser(t *testing.T) {
	//make sure roles is endorser not committer
	if roles.IsCommitter() {
		rolesValue := make(map[roles.Role]struct{})
		rolesValue[roles.EndorserRole] = struct{}{}
		roles.SetRoles(rolesValue)
		defer func() { roles.SetRoles(nil) }()
	}

	handler3 := mocks.NewMockBlockHandler()
	AddCCUpgradeHandler(channelID, handler3.HandleChaincodeUpgradeEvent)

	b := mocks.NewBlockBuilder(channelID, 1100)

	lceBytes, err := proto.Marshal(&pb.LifecycleEvent{ChaincodeName: ccID})
	require.NoError(t, err)
	require.NotNil(t, lceBytes)

	b.Transaction(txID, pb.TxValidationCode_VALID).
		ChaincodeAction(lsccID).
		ChaincodeEvent(upgradeEvent, lceBytes)

	blockpublisher.ForChannel(channelID).Publish(b.Build())

	// Wait a bit for the events to be published
	time.Sleep(500 * time.Millisecond)
	require.Equal(t, 1, handler3.NumCCUpgradeEvents())
}

func TestAddCCUpgradeHandlerAsCommitter(t *testing.T) {

	//make sure roles is committer not endorser
	if roles.IsEndorser() {
		rolesValue := make(map[roles.Role]struct{})
		rolesValue[roles.CommitterRole] = struct{}{}
		roles.SetRoles(rolesValue)
		defer func() { roles.SetRoles(nil) }()
	}

	handler3 := mocks.NewMockBlockHandler()
	AddCCUpgradeHandler(channelID, handler3.HandleChaincodeUpgradeEvent)

	b := mocks.NewBlockBuilder(channelID, 1100)

	lceBytes, err := proto.Marshal(&pb.LifecycleEvent{ChaincodeName: ccID})
	require.NoError(t, err)
	require.NotNil(t, lceBytes)

	b.Transaction(txID, pb.TxValidationCode_VALID).
		ChaincodeAction(lsccID).
		ChaincodeEvent(upgradeEvent, lceBytes)

	blockpublisher.ForChannel(channelID).Publish(b.Build())

	// Wait a bit for the events to be published
	time.Sleep(500 * time.Millisecond)
	//cc upgrade event will not get published
	require.Equal(t, 0, handler3.NumCCUpgradeEvents())

}
