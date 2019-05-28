/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package policy

import (
	"fmt"

	"github.com/hyperledger/fabric/protos/common"
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
func (v *Validator) Validate(collConfig *common.CollectionConfig) error {
	config := collConfig.GetStaticCollectionConfig()
	if config == nil {
		return errors.New("unknown collection configuration type")
	}

	switch config.Type {
	case common.CollectionType_COL_TRANSIENT:
		return tdatapolicy.ValidateConfig(config)
	case common.CollectionType_COL_DCAS:
		fallthrough
	case common.CollectionType_COL_OFFLEDGER:
		return olpolicy.ValidateConfig(config)
	default:
		// Nothing to do
		return nil
	}
}

// ValidateNewCollectionConfigsAgainstOld validates updated collection configs
func (v *Validator) ValidateNewCollectionConfigsAgainstOld(newCollectionConfigs []*common.CollectionConfig, oldCollectionConfigs []*common.CollectionConfig,
) error {
	newCollectionsMap := make(map[string]*common.StaticCollectionConfig, len(newCollectionConfigs))
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

func getCollType(config *common.StaticCollectionConfig) common.CollectionType {
	switch config.Type {
	case common.CollectionType_COL_TRANSIENT:
		return common.CollectionType_COL_TRANSIENT
	case common.CollectionType_COL_OFFLEDGER:
		return common.CollectionType_COL_OFFLEDGER
	case common.CollectionType_COL_DCAS:
		return common.CollectionType_COL_DCAS
	case common.CollectionType_COL_PRIVATE:
		fallthrough
	default:
		return common.CollectionType_COL_PRIVATE
	}
}
