/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pvtdatastorage

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/hyperledger/fabric/extensions/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestNewProvider(t *testing.T) {
	cleanup := setupPath(t)
	defer cleanup()
	_, _, destroy := testutil.SetupExtTestEnv()
	defer destroy()
	require.NotEmpty(t, NewProvider())
}

func setupPath(t *testing.T) (cleanup func()) {
	tempDir, err := ioutil.TempDir("", "pvtdatastorage")
	require.NoError(t, err)

	viper.Set("peer.fileSystemPath", tempDir)
	return func() { os.RemoveAll(tempDir) }
}
