/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validation

import (
	"testing"

	viper "github.com/spf13/viper2015"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
	vmocks "github.com/trustbloc/fabric-peer-ext/pkg/validation/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/validation/validator"
)

//go:generate counterfeiter -o mocks/channelresources.gen.go --fake-name ChannelResources . ChannelResources
//go:generate counterfeiter -o mocks/semaphore.gen.go --fake-name Semaphore . Semaphore

const (
	channelID = "channel1"
)

// Ensure the roles are initialized
var _ = roles.GetRoles()

func TestNewValidator(t *testing.T) {
	t.Run("Distributed validation disabled", func(t *testing.T) {
		reset := setViper(config.ConfDistributedValidationEnabled, false)
		defer reset()

		providers := &validator.Providers{
			Gossip: &mocks.GossipProvider{},
			Idp:    &mocks.IdentityDeserializerProvider{},
		}

		p := validator.NewProvider(providers)
		require.NotNil(t, p)

		v := NewValidator(channelID, &vmocks.Semaphore{}, &vmocks.ChannelResources{}, nil, nil, nil, nil, nil, nil)
		require.NotNil(t, v)
	})

	t.Run("Distributed validation enabled", func(t *testing.T) {
		reset := setViper(config.ConfDistributedValidationEnabled, true)
		defer reset()

		providers := &validator.Providers{
			Gossip: &mocks.GossipProvider{},
			Idp:    &mocks.IdentityDeserializerProvider{},
		}

		p := validator.NewProvider(providers)
		require.NotNil(t, p)

		t.Run("Clustered", func(t *testing.T) {
			reset := roles.SetRole(roles.EndorserRole)
			defer reset()

			v := NewValidator(channelID, &vmocks.Semaphore{}, &vmocks.ChannelResources{}, nil, nil, nil, nil, nil, nil)
			require.NotNil(t, v)
		})

		t.Run("Not clustered", func(t *testing.T) {
			require.Panics(t, func() {
				v := NewValidator(channelID, &vmocks.Semaphore{}, &vmocks.ChannelResources{}, nil, nil, nil, nil, nil, nil)
				require.NotNil(t, v)
			})
		})
	})
}

func setViper(key string, value interface{}) (reset func()) {
	oldVal := viper.Get(key)
	viper.Set(key, value)

	return func() { viper.Set(config.ConfDistributedValidationEnabled, oldVal) }
}
