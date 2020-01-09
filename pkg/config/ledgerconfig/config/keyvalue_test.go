/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	msp1  = "org1MSP"
	peer1 = "peer1"
	app1  = "app1"
	v1    = "v1"
	comp1 = "comp1"
	tx1   = "tx1"
)

func TestNewAppKey(t *testing.T) {
	key := NewAppKey(msp1, app1, v1)
	require.NotNil(t, key)
	require.Equal(t, msp1, key.MspID)
	require.Equal(t, app1, key.AppName)
	require.Equal(t, v1, key.AppVersion)
	require.Empty(t, key.PeerID)
	require.Empty(t, key.ComponentName)
	require.Empty(t, key.ComponentVersion)
}

func TestNewComponentKey(t *testing.T) {
	key := NewComponentKey(msp1, app1, v1, comp1, v1)
	require.NotNil(t, key)
	require.Equal(t, msp1, key.MspID)
	require.Equal(t, app1, key.AppName)
	require.Equal(t, v1, key.AppVersion)
	require.Equal(t, comp1, key.ComponentName)
	require.Equal(t, v1, key.ComponentVersion)
	require.Empty(t, key.PeerID)
}

func TestNewPeerKey(t *testing.T) {
	key := NewPeerKey(msp1, peer1, app1, v1)
	require.NotNil(t, key)
	require.Equal(t, msp1, key.MspID)
	require.Equal(t, peer1, key.PeerID)
	require.Equal(t, app1, key.AppName)
	require.Equal(t, v1, key.AppVersion)
	require.Empty(t, key.ComponentName)
	require.Empty(t, key.ComponentVersion)
}

func TestNewPeerComponentKey(t *testing.T) {
	key := NewPeerComponentKey(msp1, peer1, app1, v1, comp1, v1)
	require.NotNil(t, key)
	require.Equal(t, msp1, key.MspID)
	require.Equal(t, peer1, key.PeerID)
	require.Equal(t, app1, key.AppName)
	require.Equal(t, v1, key.AppVersion)
	require.Equal(t, comp1, key.ComponentName)
	require.Equal(t, v1, key.ComponentVersion)
}

func TestNewKeyValue(t *testing.T) {
	key := NewAppKey(msp1, app1, v1)
	value := NewValue(tx1, "some config", FormatOther)
	kv := NewKeyValue(key, value)
	require.NotNil(t, kv)
	require.Equal(t, key, kv.Key)
	require.Equal(t, value, kv.Value)
}

func TestKey_String(t *testing.T) {
	key := NewAppKey(msp1, app1, v1)
	require.Equal(t, "(MSP:org1MSP),(Peer:),(AppName:app1),(AppVersion:v1),(Comp:),(CompVersion:)", key.String())
}

func TestValue_String(t *testing.T) {
	kv := NewKeyValue(NewAppKey(msp1, app1, v1), NewValue(tx1, "some config", FormatOther))
	require.Equal(t, "[(MSP:org1MSP),(Peer:),(AppName:app1),(AppVersion:v1),(Comp:),(CompVersion:)]=[(TxID:tx1),(Config:some config),(Format:OTHER)]", kv.String())
}

func TestKey_Validate(t *testing.T) {
	key := &Key{}
	require.EqualError(t, key.Validate(), "field [MspID] is required")

	key = &Key{MspID: msp1, PeerID: peer1}
	require.EqualError(t, key.Validate(), "field [Name] is required for PeerID [peer1]")

	key = &Key{MspID: msp1, AppName: app1}
	require.EqualError(t, key.Validate(), "field [AppVersion] is required")

	key = &Key{MspID: msp1, AppVersion: v1}
	require.EqualError(t, key.Validate(), "one of the fields [PeerID] or [Name] is required")

	key = &Key{MspID: msp1, AppName: app1, AppVersion: v1, ComponentName: comp1}
	require.EqualError(t, key.Validate(), "field [ComponentVersion] is required")

	key = &Key{MspID: msp1, AppName: app1, AppVersion: v1, ComponentVersion: v1}
	require.EqualError(t, key.Validate(), "field [ComponentName] is required")

	key = NewPeerComponentKey(msp1, peer1, app1, v1, comp1, v1)
	require.NoError(t, key.Validate())
}
