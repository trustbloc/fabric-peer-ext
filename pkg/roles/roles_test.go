/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package roles

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
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

	assert.True(t, roles.Contains(role1))
	assert.True(t, roles.Contains(role2))
	assert.True(t, roles.Contains(role3))
	assert.False(t, roles.Contains(role4))

	var emptyRoles Roles
	assert.True(t, emptyRoles.Contains(role1))
}

func TestFromStrings(t *testing.T) {
	roles := FromStrings(r1, r2, r3, r4)
	assert.Equal(t, role1, roles[0])
	assert.Equal(t, role2, roles[1])
	assert.Equal(t, role3, roles[2])
	assert.Equal(t, role4, roles[3])
}

func TestLocalRoles(t *testing.T) {
	oldVal := viper.Get(confRoles)
	defer viper.Set(confRoles, oldVal)

	testRole := ""
	viper.Set(confRoles, testRole)
	roles = getRoles()
	assert.True(t, IsCommitter())
	assert.True(t, IsEndorser())
	assert.True(t, IsValidator())

	testRole = "committer,endorser,validator"
	viper.Set(confRoles, testRole)
	roles = getRoles()
	assert.True(t, IsCommitter())
	assert.True(t, IsEndorser())
	assert.True(t, IsValidator())

	testRole = "CoMMiTTER,  ENDORSER,   validator"
	viper.Set(confRoles, testRole)
	roles = getRoles()
	assert.True(t, IsCommitter())
	assert.True(t, IsEndorser())
	assert.True(t, IsValidator())

	testRole = "committer,endorser"
	viper.Set(confRoles, testRole)
	roles = getRoles()
	assert.True(t, IsCommitter())
	assert.True(t, IsEndorser())
	assert.False(t, IsValidator())

}
