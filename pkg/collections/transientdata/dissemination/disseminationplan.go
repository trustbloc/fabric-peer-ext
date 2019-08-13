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
	"github.com/hyperledger/fabric/gossip/util"
	cb "github.com/hyperledger/fabric/protos/common"
	protosgossip "github.com/hyperledger/fabric/protos/gossip"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
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

	keysByEndorser, err := getKeysByEndorser(ns, rwSet.CollectionName, kvRwSet, disseminator)
	if err != nil {
		return nil, true, err
	}

	pvtPayload := pvtDataMsg.Content.(*protosgossip.GossipMessage_PrivateData).PrivateData.Payload

	// Construct one dissemination plan per endorser, which will include all keys that the endorser should store
	var plans []*dissemination.Plan
	for e, keys := range keysByEndorser {
		endpoint := e
		logger.Debugf("Keys for endorser [%s] in collection [%s:%s]: %s", endpoint, ns, rwSet.CollectionName, keys)

		rwSetForKeys, err := newRWSetForKeys(ns, rwSet, keys)
		if err != nil {
			return nil, true, err
		}

		plan, err := computeDisseminationPlanForEndorser(channelID, e, ns, rwSetForKeys, pvtPayload, keys)
		if err != nil {
			return nil, true, err
		}

		logger.Debugf("Adding dissemination plan for endorser [%s] and keys [%s:%s]-%s", endpoint, ns, rwSet.CollectionName, keys)
		plans = append(plans, plan)
	}

	logger.Debugf("Returning [%d] dissemination plan(s) for collection [%s:%s] in Tx [%s]", len(plans), ns, rwSet.CollectionName, pvtPayload.TxId)
	return plans, true, nil
}

func getKeysByEndorser(ns, coll string, kvRwSet *kvrwset.KVRWSet, disseminator *Disseminator) (map[string][]string, error) {
	keysByEndorser := make(map[string][]string)
	for _, kvWrite := range kvRwSet.Writes {
		if kvWrite.IsDelete {
			continue
		}
		endorsersForKey, err := disseminator.ResolveEndorsers(kvWrite.Key)
		if err != nil {
			return nil, errors.WithMessage(err, "error resolving endorsers for transient data")
		}

		logger.Debugf("Endorsers for key [%s:%s:%s]: %s", ns, coll, kvWrite.Key, endorsersForKey)

		for _, endorser := range endorsersForKey {
			if endorser.Local {
				logger.Debugf("Not adding local endorser for key [%s:%s:%s]", ns, coll, kvWrite.Key)
				continue
			}
			keysByEndorser[endorser.Endpoint] = append(keysByEndorser[endorser.Endpoint], kvWrite.Key)
		}
	}
	return keysByEndorser, nil
}

func computeDisseminationPlanForEndorser(channelID, endpoint, ns string, rwSet *rwset.CollectionPvtReadWriteSet, pvtPayload *protosgossip.PrivatePayload, keys []string) (*dissemination.Plan, error) {
	msgForKey, err := createPrivateDataMessage(
		channelID, pvtPayload.TxId, ns, rwSet,
		pvtPayload.CollectionConfigs, pvtPayload.PrivateSimHeight)
	if err != nil {
		return nil, err
	}

	routingFilter := func(member gdiscovery.NetworkMember) bool {
		if endpoint == member.Endpoint {
			logger.Debugf("Peer [%s] is an endorser for key(s) [%s:%s]-%s", member.Endpoint, ns, rwSet.CollectionName, keys)
			return true
		}
		logger.Debugf("Peer [%s] is NOT an endorser for key(s) [%s:%s]-%s", member.Endpoint, ns, rwSet.CollectionName, keys)
		return false
	}

	// Since we are pushing data to each endorser individually, MaxPeers and MinAck are both set to 1
	sc := gossip.SendCriteria{
		Timeout:    viper.GetDuration("peer.gossip.pvtData.pushAckTimeout"),
		Channel:    common.ChainID(channelID),
		MaxPeers:   1,
		MinAck:     1,
		IsEligible: routingFilter,
	}
	return &dissemination.Plan{Criteria: sc, Msg: msgForKey}, nil
}

func newRWSetForKeys(ns string, rwSet *rwset.CollectionPvtReadWriteSet, keys []string) (*rwset.CollectionPvtReadWriteSet, error) {
	logger.Debugf("Creating a new collection Pvt rw-set for keys %s in collection [%s:%s]", keys, ns, rwSet.CollectionName)

	kvRWSet, err := unmarshalKVRWSet(rwSet.Rwset)
	if err != nil {
		return nil, err
	}

	var kvWrites []*kvrwset.KVWrite
	for _, kvWrite := range kvRWSet.Writes {
		if containsKey(keys, kvWrite.Key) {
			logger.Debugf("Adding KV-write for key [%s:%s:%s]", ns, rwSet.CollectionName, kvWrite.Key)
			kvWrites = append(kvWrites, kvWrite)
		}
	}

	rwSetBytes, err := marshalKVRWSet(&kvrwset.KVRWSet{Writes: kvWrites})
	if err != nil {
		return nil, err
	}

	return &rwset.CollectionPvtReadWriteSet{
		CollectionName: rwSet.CollectionName,
		Rwset:          rwSetBytes,
	}, nil
}

func containsKey(keys []string, key string) bool {
	for _, k := range keys {
		if k == key {
			return true
		}
	}
	return false
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

// marshalKVRWSet marshals the given KV rw-set bytes. This variable may be overridden by unit tests.
var marshalKVRWSet = func(rwSet *kvrwset.KVRWSet) ([]byte, error) {
	return protobuf.Marshal(rwSet)
}

func createPrivateDataMessage(
	channelID, txID, namespace string,
	collRWSet *rwset.CollectionPvtReadWriteSet,
	ccp *cb.CollectionConfigPackage, blkHt uint64) (*protoext.SignedGossipMessage, error) {
	msg := &protosgossip.GossipMessage{
		Channel: []byte(channelID),
		Nonce:   util.RandomUInt64(),
		Tag:     protosgossip.GossipMessage_CHAN_ONLY,
		Content: &protosgossip.GossipMessage_PrivateData{
			PrivateData: &protosgossip.PrivateDataMessage{
				Payload: &protosgossip.PrivatePayload{
					Namespace:         namespace,
					CollectionName:    collRWSet.CollectionName,
					TxId:              txID,
					PrivateRwset:      collRWSet.Rwset,
					PrivateSimHeight:  blkHt,
					CollectionConfigs: ccp,
				},
			},
		},
	}
	pvtDataMsg, err := protoext.NoopSign(msg)
	if err != nil {
		return nil, err
	}
	return pvtDataMsg, nil
}
