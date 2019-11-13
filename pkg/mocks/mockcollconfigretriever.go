/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/pkg/errors"
)

// CollectionConfigRetriever is a mock collection config retriever
type CollectionConfigRetriever struct {
	policy  privdata.CollectionAccessPolicy
	configs []*common.StaticCollectionConfig
	err     error
}

// NewCollectionConfigRetriever returns a mock collection config retriever
func NewCollectionConfigRetriever() *CollectionConfigRetriever {
	return &CollectionConfigRetriever{}
}

// WithError injects an error
func (s *CollectionConfigRetriever) WithError(err error) *CollectionConfigRetriever {
	s.err = err
	return s
}

// WithCollectionPolicy sets the collection access policy for the given collection
func (s *CollectionConfigRetriever) WithCollectionPolicy(collPolicy privdata.CollectionAccessPolicy) *CollectionConfigRetriever {
	s.policy = collPolicy
	return s
}

// WithCollectionConfig sets the collection config for the given collection
func (s *CollectionConfigRetriever) WithCollectionConfig(collConfig *common.StaticCollectionConfig) *CollectionConfigRetriever {
	s.configs = append(s.configs, collConfig)
	return s
}

// Policy returns the collection access policy for the given collection
func (s *CollectionConfigRetriever) Policy(ns, coll string) (privdata.CollectionAccessPolicy, error) {
	return s.policy, s.err
}

// Config returns the collection config for the given collection
func (s *CollectionConfigRetriever) Config(ns, coll string) (*common.StaticCollectionConfig, error) {
	for _, config := range s.configs {
		if config.Name == coll {
			return config, nil
		}
	}
	return nil, errors.Errorf("config not found for collection: %s", coll)
}
