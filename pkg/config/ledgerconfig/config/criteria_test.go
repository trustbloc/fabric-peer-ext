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

func TestCriteria_AsKey(t *testing.T) {
	t.Run("Incomplete criteria for key -> error", func(t *testing.T) {
		c := &Criteria{}
		k, err := c.AsKey()
		require.EqualError(t, err, "criteria is not unique")
		require.Nil(t, k)
	})
	t.Run("Complete criteria for key -> success", func(t *testing.T) {
		c := &Criteria{MspID: msp1, PeerID: peer1, AppName: app1, AppVersion: v1, ComponentName: comp1, ComponentVersion: v1}
		k, err := c.AsKey()
		require.NoError(t, err)
		require.NotNil(t, k)
		require.Equal(t, c.MspID, k.MspID)
		require.Equal(t, c.AppName, k.AppName)
		require.Equal(t, c.AppVersion, k.AppVersion)
		require.Equal(t, c.PeerID, k.PeerID)
		require.Equal(t, c.ComponentName, k.ComponentName)
		require.Equal(t, c.ComponentVersion, k.ComponentVersion)
	})
}

func TestCriteriaFromKey(t *testing.T) {
	k := NewPeerComponentKey(msp1, peer1, app1, v1, comp1, v1)
	c := CriteriaFromKey(k)
	require.NotNil(t, c)
	require.Equal(t, k.MspID, c.MspID)
	require.Equal(t, k.AppName, c.AppName)
	require.Equal(t, k.AppVersion, c.AppVersion)
	require.Equal(t, k.PeerID, c.PeerID)
	require.Equal(t, k.ComponentName, c.ComponentName)
	require.Equal(t, k.ComponentVersion, c.ComponentVersion)
}
