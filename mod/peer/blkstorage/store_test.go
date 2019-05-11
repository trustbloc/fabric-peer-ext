/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blkstorage

import (
	"testing"

	"github.com/hyperledger/fabric/common/ledger/blkstorage"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/hyperledger/fabric/extensions/testutil"
	"github.com/stretchr/testify/require"
)

func TestNewProvider(t *testing.T) {
	_, _, destroy := testutil.SetupExtTestEnv()
	defer destroy()
	require.NotEmpty(t, NewProvider(NewConf(ledgerconfig.GetBlockStorePath(), ledgerconfig.GetMaxBlockfileSize()), &blkstorage.IndexConfig{}))
}
