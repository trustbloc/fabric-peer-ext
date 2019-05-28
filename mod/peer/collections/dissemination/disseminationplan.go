/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dissemination

import (
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/extensions/collections/api/dissemination"
	gossipapi "github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/protoext"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/pkg/errors"
	oldissemination "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/dissemination"
	tdissemination "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/dissemination"
)

type gossipAdapter interface {
	PeersOfChannel(common.ChainID) []discovery.NetworkMember
	SelfMembershipInfo() discovery.NetworkMember
	IdentityInfo() gossipapi.PeerIdentitySet
}

var computeTransientDataDisseminationPlan = func(
	channelID, ns string,
	rwSet *rwset.CollectionPvtReadWriteSet,
	colAP privdata.CollectionAccessPolicy,
	pvtDataMsg *protoext.SignedGossipMessage,
	gossipAdapter gossipAdapter) ([]*dissemination.Plan, bool, error) {
	return tdissemination.ComputeDisseminationPlan(channelID, ns, rwSet, colAP, pvtDataMsg, gossipAdapter)
}

var computeOffLedgerDisseminationPlan = func(
	channelID, ns string,
	rwSet *rwset.CollectionPvtReadWriteSet,
	collConfig *cb.StaticCollectionConfig,
	colAP privdata.CollectionAccessPolicy,
	pvtDataMsg *protoext.SignedGossipMessage,
	gossipAdapter gossipAdapter) ([]*dissemination.Plan, bool, error) {
	return oldissemination.ComputeDisseminationPlan(channelID, ns, rwSet, collConfig, colAP, pvtDataMsg, gossipAdapter)
}

// ComputeDisseminationPlan returns the dissemination plan for various collection types
func ComputeDisseminationPlan(
	channelID, ns string,
	rwSet *rwset.CollectionPvtReadWriteSet,
	colCP *cb.CollectionConfig,
	colAP privdata.CollectionAccessPolicy,
	pvtDataMsg *protoext.SignedGossipMessage,
	gossipAdapter gossipAdapter) ([]*dissemination.Plan, bool, error) {

	collConfig := colCP.GetStaticCollectionConfig()
	if collConfig == nil {
		return nil, false, errors.New("static collection config not defined")
	}

	switch collConfig.Type {
	case cb.CollectionType_COL_TRANSIENT:
		return computeTransientDataDisseminationPlan(channelID, ns, rwSet, colAP, pvtDataMsg, gossipAdapter)
	case cb.CollectionType_COL_DCAS:
		fallthrough
	case cb.CollectionType_COL_OFFLEDGER:
		return computeOffLedgerDisseminationPlan(channelID, ns, rwSet, collConfig, colAP, pvtDataMsg, gossipAdapter)
	default:
		return nil, false, errors.Errorf("unsupported collection type: [%s]", collConfig.Type)
	}
}
