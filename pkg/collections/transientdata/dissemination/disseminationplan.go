/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dissemination

import (
	protobuf "github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/extensions/collections/api/dissemination"
	gossipapi "github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/common"
	gdiscovery "github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/gossip"
	"github.com/hyperledger/fabric/gossip/protoext"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/discovery"
)

type gossipAdapter interface {
	PeersOfChannel(common.ChainID) []gdiscovery.NetworkMember
	SelfMembershipInfo() gdiscovery.NetworkMember
	IdentityInfo() gossipapi.PeerIdentitySet
}

// ComputeDisseminationPlan returns the dissemination plan for transient data
func ComputeDisseminationPlan(
	channelID, ns string,
	rwSet *rwset.CollectionPvtReadWriteSet,
	colAP privdata.CollectionAccessPolicy,
	pvtDataMsg *protoext.SignedGossipMessage,
	gossipAdapter gossipAdapter) ([]*dissemination.Plan, bool, error) {
	logger.Debugf("Computing transient data dissemination plan for [%s:%s]", ns, rwSet.CollectionName)

	disseminator := New(channelID, ns, rwSet.CollectionName, colAP, gossipAdapter)

	kvRwSet := &kvrwset.KVRWSet{}
	if err := protobuf.Unmarshal(rwSet.Rwset, kvRwSet); err != nil {
		return nil, true, errors.WithMessage(err, "error unmarshalling KV read/write set for transient data")
	}

	var endorsers discovery.PeerGroup
	for _, kvWrite := range kvRwSet.Writes {
		if kvWrite.IsDelete {
			continue
		}
		endorsersForKey, err := disseminator.ResolveEndorsers(kvWrite.Key)
		if err != nil {
			return nil, true, errors.WithMessage(err, "error resolving endorsers for transient data")
		}

		logger.Debugf("Endorsers for key [%s:%s:%s]: %s", ns, rwSet.CollectionName, kvWrite.Key, endorsersForKey)

		for _, endorser := range endorsersForKey {
			if endorser.Local {
				logger.Debugf("Not adding local endorser for key [%s:%s:%s]", ns, rwSet.CollectionName, kvWrite.Key)
				continue
			}
			endorsers = discovery.Merge(endorsers, endorser)
		}
	}

	logger.Debugf("Endorsers for collection [%s:%s]: %s", ns, rwSet.CollectionName, endorsers)

	routingFilter := func(member gdiscovery.NetworkMember) bool {
		if endorsers.ContainsPeer(member.Endpoint) {
			logger.Debugf("Peer [%s] is an endorser for [%s:%s]", member.Endpoint, ns, rwSet.CollectionName)
			return true
		}

		logger.Debugf("Peer [%s] is NOT an endorser for [%s:%s]", member.Endpoint, ns, rwSet.CollectionName)
		return false
	}

	sc := gossip.SendCriteria{
		Timeout:    viper.GetDuration("peer.gossip.pvtData.pushAckTimeout"),
		Channel:    common.ChainID(channelID),
		MaxPeers:   colAP.MaximumPeerCount(),
		MinAck:     colAP.RequiredPeerCount(),
		IsEligible: routingFilter,
	}

	return []*dissemination.Plan{{
		Criteria: sc,
		Msg:      pvtDataMsg,
	}}, true, nil
}
