/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idstore

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/spf13/viper"

	"github.com/stretchr/testify/require"
)

func TestOpenIDStore(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "idstore")
	require.NoError(t, err)
	viper.Set("peer.fileSystemPath", tempDir)
	defer os.RemoveAll(tempDir)
	require.NotEmpty(t, OpenIDStore(tempDir))
}
