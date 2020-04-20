/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package state

import (
	pb "github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	proto "github.com/hyperledger/fabric-protos-go/gossip"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/common/flogging"
	ledgerUtil "github.com/hyperledger/fabric/core/ledger/util"
	common2 "github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/protoext"
	"github.com/hyperledger/fabric/gossip/util"
	"github.com/pkg/errors"

	"github.com/hyperledger/fabric/extensions/gossip/api"
	"github.com/hyperledger/fabric/extensions/gossip/blockpublisher"
	"github.com/hyperledger/fabric/extensions/roles"
)

var logger = flogging.MustGetLogger("ext_gossip_state")

//GossipStateProviderExtension extends GossipStateProvider features
type GossipStateProviderExtension interface {

	//HandleStateRequest can used to extend given request handle
	HandleStateRequest(func(msg protoext.ReceivedMessage)) func(msg protoext.ReceivedMessage)

	//Predicate can used to override existing predicate to filter peers to be asked for blocks
	Predicate(func(peer discovery.NetworkMember) bool) func(peer discovery.NetworkMember) bool

	//AddPayload can used to extend given add payload handle
	AddPayload(func(payload *proto.Payload, blockingMode bool) error) func(payload *proto.Payload, blockingMode bool) error

	//StoreBlock  can used to extend given store block handle
	StoreBlock(func(block *common.Block, pvtData util.PvtDataCollections) error) func(block *common.Block, pvtData util.PvtDataCollections) error

	//LedgerHeight can used to extend ledger height feature to get current ledger height
	LedgerHeight(func() (uint64, error)) func() (uint64, error)

	//RequestBlocksInRange can be used to extend given request blocks feature
	RequestBlocksInRange(func(start uint64, end uint64), func(payload *proto.Payload, blockingMode bool) error) func(start uint64, end uint64)
}

// GossipServiceMediator aggregated adapter interface to compound basic mediator services
// required by state transfer into single struct
type GossipServiceMediator interface {
	// VerifyBlock returns nil if the block is properly signed, and the claimed seqNum is the
	// sequence number that the block's header contains.
	// else returns error
	VerifyBlock(channelID common2.ChannelID, seqNum uint64, signedBlock *common.Block) error

	// PeersOfChannel returns the NetworkMembers considered alive
	// and also subscribed to the channel given
	PeersOfChannel(common2.ChannelID) []discovery.NetworkMember

	// Gossip sends a message to other peers to the network
	Gossip(msg *proto.GossipMessage)
}

//NewGossipStateProviderExtension returns new GossipStateProvider Extension implementation
func NewGossipStateProviderExtension(chainID string, mediator GossipServiceMediator, support *api.Support, blockingMode bool) GossipStateProviderExtension {
	return &gossipStateProviderExtension{chainID, mediator, support, blockingMode}
}

type gossipStateProviderExtension struct {
	chainID      string
	mediator     GossipServiceMediator
	support      *api.Support
	blockingMode bool
}

func (s *gossipStateProviderExtension) HandleStateRequest(handle func(msg protoext.ReceivedMessage)) func(msg protoext.ReceivedMessage) {
	return func(msg protoext.ReceivedMessage) {
		if roles.IsEndorser() {
			handle(msg)
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
		logger.Debugf("[%s] Got payload for sequence [%d]", s.chainID, payload.SeqNum)

		block := &common.Block{}
		if err := pb.Unmarshal(payload.Data, block); err != nil {
			return errors.WithMessagef(err, "error unmarshalling block with seqNum %d", payload.SeqNum)
		}

		if block.Data == nil || block.Header == nil {
			return errors.Errorf("block with sequence %d has no header (%v) or data (%v)", payload.SeqNum, block.Header, block.Data)
		}

		if !roles.IsCommitter() && !isBlockValidated(block) {
			logger.Debugf("[%s] I'm not a committer so will not add payload for unvalidated block [%d]", s.chainID, block.Header.Number)
			return nil
		}

		logger.Debugf("[%s] Adding payload for sequence [%d]", s.chainID, payload.SeqNum)

		return handle(payload, blockingMode)
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
			return nil
		}

		//in case of non-committer handle pre commit operations
		if isBlockValidated(block) {
			err := s.support.Ledger.CheckpointBlock(block)
			if err != nil {
				logger.Warning("Failed to update checkpoint info for cid[%s] block[%d]", s.chainID, block.Header.Number)
				return err
			}
			blockpublisher.ForChannel(s.chainID).Publish(block)
			return nil
		}

		logger.Warnf("[%s] I'm not a committer but received an unvalidated block [%d]", s.chainID, block.Header.Number)

		return nil
	}
}

