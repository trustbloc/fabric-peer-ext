/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package policy

import (
	"time"

	"github.com/hyperledger/fabric/protos/common"
	"github.com/pkg/errors"
)

// ValidateConfig validates the Transient Data Collection configuration
func ValidateConfig(config *common.StaticCollectionConfig) error {
	if config.RequiredPeerCount <= 0 {
		return errors.Errorf("collection-name: %s -- required peer count must be greater than 0", config.Name)
	}

	if config.RequiredPeerCount > config.MaximumPeerCount {
		return errors.Errorf("collection-name: %s -- maximum peer count (%d) must be greater than or equal to required peer count (%d)", config.Name, config.MaximumPeerCount, config.RequiredPeerCount)
	}
	if config.TimeToLive == "" {
		return errors.Errorf("collection-name: %s -- time to live must be specified", config.Name)
	}

	if config.BlockToLive != 0 {
		return errors.Errorf("collection-name: %s -- block-to-live not supported", config.Name)
	}

	_, err := time.ParseDuration(config.TimeToLive)
	if err != nil {
		return errors.Errorf("collection-name: %s -- invalid time format for time to live: %s", config.Name, err)
	}

	return nil
}
