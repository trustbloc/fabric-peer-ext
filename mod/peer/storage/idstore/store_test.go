/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idstore

import (
	"testing"

	"github.com/hyperledger/fabric/extensions/testutil"
	"github.com/stretchr/testify/require"
)

func TestOpenIDStore(t *testing.T) {
	_, _, destroy := testutil.SetupExtTestEnv()
	defer destroy()

	s, err := OpenIDStore("", testutil.TestLedgerConf(), nil)
	require.NoError(t, err)
	require.NotEmpty(t, s)
}
