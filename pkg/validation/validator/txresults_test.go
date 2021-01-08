/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator

import (
	"testing"

	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/txflags"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

const (
	channelID = "channel1"

	txID1 = "tx1"
	txID2 = "tx2"
	txID3 = "tx3"
)

func TestTxResults(t *testing.T) {
	bb := mocks.NewBlockBuilder(channelID, 1000)
	bb.Transaction(txID1, peer.TxValidationCode_NOT_VALIDATED)
	bb.Transaction(txID2, peer.TxValidationCode_NOT_VALIDATED)
	bb.Transaction(txID2, peer.TxValidationCode_NOT_VALIDATED)

	r := newTxResults(channelID, bb.Build())
	require.NotNil(t, r)
	require.False(t, r.AllValidated())
	require.Len(t, r.UnvalidatedMap(), 3)

	f := txflags.New(2)
	done, err := r.Merge(f, []string{"", "", ""})
	require.EqualError(t, err, "the length of the provided flags 2 does not match the length of the existing flags 3")

	f = txflags.New(3)
	done, err = r.Merge(f, []string{"", ""})
	require.EqualError(t, err, "the length of the provided Tx IDs 2 does not match the length of the existing Tx IDs 3")

	f = txflags.New(3)
	f.SetFlag(1, peer.TxValidationCode_VALID)

	done, err = r.Merge(f, []string{"", txID2, ""})
	require.NoError(t, err)
	require.False(t, done)
	require.Len(t, r.UnvalidatedMap(), 2)

	f.SetFlag(0, peer.TxValidationCode_VALID)
	f.SetFlag(2, peer.TxValidationCode_MVCC_READ_CONFLICT)

	done, err = r.Merge(f, []string{txID1, "", txID2})
	require.NoError(t, err)
	require.True(t, done)
	require.Empty(t, r.UnvalidatedMap())
	require.Equal(t, txflags.ValidationFlags{
		uint8(peer.TxValidationCode_VALID),
		uint8(peer.TxValidationCode_VALID),
		uint8(peer.TxValidationCode_DUPLICATE_TXID),
	}, r.Flags())
}
