/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package roles

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsCommitter(t *testing.T) {
	require.True(t, IsCommitter())
}

func TestIsEndorser(t *testing.T) {
	require.True(t, IsEndorser())
}

func TestIsValidator(t *testing.T) {
	require.True(t, IsValidator())
}

func TestRolesAsString(t *testing.T) {
	require.Empty(t, RolesAsString())
}

func TestHasEndorserRole(t *testing.T) {
	endorserSamples := [][]string{
		{"committer", "endorser"},
		{"committer", "Endorser"},
		{"committer", "ENdOrsEr"},
		{"Endorser"},
		{ "enDorser", "committer"},
		{},
	}

	for _, v := range endorserSamples {
		fmt.Println("==",v)
		require.True(t, HasEndorserRole(v))
	}

	nonEndorserSamples := [][]string{
		{"com", "endorsers"},
		{ "committer"},
		{"", ""},
		{""},

	}

	for _, v := range nonEndorserSamples {
		fmt.Println("--",v)
		require.False(t, HasEndorserRole(v))
	}
}
