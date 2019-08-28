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
	configData = "x = y"
)

func TestConfig_Validate(t *testing.T) {
	t.Run("Apps", func(t *testing.T) {
		cfg := &Config{
			MspID: msp1,
			Apps: []*App{
				{AppName: app1, Version: v1, Format: FormatOther, Config: configData},
			},
		}
		require.NoError(t, cfg.Validate())
	})

	t.Run("Apps components", func(t *testing.T) {
		cfg := &Config{
			MspID: msp1,
			Apps: []*App{
				{
					AppName: app1, Version: v1,
					Components: []*Component{
						{Name: comp1, Version: v1, Format: FormatOther, Config: configData},
					},
				},
			},
		}
		require.NoError(t, cfg.Validate())
	})

	t.Run("Peer apps", func(t *testing.T) {
		cfg := &Config{
			MspID: msp1,
			Peers: []*Peer{
				{
					PeerID: peer1,
					Apps: []*App{
						{AppName: app1, Version: v1, Format: FormatOther, Config: configData},
					},
				},
			},
		}
		require.NoError(t, cfg.Validate())
	})

	t.Run("Peer app components", func(t *testing.T) {
		cfg := &Config{
			MspID: msp1,
			Peers: []*Peer{
				{
					PeerID: peer1,
					Apps: []*App{
						{
							AppName: app1, Version: v1,
							Components: []*Component{
								{Name: comp1, Version: v1, Format: FormatOther, Config: configData},
							},
						},
					},
				},
			},
		}
		require.NoError(t, cfg.Validate())
	})

	t.Run("No MSP ID", func(t *testing.T) {
		cfg := &Config{}
		err := cfg.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "field [MspID] is required")
	})

	t.Run("No peers nor apps", func(t *testing.T) {
		cfg := &Config{
			MspID: msp1,
		}
		err := cfg.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "either one of the fields [Peers] or [Apps] is required")
	})

	t.Run("No app name", func(t *testing.T) {
		cfg := &Config{
			MspID: msp1,
			Apps: []*App{
				{Version: v1, Config: configData},
			},
		}
		err := cfg.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "field [Name] is required")
	})

	t.Run("No peer ID", func(t *testing.T) {
		cfg := &Config{
			MspID: msp1,
			Peers: []*Peer{
				{
					Apps: []*App{
						{AppName: app1, Version: v1, Config: configData},
					},
				},
			},
		}
		err := cfg.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "field [PeerID] is required")
	})

	t.Run("No peer app", func(t *testing.T) {
		cfg := &Config{
			MspID: msp1,
			Peers: []*Peer{
				{PeerID: peer1},
			},
		}
		err := cfg.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "field [Apps] is required")
	})

	t.Run("No peer app name", func(t *testing.T) {
		cfg := &Config{
			MspID: msp1,
			Peers: []*Peer{
				{
					PeerID: peer1,
					Apps: []*App{
						{Version: v1, Config: configData},
					},
				},
			},
		}
		err := cfg.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "field [Name] is required")
	})

	t.Run("No peer app version", func(t *testing.T) {
		cfg := &Config{
			MspID: msp1,
			Peers: []*Peer{
				{
					PeerID: peer1,
					Apps: []*App{
						{AppName: app1, Config: configData},
					},
				},
			},
		}
		err := cfg.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "field [Version] is required")
	})

	t.Run("No peer app config", func(t *testing.T) {
		cfg := &Config{
			MspID: msp1,
			Peers: []*Peer{
				{
					PeerID: peer1,
					Apps: []*App{
						{AppName: app1, Version: v1},
					},
				},
			},
		}
		err := cfg.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "either one of the fields [Config] or [Components] is required")
	})

	t.Run("No peer app config format", func(t *testing.T) {
		cfg := &Config{
			MspID: msp1,
			Peers: []*Peer{
				{
					PeerID: peer1,
					Apps: []*App{
						{AppName: app1, Version: v1, Config: configData},
					},
				},
			},
		}
		err := cfg.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "field [Format] is required")
	})

	t.Run("No peer app component name", func(t *testing.T) {
		cfg := &Config{
			MspID: msp1,
			Peers: []*Peer{
				{
					PeerID: peer1,
					Apps: []*App{
						{
							AppName: app1, Version: v1,
							Components: []*Component{
								{Version: v1, Config: configData},
							},
						},
					},
				},
			},
		}
		err := cfg.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "field [Name] is required")
	})

	t.Run("No peer app component version", func(t *testing.T) {
		cfg := &Config{
			MspID: msp1,
			Peers: []*Peer{
				{
					PeerID: peer1,
					Apps: []*App{
						{
							AppName: app1, Version: v1,
							Components: []*Component{
								{Name: comp1, Format: FormatOther, Config: configData},
							},
						},
					},
				},
			},
		}
		err := cfg.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "field [Version] is required")
	})

	t.Run("No peer app component config", func(t *testing.T) {
		cfg := &Config{
			MspID: msp1,
			Peers: []*Peer{
				{
					PeerID: peer1,
					Apps: []*App{
						{
							AppName: app1, Version: v1,
							Components: []*Component{
								{Name: comp1, Version: v1},
							},
						},
					},
				},
			},
		}
		err := cfg.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "field [Config] is required")
	})

	t.Run("No peer app component config format", func(t *testing.T) {
		cfg := &Config{
			MspID: msp1,
			Peers: []*Peer{
				{
					PeerID: peer1,
					Apps: []*App{
						{
							AppName: app1, Version: v1,
							Components: []*Component{
								{Name: comp1, Version: v1, Config: configData},
							},
						},
					},
				},
			},
		}
		err := cfg.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "field [Format] is required")
	})
}
