/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package policy

import (
	"fmt"

	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/pkg/errors"
	olpolicy "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/policy"
	tdatapolicy "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/policy"
)

// Validator is a collection policy validator
type Validator struct {
}

// NewValidator returns a new collection policy validator
func NewValidator() *Validator {
	return &Validator{}
}

// Validate validates various collection config types
func (v *Validator) Validate(collConfig *pb.CollectionConfig) error {
	config := collConfig.GetStaticCollectionConfig()
	if config == nil {
		return errors.New("unknown collection configuration type")
	}

	switch config.Type {
	case pb.CollectionType_COL_TRANSIENT:
		return tdatapolicy.ValidateConfig(config)
	case pb.CollectionType_COL_DCAS:
		fallthrough
	case pb.CollectionType_COL_OFFLEDGER:
		return olpolicy.ValidateConfig(config)
	default:
		// Nothing to do
		return nil
	}
}

// ValidateNewCollectionConfigsAgainstOld validates updated collection configs
func (v *Validator) ValidateNewCollectionConfigsAgainstOld(newCollectionConfigs []*pb.CollectionConfig, oldCollectionConfigs []*pb.CollectionConfig,
) error {
	newCollectionsMap := make(map[string]*pb.StaticCollectionConfig, len(newCollectionConfigs))
	for _, newCollectionConfig := range newCollectionConfigs {
		newCollection := newCollectionConfig.GetStaticCollectionConfig()
		newCollectionsMap[newCollection.GetName()] = newCollection
	}

	for _, oldCollConfig := range oldCollectionConfigs {
		oldColl := oldCollConfig.GetStaticCollectionConfig()
		if oldColl == nil {
			// This shouldn't happen since we've already gone through the basic validation
			return errors.New("invalid collection")
		}
		newColl, ok := newCollectionsMap[oldColl.Name]
		if !ok {
			continue
		}

		newCollType := getCollType(newColl)
		oldCollType := getCollType(oldColl)
		if newCollType != oldCollType {
			return fmt.Errorf("collection-name: %s -- attempt to change collection type from [%s] to [%s]", oldColl.Name, oldCollType, newCollType)
		}
	}
	return nil
}

func getCollType(config *pb.StaticCollectionConfig) pb.CollectionType {
	switch config.Type {
	case pb.CollectionType_COL_TRANSIENT:
		return pb.CollectionType_COL_TRANSIENT
	case pb.CollectionType_COL_OFFLEDGER:
		return pb.CollectionType_COL_OFFLEDGER
	case pb.CollectionType_COL_DCAS:
		return pb.CollectionType_COL_DCAS
	case pb.CollectionType_COL_PRIVATE:
		fallthrough
	default:
		return pb.CollectionType_COL_PRIVATE
	}
}