func (s *gossipStateProviderExtension) gossipBlock(block *common.Block) {
	blockNum := block.Header.Number

	if err := s.mediator.VerifyBlock(common2.ChannelID(s.chainID), blockNum, block); err != nil {
		logger.Errorf("[%s] Error verifying block with sequence number %d, due to %s", s.chainID, blockNum, err)
		return
	}

	numberOfPeers := len(s.mediator.PeersOfChannel(common2.ChannelID(s.chainID)))

	marshaledBlock, err := pb.Marshal(block)
	if err != nil {
		logger.Errorf("[%s] Error serializing block with sequence number %d, due to %s", s.chainID, blockNum, err)
		return
	}

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

func (s *gossipStateProviderExtension) LedgerHeight(handle func() (uint64, error)) func() (uint64, error) {
	if s.support.LedgerHeightProvider != nil {
		return func() (uint64, error) {
			return s.support.LedgerHeightProvider.LedgerHeight(), nil
		}
	}
	return handle
}

func (s *gossipStateProviderExtension) RequestBlocksInRange(handle func(start uint64, end uint64), addPayload func(payload *proto.Payload, blockingMode bool) error) func(start uint64, end uint64) {
	if roles.IsCommitter() {
		return handle
	}

	return func(start uint64, end uint64) {
		logger.Debugf("[%s] Loading blocks in range [%d:%d]", s.chainID, start, end)

		payloads, err := s.loadBlocksInRange(start, end)
		if err != nil {
			logger.Errorf("Error loading blocks for channel [%s]: %s", s.chainID, err)
			return
		}
		for _, payload := range payloads {
			logger.Debugf("[%s] Adding payload for block [%d]", s.chainID, payload.SeqNum)

			if err := addPayload(payload, s.blockingMode); err != nil {
				logger.Errorf("Error adding payloads for channel [%s]: %s", s.chainID, err)
				continue
			}
		}
	}
}

func (s *gossipStateProviderExtension) loadBlocksInRange(fromBlock, toBlock uint64) ([]*proto.Payload, error) {

	var payloads []*proto.Payload

	for num := fromBlock; num <= toBlock; num++ {
		// Don't need to load the private data since we don't actually do anything with it on the endorser.
		block, err := s.support.Ledger.GetBlockByNumber(num)
		if err != nil {
			return nil, errors.WithMessagef(err, "Error reading block and private data for block %d", num)
		}

		blockBytes, err := pb.Marshal(block)
		if err != nil {
			return nil, errors.WithMessagef(err, "Error marshalling block %d", num)
		}

		payloads = append(payloads,
			&proto.Payload{
				SeqNum: num,
				Data:   blockBytes,
			},
		)
	}

	return payloads, nil
}

//isBlockValidated checks if given block is validated
func isBlockValidated(block *common.Block) bool {

	blockData := block.GetData()
	envelopes := blockData.GetData()
	envelopesLen := len(envelopes)

	blockMetadata := block.GetMetadata()
	if blockMetadata == nil || blockMetadata.GetMetadata() == nil {
		return false
	}

	txValidationFlags := ledgerUtil.TxValidationFlags(blockMetadata.GetMetadata()[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
	flagsLen := len(txValidationFlags)

	if envelopesLen != flagsLen {
		return false
	}

	for _, flag := range txValidationFlags {
		if peer.TxValidationCode(flag) == peer.TxValidationCode_NOT_VALIDATED {
			return false
		}
	}

	return true
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
