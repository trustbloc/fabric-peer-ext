/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package endorser

import (
	"testing"

	"github.com/hyperledger/fabric-protos-go/ledger/rwset"
	"github.com/stretchr/testify/require"
)

const (
	channelID = "testchannel"
)

func TestFilterPubSimulationResults(t *testing.T) {
	f := NewCollRWSetFilter()
	require.NotNil(t, f)

	pubSimulationResults := &rwset.TxReadWriteSet{}
	p, err := f.Filter(channelID, pubSimulationResults)
	require.NoError(t, err)
	require.Equal(t, pubSimulationResults, p)
}
