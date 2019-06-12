/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ledgerstorage

import (
	"errors"
	"testing"

	"github.com/trustbloc/fabric-peer-ext/pkg/roles"

	"github.com/stretchr/testify/require"
)

func TestSyncPvtdataStoreWithBlockStoreHandlerAsCommitter(t *testing.T) {
	sampleError := errors.New("sample-error")
	handle := func() error {
		return sampleError
	}

	//make sure roles is committer not endorser
	if roles.IsEndorser() {
		rolesValue := make(map[roles.Role]struct{})
		rolesValue[roles.CommitterRole] = struct{}{}
		roles.SetRoles(rolesValue)
		defer func() { roles.SetRoles(nil) }()
	}
	require.True(t, roles.IsCommitter())
	require.False(t, roles.IsEndorser())

	err := SyncPvtdataStoreWithBlockStoreHandler(handle)()
	require.Equal(t, sampleError, err)
}

func TestSyncPvtdataStoreWithBlockStoreHandlerAsEndorser(t *testing.T) {
	sampleError := errors.New("sample-error")
	handle := func() error {
		return sampleError
	}

	//make sure roles is endorser not committer
	if roles.IsCommitter() {
		rolesValue := make(map[roles.Role]struct{})
		rolesValue[roles.EndorserRole] = struct{}{}
		roles.SetRoles(rolesValue)
		defer func() { roles.SetRoles(nil) }()
	}
	require.True(t, roles.IsEndorser())
	require.False(t, roles.IsCommitter())

	err := SyncPvtdataStoreWithBlockStoreHandler(handle)()
	require.NoError(t, err)
}
