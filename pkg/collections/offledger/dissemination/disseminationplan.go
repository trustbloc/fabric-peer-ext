/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dissemination

import (
	protobuf "github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/ledger/rwset"
	"github.com/hyperledger/fabric-protos-go/ledger/rwset/kvrwset"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/extensions/collections/api/dissemination"
	gossipapi "github.com/hyperledger/fabric/gossip/api"
	gcommon "github.com/hyperledger/fabric/gossip/common"
	gdiscovery "github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/gossip"
	"github.com/hyperledger/fabric/gossip/protoext"
	"github.com/pkg/errors"
	viper "github.com/spf13/viper2015"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/dcas"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/implicitpolicy"
)

type gossipAdapter interface {
	PeersOfChannel(id gcommon.ChannelID) []gdiscovery.NetworkMember
	SelfMembershipInfo() gdiscovery.NetworkMember
	IdentityInfo() gossipapi.PeerIdentitySet
}

// ComputeDisseminationPlan returns the dissemination plan for off ledger data
func ComputeDisseminationPlan(
	channelID, ns string,
	rwSet *rwset.CollectionPvtReadWriteSet,
	collConfig *pb.StaticCollectionConfig,
	colAP privdata.CollectionAccessPolicy,
	pvtDataMsg *protoext.SignedGossipMessage,
	gossipAdapter gossipAdapter) ([]*dissemination.Plan, bool, error) {
	logger.Debugf("Computing dissemination plan for [%s:%s]", ns, rwSet.CollectionName)

	kvRwSet, err := unmarshalKVRWSet(rwSet.Rwset)
	if err != nil {
		return nil, true, errors.WithMessage(err, "error unmarshalling KV read/write set")
	}

	err = validateAll(collConfig.Type, kvRwSet)
	if err != nil {
		return nil, true, errors.WithMessagef(err, "one or more keys did not validate for collection [%s:%s]", ns, rwSet.CollectionName)
	}

	localMSP, err := LocalMSPProvider.LocalMSP()
	if err != nil {
		return nil, true, err
	}

	peers := New(channelID, ns, rwSet.CollectionName, implicitpolicy.NewResolver(localMSP, colAP), gossipAdapter).resolvePeersForDissemination().Remote()

	logger.Debugf("Peers for dissemination of collection [%s:%s]: %s", ns, rwSet.CollectionName, peers)

	routingFilter := func(member gdiscovery.NetworkMember) bool {
		if peers.ContainsPeer(member.Endpoint) {
			logger.Debugf("Including peer [%s] for dissemination of [%s:%s]", member.Endpoint, ns, rwSet.CollectionName)
			return true
		}

		logger.Debugf("Not including peer [%s] for dissemination of [%s:%s]", member.Endpoint, ns, rwSet.CollectionName)
		return false
	}

	sc := gossip.SendCriteria{
		Timeout:    viper.GetDuration("peer.gossip.pvtData.pushAckTimeout"),
		Channel:    gcommon.ChannelID(channelID),
		MaxPeers:   len(peers),
		MinAck:     colAP.RequiredPeerCount(),
		IsEligible: routingFilter,
	}

	return []*dissemination.Plan{{
		Criteria: sc,
		Msg:      pvtDataMsg,
	}}, true, nil
}

func validateAll(collType pb.CollectionType, kvRWSet *kvrwset.KVRWSet) error {
	for _, ws := range kvRWSet.Writes {
		if err := validate(collType, ws); err != nil {
			return err
		}
	}
	return nil
}

func validate(collType pb.CollectionType, ws *kvrwset.KVWrite) error {
	if collType == pb.CollectionType_COL_DCAS && ws.Value != nil {
		expectedKey, _, err := dcas.GetCASKeyAndValueBase58(ws.Value)
		if err != nil {
			return err
		}
		if ws.Key != expectedKey {
			return errors.Errorf("invalid CAS key [%s] - the key should be the hash of the value [%s]", ws.Key, expectedKey)
		}
	}
	return nil
}

// unmarshalKVRWSet unmarshals the given KV rw-set bytes. This variable may be overridden by unit tests.
var unmarshalKVRWSet = func(bytes []byte) (*kvrwset.KVRWSet, error) {
	kvRwSet := &kvrwset.KVRWSet{}
	err := protobuf.Unmarshal(bytes, kvRwSet)
	if err != nil {
		return nil, err
	}
	return kvRwSet, nil
}
