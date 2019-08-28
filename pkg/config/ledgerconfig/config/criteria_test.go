/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCriteria_IsUnique(t *testing.T) {
	t.Run("Unique", func(t *testing.T) {
		c := Criteria{MspID: msp1, AppName: app1, AppVersion: v1}
		require.True(t, c.IsUnique())

		c = Criteria{MspID: msp1, AppName: app1, AppVersion: v1, ComponentName: comp1, ComponentVersion: v1}
		require.True(t, c.IsUnique())

		c = Criteria{MspID: msp1, PeerID: peer1, AppName: app1, AppVersion: v1}
		require.True(t, c.IsUnique())

		c = Criteria{MspID: msp1, PeerID: peer1, AppName: app1, AppVersion: v1, ComponentName: comp1, ComponentVersion: v1}
		require.True(t, c.IsUnique())
	})

	t.Run("Not unique", func(t *testing.T) {
		c := Criteria{}
		require.False(t, c.IsUnique())

		c = Criteria{MspID: msp1}
		require.False(t, c.IsUnique())

		c = Criteria{MspID: msp1, AppName: app1}
		require.False(t, c.IsUnique())

		c = Criteria{MspID: msp1, AppName: app1, AppVersion: v1, ComponentName: comp1}
		require.False(t, c.IsUnique())

		c = Criteria{MspID: msp1, PeerID: peer1}
		require.False(t, c.IsUnique())

		c = Criteria{MspID: msp1, PeerID: peer1, AppName: app1, AppVersion: v1, ComponentName: comp1}
		require.False(t, c.IsUnique())
	})
}

func TestCriteria_Validate(t *testing.T) {
	t.Run("Test valid", func(t *testing.T) {
		c := Criteria{MspID: msp1, PeerID: peer1}
		require.NoError(t, c.Validate())

		c = Criteria{MspID: msp1, AppName: app1}
		require.NoError(t, c.Validate())

		c = Criteria{MspID: msp1, AppName: app1, AppVersion: v1}
		require.NoError(t, c.Validate())

		c = Criteria{MspID: msp1, PeerID: peer1, AppName: app1}
		require.NoError(t, c.Validate())

		c = Criteria{MspID: msp1, PeerID: peer1, AppName: app1, AppVersion: v1}
		require.NoError(t, c.Validate())

		c = Criteria{MspID: msp1, AppName: app1, ComponentName: comp1}
		require.NoError(t, c.Validate())

		c = Criteria{MspID: msp1, AppName: app1, ComponentName: comp1, ComponentVersion: v1}
		require.NoError(t, c.Validate())
	})

	t.Run("Test invalid", func(t *testing.T) {
		c := Criteria{}
		require.EqualError(t, c.Validate(), "field [MspID] is required")

		c = Criteria{MspID: msp1, AppVersion: v1}
		require.EqualError(t, c.Validate(), "field [Name] is required")

		c = Criteria{MspID: msp1, ComponentName: comp1}
		require.EqualError(t, c.Validate(), "field [Name] is required")

		c = Criteria{MspID: msp1, AppName: app1, ComponentVersion: v1}
		require.EqualError(t, c.Validate(), "field [ComponentName] is required")
	})
}

func TestCriteria_String(t *testing.T) {
	c := Criteria{MspID: msp1, PeerID: peer1, AppName: app1, AppVersion: v1, ComponentName: comp1, ComponentVersion: v1}
	require.Equal(t, "(MSP:org1MSP),(Peer:peer1),(App:app1),(AppVersion:v1),(Comp:comp1),(CompVersion:v1)", c.String())
}
