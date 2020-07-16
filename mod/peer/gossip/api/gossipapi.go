/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	cb "github.com/hyperledger/fabric-protos-go/common"
	proto "github.com/hyperledger/fabric-protos-go/gossip"
	"github.com/hyperledger/fabric-protos-go/ledger/rwset/kvrwset"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric-protos-go/transientstore"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/comm"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/gossip"
	"github.com/hyperledger/fabric/gossip/protoext"
)

// ConfigUpdateHandler handles a config update
type ConfigUpdateHandler func(blockNum uint64, configUpdate *cb.ConfigUpdate) error

// WriteHandler handles a KV write
type WriteHandler func(txMetadata TxMetadata, namespace string, kvWrite *kvrwset.KVWrite) error

// ReadHandler handles a KV read
type ReadHandler func(txMetadata TxMetadata, namespace string, kvRead *kvrwset.KVRead) error

// CollHashWriteHandler handles a KV collection hash write
type CollHashWriteHandler func(txMetadata TxMetadata, namespace, collection string, kvWrite *kvrwset.KVWriteHash) error

// CollHashReadHandler handles a KV collection hash read
type CollHashReadHandler func(txMetadata TxMetadata, namespace, collection string, kvRead *kvrwset.KVReadHash) error

// ChaincodeEventHandler handles a chaincode event
type ChaincodeEventHandler func(txMetadata TxMetadata, event *pb.ChaincodeEvent) error

// ChaincodeUpgradeHandler handles chaincode upgrade events
type ChaincodeUpgradeHandler func(txMetadata TxMetadata, chaincodeName string) error

// LSCCWriteHandler handles chaincode instantiation/upgrade events
type LSCCWriteHandler func(txMetadata TxMetadata, chaincodeName string, ccData *ccprovider.ChaincodeData, ccp *pb.CollectionConfigPackage) error

// BlockHandler allows clients to add handlers for various block events
type BlockHandler interface {
	// AddCCUpgradeHandler adds a handler for chaincode upgrades
	AddCCUpgradeHandler(handler ChaincodeUpgradeHandler)
	// AddConfigUpdateHandler adds a handler for config updates
	AddConfigUpdateHandler(handler ConfigUpdateHandler)
	// AddWriteHandler adds a handler for KV writes
	AddWriteHandler(handler WriteHandler)
	// AddReadHandler adds a handler for KV reads
	AddReadHandler(handler ReadHandler)
	// AddCollHashReadHandler adds a new handler for KV collection hash reads
	AddCollHashReadHandler(handler CollHashReadHandler)
	// AddCollHashWriteHandler adds a new handler for KV collection hash writes
	AddCollHashWriteHandler(handler CollHashWriteHandler)
	// AddLSCCWriteHandler adds a handler for LSCC writes (for chaincode instantiate/upgrade)
	AddLSCCWriteHandler(handler LSCCWriteHandler)
	// AddCCEventHandler adds a handler for chaincode events
	AddCCEventHandler(handler ChaincodeEventHandler)
	//LedgerHeight returns current in memory ledger height
	LedgerHeight() uint64
}

// BlockVisitor allows clients to add handlers for various block events
type BlockVisitor interface {
	BlockHandler
	// Visit traverses the block and invokes all applicable handlers
	Visit(block *cb.Block) error
}

// BlockPublisher allows clients to add handlers for various block events
type BlockPublisher interface {
	BlockHandler
	// Publish traverses the block and invokes all applicable handlers
	Publish(block *cb.Block, pvtData ledger.TxPvtDataMap)
}

// TxMetadata contain txn metadata
type TxMetadata struct {
	BlockNum  uint64
	TxNum     uint64
	ChannelID string
	TxID      string
}

//LedgerHeightProvider provides current ledger height
type LedgerHeightProvider interface {
	//LedgerHeight  returns current in-memory ledger height
	LedgerHeight() uint64
}

// Support aggregates functionality of several
// interfaces required by gossip service
type Support struct {
	Ledger               ledger.PeerLedger
	LedgerHeightProvider LedgerHeightProvider
}

// GossipService encapsulates gossip and state capabilities into single interface
type GossipService interface {
	// SelfMembershipInfo returns the peer's membership information
	SelfMembershipInfo() discovery.NetworkMember

	// SelfChannelInfo returns the peer's latest StateInfo message of a given channel
	SelfChannelInfo(id common.ChannelID) *protoext.SignedGossipMessage

	// Send sends a message to remote peers
	Send(msg *proto.GossipMessage, peers ...*comm.RemotePeer)

	// SendByCriteria sends a given message to all peers that match the given SendCriteria
	SendByCriteria(*protoext.SignedGossipMessage, gossip.SendCriteria) error

	// GetPeers returns the NetworkMembers considered alive
	Peers() []discovery.NetworkMember

	// PeersOfChannel returns the NetworkMembers considered alive
	// and also subscribed to the channel given
	PeersOfChannel(common.ChannelID) []discovery.NetworkMember

	// Gossip sends a message to other peers to the network
	Gossip(msg *proto.GossipMessage)

	// Accept returns a dedicated read-only channel for messages sent by other nodes that match a certain predicate.
	// If passThrough is false, the messages are processed by the gossip layer beforehand.
	// If passThrough is true, the gossip layer doesn't intervene and the messages
	// can be used to send a reply back to the sender
	Accept(acceptor common.MessageAcceptor, passThrough bool) (<-chan *proto.GossipMessage, <-chan protoext.ReceivedMessage)

	// IdentityInfo returns information known peer identities
	IdentityInfo() api.PeerIdentitySet

	// IsInMyOrg checks whether a network member is in this peer's org
	IsInMyOrg(member discovery.NetworkMember) bool

	// DistributePrivateData distributes private data to the peers in the collections
	// according to policies induced by the PolicyStore and PolicyParser
	DistributePrivateData(chainID string, txID string, privateData *transientstore.TxPvtReadWriteSetWithConfigInfo, blkHt uint64) error
}
