/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvledger

import (
	"testing"

	configtxtest "github.com/hyperledger/fabric/common/configtx/test"
	"github.com/hyperledger/fabric/core/ledger/mock"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/idstore"
)

func TestLedgerMetataDataUnmarshalError(t *testing.T) {
	conf, cleanup := testConfig(t)
	defer cleanup()
	provider := testutilNewProvider(conf, t, &mock.DeployedChaincodeInfoProvider{})
	defer provider.Close()

	ledgerID := constructTestLedgerID(0)
	genesisBlock, _ := configtxtest.MakeGenesisBlock(ledgerID)
	provider.Create(genesisBlock)

	env := idstore.NewTestStoreEnv(t, "", nil)
	// put invalid bytes for the metatdata key
	require.NoError(t, idstore.SaveLedgerID(env.TestStore, ledgerID, "", "active"))

	_, err := provider.List()
	require.EqualError(t, err, "ledger inventory document is invalid [ledger_ledger_000000]")

	_, err = provider.Open(ledgerID)
	require.EqualError(t, err, "Ledger is not active")
}
