/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package roles

import (
	"testing"

	viper "github.com/spf13/viper2015"
	"github.com/stretchr/testify/require"
)

const (
	r1 = "role1"
	r2 = "role2"
	r3 = "role3"
	r4 = "role4"

	role1 = Role(r1)
	role2 = Role(r2)
	role3 = Role(r3)
	role4 = Role(r4)

	confRoles = "ledger.roles"
)

func TestRoles_Contains(t *testing.T) {
	roles := New(role1, role2, role3)

	require.True(t, roles.Contains(role1))
	require.True(t, roles.Contains(role2))
	require.True(t, roles.Contains(role3))
	require.False(t, roles.Contains(role4))

	var emptyRoles Roles
	require.True(t, emptyRoles.Contains(role1))
}

func TestFromStrings(t *testing.T) {
	roles := FromStrings(r1, r2, r3, r4)
	require.Equal(t, role1, roles[0])
	require.Equal(t, role2, roles[1])
	require.Equal(t, role3, roles[2])
	require.Equal(t, role4, roles[3])
}

func TestLocalRoles(t *testing.T) {
	oldVal := viper.Get(confRoles)
	defer viper.Set(confRoles, oldVal)

	testRole := ""
	viper.Set(confRoles, testRole)
	roles = initRoles()
	require.True(t, IsCommitter())
	require.True(t, IsEndorser())
	require.True(t, IsValidator())
	require.False(t, HasRole("newRole"))

	testRole = "committer,endorser,validator"
	viper.Set(confRoles, testRole)
	roles = initRoles()
	require.True(t, IsCommitter())
	require.True(t, IsEndorser())
	require.True(t, IsValidator())
	require.Equal(t, len(GetRoles()), 3)
	require.Equal(t, len(AsString()), 3)

	testRole = "CoMMiTTER,  ENDORSER,   validator"
	viper.Set(confRoles, testRole)
	roles = initRoles()
	require.True(t, IsCommitter())
	require.True(t, IsEndorser())
	require.True(t, IsValidator())

	testRole = "committer,endorser"
	viper.Set(confRoles, testRole)
	roles = initRoles()
	require.True(t, IsCommitter())
	require.True(t, IsEndorser())
	require.False(t, IsValidator())

}
