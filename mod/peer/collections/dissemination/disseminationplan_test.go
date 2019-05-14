/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dissemination

import (
	"testing"

	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/extensions/collections/api/dissemination"
	"github.com/hyperledger/fabric/gossip/protoext"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/stretchr/testify/assert"
)

func TestDisseminationPlan(t *testing.T) {
	const (
		channelID = "testchannel"
		ns        = "ns1"
	)

	computeTransientDataDisseminationPlan = func(
		channelID, ns string,
		rwSet *rwset.CollectionPvtReadWriteSet,
		colAP privdata.CollectionAccessPolicy,
		pvtDataMsg *protoext.SignedGossipMessage,
		gossipAdapter gossipAdapter) ([]*dissemination.Plan, bool, error) {
		return nil, false, nil
	}

	t.Run("Empty config", func(t *testing.T) {
		colConfig1 := &common.CollectionConfig{}
		_, _, err := ComputeDisseminationPlan(
			channelID, ns, nil, colConfig1, nil, nil, nil)
		assert.EqualError(t, err, "static collection config not defined")
	})

	t.Run("Unknown config", func(t *testing.T) {
		colConfig2 := &common.CollectionConfig{
			Payload: &common.CollectionConfig_StaticCollectionConfig{
				StaticCollectionConfig: &common.StaticCollectionConfig{},
			},
		}
		_, _, err := ComputeDisseminationPlan(
			channelID, ns, nil, colConfig2, nil, nil, nil)
		assert.EqualError(t, err, "unsupported collection type: [COL_UNKNOWN]")
	})

	t.Run("Transient Data config", func(t *testing.T) {
		transientConfig := &common.CollectionConfig{
			Payload: &common.CollectionConfig_StaticCollectionConfig{
				StaticCollectionConfig: &common.StaticCollectionConfig{
					Type: common.CollectionType_COL_TRANSIENT,
				},
			},
		}
		_, _, err := ComputeDisseminationPlan(
			channelID, ns, nil, transientConfig, nil, nil, nil)
		assert.NoError(t, err)
	})
}
