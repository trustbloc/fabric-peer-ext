/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transientstore

import (
	"io/ioutil"
	"os"
	"testing"

	viper "github.com/spf13/viper2015"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/config"
	exttransientstore "github.com/trustbloc/fabric-peer-ext/pkg/transientstore"
)

func TestNewStoreProvider(t *testing.T) {
	t.Run("Memory store", func(t *testing.T) {
		dbType := config.GetTransientStoreDBType()
		require.Equal(t, config.MemDBType, dbType)

		p, err := NewStoreProvider("")
		require.NoError(t, err)
		require.NotNil(t, p)

		_, ok := p.(*exttransientstore.Provider)
		require.True(t, ok)
	})

	t.Run("Default", func(t *testing.T) {
		path, cleanup := setupPath(t, "transientstore")
		defer cleanup()

		viper.Set(config.ConfTransientStoreDBType, config.LevelDBType)
		dbType := config.GetTransientStoreDBType()
		require.Equal(t, config.LevelDBType, dbType)
		defer func() { viper.Set(config.ConfTransientStoreDBType, dbType) }()

		p, err := NewStoreProvider(path)
		require.NoError(t, err)
		require.NotNil(t, p)

		_, ok := p.(*exttransientstore.Provider)
		require.False(t, ok)
	})
}

func setupPath(t *testing.T, name string) (string, func()) {
	tempDir, err := ioutil.TempDir("", name)
	require.NoError(t, err)

	return tempDir, func() { os.RemoveAll(tempDir) }
}
