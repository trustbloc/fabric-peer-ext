/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idstore

import (
	"testing"

	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/extensions/storage/api"
	"github.com/hyperledger/fabric/extensions/storage/idstore/mocks"
	"github.com/hyperledger/fabric/extensions/testutil"
	viper "github.com/spf13/viper2015"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/config"
	xidstore "github.com/trustbloc/fabric-peer-ext/pkg/idstore"
)

//go:generate counterfeiter -o ./mocks/idstore.gen.go --fake-name IDStore ../api IDStore

func TestOpenIDStore(t *testing.T) {
	_, _, destroy := testutil.SetupExtTestEnv()
	defer destroy()

	t.Run("CouchDB", func(t *testing.T) {
		dbType := config.GetIDStoreDBType()
		require.Equal(t, config.CouchDBType, dbType)

		s, err := OpenIDStore("", testutil.TestLedgerConf(), nil)
		require.NoError(t, err)
		require.NotNil(t, s)

		_, ok := s.(*xidstore.Store)
		require.True(t, ok)
	})

	t.Run("Default", func(t *testing.T) {
		viper.Set(config.ConfIDStoreDBType, config.LevelDBType)
		dbType := config.GetIDStoreDBType()
		require.Equal(t, config.LevelDBType, dbType)
		defer func() { viper.Set(config.ConfIDStoreDBType, dbType) }()

		idStore := &mocks.IDStore{}
		s, err := OpenIDStore("", testutil.TestLedgerConf(),
			func(path string, ledgerconfig *ledger.Config) (storageapi.IDStore, error) {
				return idStore, nil
			},
		)
		require.NoError(t, err)
		require.True(t, s == idStore)
	})
}
