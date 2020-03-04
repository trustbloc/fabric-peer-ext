/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockvisitor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContext(t *testing.T) {
	ccEvent := &CCEvent{
		BlockNum: 1000,
		TxID:     txID1,
		TxNum:    12,
	}

	lsccWrite := &LSCCWrite{
		BlockNum: 1000,
		TxID:     txID1,
		TxNum:    12,
	}

	configUpdate := &ConfigUpdate{
		BlockNum: 1000,
	}

	read := &Read{
		BlockNum: 1000,
		TxID:     txID1,
		TxNum:    12,
	}

	write := &Write{
		BlockNum: 1000,
		TxID:     txID1,
		TxNum:    12,
	}

	ctx := newContext(
		UnmarshalErr, channelID, 1000,
		withTxID(txID1),
		withTxNum(12),
		withCCEvent(ccEvent),
		withConfigUpdate(configUpdate),
		withLSCCWrite(lsccWrite),
		withRead(read),
		withWrite(write),
	)
	require.NotNil(t, ctx)
	require.Equal(t, UnmarshalErr, ctx.Category)
	require.Equal(t, channelID, ctx.ChannelID)
	require.Equal(t, uint64(1000), ctx.BlockNum)
	require.Equal(t, txID1, ctx.TxID)
	require.Equal(t, ccEvent, ctx.CCEvent)
	require.Equal(t, configUpdate, ctx.ConfigUpdate)
	require.Equal(t, lsccWrite, ctx.LSCCWrite)
	require.Equal(t, read, ctx.Read)
	require.Equal(t, write, ctx.Write)

	s := ctx.String()
	require.Contains(t, s, "ChannelID: testchannel")
	require.Contains(t, s, "Category: UNMARSHAL_ERROR")
	require.Contains(t, s, "Block: 1000")
	require.Contains(t, s, "TxNum: 12")
	require.Contains(t, s, "TxID: tx1")
	require.Contains(t, s, "Read:")
	require.Contains(t, s, "Write:")
	require.Contains(t, s, "CCEvent:")
	require.Contains(t, s, "TxID:tx1")
	require.Contains(t, s, "TxNum:12")
	require.Contains(t, s, "ConfigUpdate:")
}
