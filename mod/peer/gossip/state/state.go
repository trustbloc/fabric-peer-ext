/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package state

import (
	"strings"

	pb "github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/ledger/cceventmgmt"
	"github.com/hyperledger/fabric/extensions/gossip/api"
	"github.com/hyperledger/fabric/extensions/roles"
	common2 "github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/protoext"
	"github.com/hyperledger/fabric/gossip/util"
	"github.com/hyperledger/fabric/protos/common"
	proto "github.com/hyperledger/fabric/protos/gossip"

	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	"github.com/pkg/errors"
)

var logger = util.GetLogger(util.StateLogger, "")

const (
	//collectionSeparator kvwrite collection key separator
	collectionSeparator = "~"
)

//GossipStateProviderExtension extends GossipStateProvider features
type GossipStateProviderExtension interface {

	//HandleStateRequest can used to extend given request handle
	HandleStateRequest(func(msg protoext.ReceivedMessage)) func(msg protoext.ReceivedMessage)

	//AntiEntropy can be used to extend Anti Entropy features
	AntiEntropy(func()) func()

	//Predicate can used to override existing predicate to filter peers to be asked for blocks
	Predicate(func(peer discovery.NetworkMember) bool) func(peer discovery.NetworkMember) bool

	//AddPayload can used to extend given add payload handle
	AddPayload(func(payload *proto.Payload, blockingMode bool) error) func(payload *proto.Payload, blockingMode bool) error

	//StoreBlock  can used to extend given store block handle
	StoreBlock(func(block *common.Block, pvtData util.PvtDataCollections) error) func(block *common.Block, pvtData util.PvtDataCollections) error
}

// GossipServiceMediator aggregated adapter interface to compound basic mediator services
// required by state transfer into single struct
type GossipServiceMediator interface {
	// VerifyBlock returns nil if the block is properly signed, and the claimed seqNum is the
	// sequence number that the block's header contains.
	// else returns error
	VerifyBlock(chainID common2.ChainID, seqNum uint64, signedBlock []byte) error

	// PeersOfChannel returns the NetworkMembers considered alive
	// and also subscribed to the channel given
	PeersOfChannel(common2.ChainID) []discovery.NetworkMember

	// Gossip sends a message to other peers to the network
	Gossip(msg *proto.GossipMessage)
}

//AddBlockHandler handles state update in gossip
func AddBlockHandler(publisher api.BlockPublisher) {
	publisher.AddWriteHandler(func(blockNum uint64, channelID, txID, namespace string, kvWrite *kvrwset.KVWrite) error {
		if namespace != "lscc" {
			return nil
		}
		return handleStateUpdate(kvWrite, channelID)
	})
}

//NewGossipStateProviderExtension returns new GossipStateProvider Extension implementation
func NewGossipStateProviderExtension(chainID string, mediator GossipServiceMediator) GossipStateProviderExtension {
	return &gossipStateProviderExtension{chainID, mediator}
}

type gossipStateProviderExtension struct {
	chainID  string
	mediator GossipServiceMediator
}

func (s *gossipStateProviderExtension) HandleStateRequest(handle func(msg protoext.ReceivedMessage)) func(msg protoext.ReceivedMessage) {
	return func(msg protoext.ReceivedMessage) {
		if roles.IsEndorser() {
			handle(msg)
		}
	}
}

func (s *gossipStateProviderExtension) AntiEntropy(handle func()) func() {
	return func() {
		if roles.IsCommitter() {
			handle()
		}
	}
}

func (s *gossipStateProviderExtension) Predicate(handle func(peer discovery.NetworkMember) bool) func(peer discovery.NetworkMember) bool {
	return func(peer discovery.NetworkMember) bool {
		canPredicate := handle(peer)
		if canPredicate {
			if len(peer.Properties.Roles) == 0 || roles.HasEndorserRole(peer.Properties.Roles) {
				logger.Debugf("Choosing [%s] since it's an endorser", peer.Endpoint)
				return true
			}
			logger.Debugf("Not choosing [%s] since it's not an endorser", peer.Endpoint)
			return false
		}
		return false
	}
}

