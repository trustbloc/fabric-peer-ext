/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"github.com/hyperledger/fabric/core/common/privdata"
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/pkg/errors"
)

// MockSupport is a holder of policy, config and error
type MockSupport struct {
	CollPolicy  privdata.CollectionAccessPolicy
	CollConfigs []*cb.StaticCollectionConfig
	Err         error
	Publisher   *MockBlockPublisher
}

// NewMockSupport returns a new MockSupport
func NewMockSupport() *MockSupport {
	return &MockSupport{
		Publisher: NewBlockPublisher(),
	}
}

// CollectionPolicy sets the collection access policy for the given collection
func (s *MockSupport) CollectionPolicy(collPolicy privdata.CollectionAccessPolicy) *MockSupport {
	s.CollPolicy = collPolicy
	return s
}

// CollectionConfig sets the collection config for the given collection
func (s *MockSupport) CollectionConfig(collConfig *cb.StaticCollectionConfig) *MockSupport {
	s.CollConfigs = append(s.CollConfigs, collConfig)
	return s
}

// Policy returns the collection access policy for the given collection
func (s *MockSupport) Policy(channelID, ns, coll string) (privdata.CollectionAccessPolicy, error) {
	return s.CollPolicy, s.Err
}

// Config returns the collection config for the given collection
func (s *MockSupport) Config(channelID, ns, coll string) (*cb.StaticCollectionConfig, error) {
	for _, config := range s.CollConfigs {
		if config.Name == coll {
			return config, nil
		}
	}
	return nil, errors.Errorf("config not found for collection: %s", coll)
}

// BlockPublisher returns a mock block publisher for the given channel
func (s *MockSupport) BlockPublisher(channelID string) gossipapi.BlockPublisher {
	return s.Publisher
}
