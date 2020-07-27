/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package notifier

import (
	"fmt"
	"testing"

	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/gossip/blockpublisher"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

//go:generate counterfeiter -o ../../mocks/ccupdatehandler.gen.go -fake-name ChaincodeUpdateHandler . updateHandler

const (
	channel1 = "channel1"
	txID1    = "tx1"
	coll1    = "collection1"
)

var _ = roles.GetRoles()

func TestNotifier(t *testing.T) {
	bpp := blockpublisher.NewProvider()
	lp := &mocks.LedgerProvider{}

	n := New(&mocks.ChaincodeUpdateHandler{}, bpp, lp)
	require.NotNil(t, n)

	t.Run("Committer role", func(t *testing.T) {
		n.ChannelJoined(channel1)
	})

	t.Run("Non-commiter role", func(t *testing.T) {
		roles.SetRoles(map[roles.Role]struct{}{roles.EndorserRole: {}})
		defer roles.SetRoles(nil)

		lp.GetLedgerReturns(&mocks.Ledger{})

		n.ChannelJoined(channel1)

		t.Run("Lifecycle writes", func(t *testing.T) {
			bb := mocks.NewBlockBuilder(channel1, 1100)
			tb1 := bb.Transaction(txID1, pb.TxValidationCode_VALID)
			action := tb1.ChaincodeAction(lifecycleNamespace)
			action.Write("cc1", []byte("cc update"))
			action.Collection(coll1).HashedWrite([]byte("key"), []byte("value"))

			bpp.ForChannel(channel1).Publish(bb.Build(), nil)
		})

		t.Run("Irrelevant writes", func(t *testing.T) {
			bb := mocks.NewBlockBuilder(channel1, 1100)
			tb1 := bb.Transaction(txID1, pb.TxValidationCode_VALID)
			action := tb1.ChaincodeAction("somecc")
			action.Write("cc1", []byte("cc update"))
			action.Collection(coll1).HashedWrite([]byte("key"), []byte("value"))

			bpp.ForChannel(channel1).Publish(bb.Build(), nil)
		})

		t.Run("QueryExecutor error", func(t *testing.T) {
			errExpected := fmt.Errorf("injected query exector error")

			lp := &mocks.LedgerProvider{}
			lp.GetLedgerReturns(&mocks.Ledger{Error: errExpected})

			n := New(&mocks.ChaincodeUpdateHandler{}, bpp, lp)
			require.NotNil(t, n)

			n.ChannelJoined(channel1)

			bb := mocks.NewBlockBuilder(channel1, 1100)
			tb1 := bb.Transaction(txID1, pb.TxValidationCode_VALID)
			action := tb1.ChaincodeAction(lifecycleNamespace)
			action.Write("cc1", []byte("cc update"))
			action.Collection(coll1).HashedWrite([]byte("key"), []byte("value"))

			bpp.ForChannel(channel1).Publish(bb.Build(), nil)
		})
	})
}
