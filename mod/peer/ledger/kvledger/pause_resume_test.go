/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvledger

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	configtxtest "github.com/hyperledger/fabric/common/configtx/test"
	"github.com/hyperledger/fabric/core/ledger/kvledger"
	"github.com/hyperledger/fabric/core/ledger/mock"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/idstore"
)

func TestPauseAndResume(t *testing.T) {
	conf, cleanup := testConfig(t)
	conf.HistoryDBConfig.Enabled = false
	defer cleanup()
	provider := testutilNewProvider(conf, t, &mock.DeployedChaincodeInfoProvider{})

	numLedgers := 10
	activeLedgerIDs, err := provider.List()
	require.NoError(t, err)
	require.Len(t, activeLedgerIDs, 0)
	genesisBlocks := make([]*common.Block, numLedgers)
	for i := 0; i < numLedgers; i++ {
		genesisBlock, _ := configtxtest.MakeGenesisBlock(constructTestLedgerID(i))
		genesisBlocks[i] = genesisBlock
		provider.Create(genesisBlock)
	}
	activeLedgerIDs, err = provider.List()
	require.NoError(t, err)
	require.Len(t, activeLedgerIDs, numLedgers)
	provider.Close()

	// pause channels
	pausedLedgers := []int{1, 3, 5}
	for _, i := range pausedLedgers {
		err = PauseChannel(conf, constructTestLedgerID(i))
		require.NoError(t, err)
	}
	// pause again should not fail
	err = PauseChannel(conf, constructTestLedgerID(1))
	require.NoError(t, err)
	// verify ledger status after pause
	provider = testutilNewProvider(conf, t, &mock.DeployedChaincodeInfoProvider{})
	env := idstore.NewTestStoreEnv(t, "", nil)
	assertLedgerStatus(t, provider, env.TestStore, genesisBlocks, numLedgers, pausedLedgers)
	provider.Close()

	// resume channels
	resumedLedgers := []int{1, 5}
	for _, i := range resumedLedgers {
		err = ResumeChannel(conf, constructTestLedgerID(i))
		require.NoError(t, err)
	}
	// resume again should not fail
	err = ResumeChannel(conf, constructTestLedgerID(1))
	require.NoError(t, err)
	// verify ledger status after resume
	pausedLedgersAfterResume := []int{3}
	provider = testutilNewProvider(conf, t, &mock.DeployedChaincodeInfoProvider{})
	defer provider.Close()
	assertLedgerStatus(t, provider, env.TestStore, genesisBlocks, numLedgers, pausedLedgersAfterResume)

	// open paused channel should fail
	_, err = provider.Open(constructTestLedgerID(3))
	require.Equal(t, kvledger.ErrInactiveLedger, err)
}

func TestPauseAndResumeErrors(t *testing.T) {
	conf, cleanup := testConfig(t)
	conf.HistoryDBConfig.Enabled = false
	defer cleanup()
	provider := testutilNewProvider(conf, t, &mock.DeployedChaincodeInfoProvider{})

	ledgerID := constructTestLedgerID(0)
	genesisBlock, _ := configtxtest.MakeGenesisBlock(ledgerID)
	provider.Create(genesisBlock)
	provider.Close()

	// fail if ledgerID does not exists
	err := PauseChannel(conf, "dummy")
	require.Error(t, err, "LedgerID does not exist")

	err = ResumeChannel(conf, "dummy")
	require.Error(t, err, "LedgerID does not exist")
}

// verify status for paused ledgers and non-paused ledgers
func assertLedgerStatus(t *testing.T, provider *kvledger.Provider, s *idstore.Store, genesisBlocks []*common.Block, numLedgers int, pausedLedgers []int) {

	activeLedgerIDs, err := provider.List()
	require.NoError(t, err)
	require.Len(t, activeLedgerIDs, numLedgers-len(pausedLedgers))
	for i := 0; i < numLedgers; i++ {
		if !contains(pausedLedgers, i) {
			require.Contains(t, activeLedgerIDs, constructTestLedgerID(i))
		}
	}

	for i := 0; i < numLedgers; i++ {
		active, exists, err := s.LedgerIDActive(constructTestLedgerID(i))
		require.NoError(t, err)
		if !contains(pausedLedgers, i) {
			require.True(t, active)
			require.True(t, exists)
		} else {
			require.False(t, active)
			require.True(t, exists)
		}

		// every channel (paused or non-paused) should have an entry for genesis block
		gb, err := s.GetGenesisBlock(constructTestLedgerID(i))
		require.NoError(t, err)
		require.True(t, proto.Equal(gb, genesisBlocks[i]), "proto messages are not equal")

	}
}

func contains(slice []int, val int) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
