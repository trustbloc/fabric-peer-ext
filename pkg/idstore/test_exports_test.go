/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idstore

import (
	"fmt"
	"testing"

	"github.com/hyperledger/fabric/core/ledger"
	"github.com/stretchr/testify/require"
)

func TestNewTestStoreEnv(t *testing.T) {
	restoreIDStore := openIDStore
	openIDStore = func(*ledger.Config) (store *Store, err error) {
		return nil, fmt.Errorf("injected error")
	}
	defer func() { openIDStore = restoreIDStore }()

	require.Panics(t, func() { NewTestStoreEnv(t, "ledgerid", nil) })
}
