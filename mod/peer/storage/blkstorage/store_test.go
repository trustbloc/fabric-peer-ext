/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blkstorage

import (
	"path/filepath"
	"testing"

	"github.com/hyperledger/fabric/common/ledger/blkstorage"
	coreconfig "github.com/hyperledger/fabric/core/config"
	"github.com/hyperledger/fabric/extensions/testutil"
	"github.com/stretchr/testify/require"
)

func TestNewProvider(t *testing.T) {
	_, _, destroy := testutil.SetupExtTestEnv()
	defer destroy()
	require.NotEmpty(t, NewProvider(NewConf(filepath.Join(coreconfig.GetPath("peer.fileSystemPath"), "chains"),
		-1), &blkstorage.IndexConfig{}, testutil.TestLedgerConf()))
}
