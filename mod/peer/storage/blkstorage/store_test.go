/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blkstorage

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/hyperledger/fabric/common/ledger/blkstorage"
	coreconfig "github.com/hyperledger/fabric/core/config"
	"github.com/hyperledger/fabric/extensions/storage/blkstorage/mocks"
	"github.com/hyperledger/fabric/extensions/testutil"
	viper "github.com/spf13/viper2015"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/blkstorage/cdbblkstorage"
	"github.com/trustbloc/fabric-peer-ext/pkg/config"
)

//go:generate counterfeiter -o ./mocks/metricsprovider.gen.go --fake-name MetricsProvider github.com/hyperledger/fabric/common/metrics.Provider
//go:generate counterfeiter -o ./mocks/metricsgauge.gen.go --fake-name MetricsGauge github.com/hyperledger/fabric/common/metrics.Gauge

func TestNewProvider(t *testing.T) {
	_, _, destroy := testutil.SetupExtTestEnv()
	defer destroy()

	t.Run("CouchDB", func(t *testing.T) {
		dbType := config.GetBlockStoreDBType()
		require.Equal(t, config.CouchDBType, dbType)

		p, err := NewProvider(NewConf(filepath.Join(coreconfig.GetPath("peer.fileSystemPath"), "chains"),
			-1), &blkstorage.IndexConfig{}, testutil.TestLedgerConf(), &mocks.MetricsProvider{})
		require.NoError(t, err)
		require.NotNil(t, p)

		_, ok := p.(*cdbblkstorage.CDBBlockstoreProvider)
		require.True(t, ok)
	})

	t.Run("Default", func(t *testing.T) {
		cleanup := setupPath(t)
		defer cleanup()

		viper.Set(config.ConfBlockStoreDBType, config.LevelDBType)
		dbType := config.GetBlockStoreDBType()
		require.Equal(t, config.LevelDBType, dbType)
		defer func() { viper.Set(config.ConfBlockStoreDBType, dbType) }()

		metricsProvider := &mocks.MetricsProvider{}
		guage := &mocks.MetricsGauge{}
		metricsProvider.NewGaugeReturns(guage)

		p, err := NewProvider(
			NewConf(filepath.Join(coreconfig.GetPath("peer.fileSystemPath"), "chains"), -1),
			&blkstorage.IndexConfig{},
			testutil.TestLedgerConf(),
			metricsProvider,
		)
		require.NoError(t, err)
		require.NotNil(t, p)

		_, ok := p.(*blkstorage.BlockStoreProvider)
		require.True(t, ok)
	})
}

func setupPath(t *testing.T) (cleanup func()) {
	tempDir, err := ioutil.TempDir("", "blockstorage")
	require.NoError(t, err)

	viper.Set("peer.fileSystemPath", tempDir)
	return func() { os.RemoveAll(tempDir) }
}
