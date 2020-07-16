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
	viper "github.com/spf13/viper2015"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/config"
	xpvtdatastorage "github.com/trustbloc/fabric-peer-ext/pkg/pvtdatastorage"
)

func TestNewProvider(t *testing.T) {
	t.Run("CouchDB", func(t *testing.T) {
		dbType := config.GetPrivateDataStoreDBType()
		require.Equal(t, config.CouchDBType, dbType)

		cleanup := setupPath(t, "pvtdatastorage")
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
		require.NotNil(t, p)

		_, ok := p.(*xpvtdatastorage.PvtDataProvider)
		require.True(t, ok)
	})

	t.Run("Default", func(t *testing.T) {
		cleanup := setupPath(t, "pvtdatastorage2")
		defer cleanup()

		viper.Set(config.ConfPrivateDataStoreDBType, config.LevelDBType)
		dbType := config.GetPrivateDataStoreDBType()
		require.Equal(t, config.LevelDBType, dbType)
		defer func() { viper.Set(config.ConfPrivateDataStoreDBType, dbType) }()

		conf := &pvtdatastorage.PrivateDataConfig{
			PrivateDataConfig: &ledger.PrivateDataConfig{
				PurgeInterval: 1,
			},
			StorePath: filepath.Join(coreconfig.GetPath("peer.fileSystemPath"), "store2"),
		}

		p, err := NewProvider(conf, testutil.TestLedgerConf())
		require.NoError(t, err)
		require.NotEmpty(t, p)

		_, ok := p.(*pvtdatastorage.Provider)
		require.True(t, ok)
	})
}

func setupPath(t *testing.T, name string) (cleanup func()) {
	tempDir, err := ioutil.TempDir("", name)
	require.NoError(t, err)

	viper.Set("peer.fileSystemPath", tempDir)
	return func() { os.RemoveAll(tempDir) }
}
