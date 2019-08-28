/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mgr

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/config"
)

func TestConfigResults_Filter(t *testing.T) {
	v := config.NewValue(txID1, "some config", config.FormatOther)

	keyPeer0App1V1 := config.NewPeerKey(msp1, peer1, app1, v1)
	keyPeer0App1V2 := config.NewPeerKey(msp1, peer1, app1, v2)
	keyPeer0App2V1 := config.NewPeerKey(msp1, peer1, app2, v1)
	keyPeer0App2V2 := config.NewPeerKey(msp1, peer1, app2, v2)
	keyPeer1App1V1 := config.NewPeerKey(msp1, peer2, app1, v1)
	keyPeer1App1V2 := config.NewPeerKey(msp1, peer2, app1, v2)
	keyPeer1App2V1 := config.NewPeerKey(msp1, peer2, app2, v1)
	keyPeer1App2V2 := config.NewPeerKey(msp1, peer2, app2, v2)

	keyApp3V1 := config.NewAppKey(msp1, app3, v1)
	keyApp3V2 := config.NewAppKey(msp1, app3, v2)
	keyApp4V1 := config.NewAppKey(msp1, app4, v1)
	keyApp4V2 := config.NewAppKey(msp1, app4, v2)

	keyApp5V1Comp1V1 := config.NewComponentKey(msp1, app5, v1, comp1, v1)
	keyApp5V1Comp1V2 := config.NewComponentKey(msp1, app5, v1, comp1, v2)
	keyApp5V1Comp2V1 := config.NewComponentKey(msp1, app5, v1, comp2, v1)
	keyApp5V1Comp2V2 := config.NewComponentKey(msp1, app5, v1, comp2, v2)
	keyApp6V1Comp1V1 := config.NewComponentKey(msp1, app6, v1, comp1, v1)
	keyApp6V1Comp1V2 := config.NewComponentKey(msp1, app6, v1, comp1, v2)
	keyApp6V1Comp2V1 := config.NewComponentKey(msp1, app6, v1, comp2, v1)
	keyApp6V1Comp2V2 := config.NewComponentKey(msp1, app6, v1, comp2, v2)

	r1 := ConfigResults{
		{Key: keyPeer0App1V1, Value: v},
		{Key: keyPeer0App1V2, Value: v},
		{Key: keyPeer0App2V1, Value: v},
		{Key: keyPeer0App2V2, Value: v},
		{Key: keyPeer1App1V1, Value: v},
		{Key: keyPeer1App1V2, Value: v},
		{Key: keyPeer1App2V1, Value: v},
		{Key: keyPeer1App2V2, Value: v},

		{Key: keyApp3V1, Value: v},
		{Key: keyApp3V2, Value: v},
		{Key: keyApp4V1, Value: v},
		{Key: keyApp4V2, Value: v},

		{Key: keyApp5V1Comp1V1, Value: v},
		{Key: keyApp5V1Comp1V2, Value: v},
		{Key: keyApp5V1Comp2V1, Value: v},
		{Key: keyApp5V1Comp2V2, Value: v},
		{Key: keyApp6V1Comp1V1, Value: v},
		{Key: keyApp6V1Comp1V2, Value: v},
		{Key: keyApp6V1Comp2V1, Value: v},
		{Key: keyApp6V1Comp2V2, Value: v},
	}

	t.Run("No filter", func(t *testing.T) {
		r := r1.Filter(&config.Criteria{MspID: msp1})
		require.Equal(t, r1, r)
	})

	t.Run("PeerID filter", func(t *testing.T) {
		r := r1.Filter(&config.Criteria{MspID: msp1, PeerID: peer2})
		require.Len(t, r, 4)
		require.Equal(t, keyPeer1App1V1, r[0].Key)
		require.Equal(t, keyPeer1App1V2, r[1].Key)
		require.Equal(t, keyPeer1App2V1, r[2].Key)
		require.Equal(t, keyPeer1App2V2, r[3].Key)
	})

	t.Run("PeerID Apps filter", func(t *testing.T) {
		r := r1.Filter(&config.Criteria{MspID: msp1, PeerID: peer2, AppName: app2})
		require.Len(t, r, 2)
		require.Equal(t, keyPeer1App2V1, r[0].Key)
		require.Equal(t, keyPeer1App2V2, r[1].Key)
	})

	t.Run("PeerID Apps Version filter", func(t *testing.T) {
		r := r1.Filter(&config.Criteria{MspID: msp1, PeerID: peer2, AppName: app2, AppVersion: v2})
		require.Len(t, r, 1)
		require.Equal(t, keyPeer1App2V2, r[0].Key)
	})

	t.Run("Apps filter", func(t *testing.T) {
		r := r1.Filter(&config.Criteria{MspID: msp1, AppName: app4})
		require.Len(t, r, 2)
		require.Equal(t, keyApp4V1, r[0].Key)
		require.Equal(t, keyApp4V2, r[1].Key)
	})

	t.Run("Apps Version filter", func(t *testing.T) {
		r := r1.Filter(&config.Criteria{MspID: msp1, AppName: app4, AppVersion: v2})
		require.Len(t, r, 1)
		require.Equal(t, keyApp4V2, r[0].Key)
	})

	t.Run("Apps ComponentName filter", func(t *testing.T) {
		r := r1.Filter(&config.Criteria{MspID: msp1, AppName: app6, ComponentName: comp2})
		require.Len(t, r, 2)
		require.Equal(t, keyApp6V1Comp2V1, r[0].Key)
		require.Equal(t, keyApp6V1Comp2V2, r[1].Key)
	})

	t.Run("Apps ComponentName filter", func(t *testing.T) {
		r := r1.Filter(&config.Criteria{MspID: msp1, AppName: app6, ComponentName: comp2, ComponentVersion: v2})
		require.Len(t, r, 1)
		require.Equal(t, keyApp6V1Comp2V2, r[0].Key)
	})
}