func (s *gossipStateProviderExtension) AddPayload(handle func(payload *proto.Payload, blockingMode bool) error) func(payload *proto.Payload, blockingMode bool) error {

	return func(payload *proto.Payload, blockingMode bool) error {
		if roles.IsCommitter() {
			return handle(payload, blockingMode)
		}
		return nil
	}
}

func (s *gossipStateProviderExtension) StoreBlock(handle func(block *common.Block, pvtData util.PvtDataCollections) error) func(block *common.Block, pvtData util.PvtDataCollections) error {

	return func(block *common.Block, pvtData util.PvtDataCollections) error {
		if roles.IsCommitter() {
			// Commit block with available private transactions
			if err := handle(block, pvtData); err != nil {
				logger.Errorf("Got error while committing(%+v)", errors.WithStack(err))
				return err
			}

			// Gossip messages with other nodes in my org
			s.gossipBlock(block)

		}
		return nil
	}
}

func (s *gossipStateProviderExtension) gossipBlock(block *common.Block) {
	blockNum := block.Header.Number

	marshaledBlock, err := pb.Marshal(block)
	if err != nil {
		logger.Errorf("[%s] Error serializing block with sequence number %d, due to %s", s.chainID, blockNum, err)
	}
	if err := s.mediator.VerifyBlock(common2.ChainID(s.chainID), blockNum, marshaledBlock); err != nil {
		logger.Errorf("[%s] Error verifying block with sequnce number %d, due to %s", s.chainID, blockNum, err)
	}

	numberOfPeers := len(s.mediator.PeersOfChannel(common2.ChainID(s.chainID)))

	// Create payload with a block received
	payload := &proto.Payload{
		Data:   marshaledBlock,
		SeqNum: blockNum,
	}

	// Use payload to create gossip message
	gossipMsg := createGossipMsg(s.chainID, payload)

	logger.Debugf("[%s] Gossiping block [%d], number of peers [%d]", s.chainID, blockNum, numberOfPeers)
	s.mediator.Gossip(gossipMsg)
}

func createGossipMsg(chainID string, payload *proto.Payload) *proto.GossipMessage {
	gossipMsg := &proto.GossipMessage{
		Nonce:   0,
		Tag:     proto.GossipMessage_CHAN_AND_ORG,
		Channel: []byte(chainID),
		Content: &proto.GossipMessage_DataMsg{
			DataMsg: &proto.DataMessage{
				Payload: payload,
			},
		},
	}
	return gossipMsg
}

func handleStateUpdate(kvWrite *kvrwset.KVWrite, channelID string) error {
	// There are LSCC entries for the chaincode and for the chaincode collections.
	// We need to ignore changes to chaincode collections, and handle changes to chaincode
	// We can detect collections based on the presence of a CollectionSeparator, which never exists in chaincode names
	if isCollectionConfigKey(kvWrite.Key) {
		return nil
	}
	// Ignore delete events
	if kvWrite.IsDelete {
		return nil
	}

	// Chaincode instantiate/upgrade is not logged on committing peer anywhere else.  This is a good place to log it.
	logger.Debugf("Handling LSCC state update for chaincode [%s] on channel [%s]", kvWrite.Key, channelID)
	chaincodeData := &ccprovider.ChaincodeData{}
	if err := pb.Unmarshal(kvWrite.Value, chaincodeData); err != nil {
		return errors.Errorf("Unmarshalling ChaincodeQueryResponse failed, error %s", err)
	}

	chaincodeDefs := []*cceventmgmt.ChaincodeDefinition{}
	chaincodeDefs = append(chaincodeDefs, &cceventmgmt.ChaincodeDefinition{Name: chaincodeData.CCName(), Version: chaincodeData.CCVersion(), Hash: chaincodeData.Hash()})

	err := cceventmgmt.GetMgr().HandleChaincodeDeploy(channelID, chaincodeDefs)
	if err != nil {
		return err
	}

	cceventmgmt.GetMgr().ChaincodeDeployDone(channelID)

	return nil
}

// isCollectionConfigKey detects if a key is a collection key
func isCollectionConfigKey(key string) bool {
	return strings.Contains(key, collectionSeparator)
}
