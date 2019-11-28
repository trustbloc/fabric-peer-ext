/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pvtdatastorage

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	coreconfig "github.com/hyperledger/fabric/core/config"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/pvtdatastorage"
	"github.com/hyperledger/fabric/extensions/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestNewProvider(t *testing.T) {
	cleanup := setupPath(t)
	defer cleanup()
	_, _, destroy := testutil.SetupExtTestEnv()
	defer destroy()
	conf := &pvtdatastorage.PrivateDataConfig{
		PrivateDataConfig: &ledger.PrivateDataConfig{
			PurgeInterval: 1,
		},
		StorePath: filepath.Join(coreconfig.GetPath("peer.fileSystemPath"), "store"),
	}
	p, err := NewProvider(conf, testutil.TestLedgerConf())
	require.NoError(t, err)
	require.NotEmpty(t, p)
}

func setupPath(t *testing.T) (cleanup func()) {
	tempDir, err := ioutil.TempDir("", "pvtdatastorage")
	require.NoError(t, err)

	viper.Set("peer.fileSystemPath", tempDir)
	return func() { os.RemoveAll(tempDir) }
}
