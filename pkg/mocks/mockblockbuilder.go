/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"crypto/sha256"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	cutil "github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	ledger_util "github.com/hyperledger/fabric/core/ledger/util"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protoutil"
)

// BlockBuilder builds a mock Block
type BlockBuilder struct {
	channelID    string
	blockNum     uint64
	previousHash []byte
	configUpdate *ConfigUpdateBuilder
	transactions []*TxBuilder
}

// NewBlockBuilder returns a new mock BlockBuilder
func NewBlockBuilder(channelID string, blockNum uint64) *BlockBuilder {
	return &BlockBuilder{
		channelID: channelID,
		blockNum:  blockNum,
	}
}

// Build builds the block
func (b *BlockBuilder) Build() *cb.Block {
	block := &cb.Block{}
	block.Header = &cb.BlockHeader{}
	block.Header.Number = b.blockNum
	block.Header.PreviousHash = b.previousHash
	block.Data = &cb.BlockData{}

	var metadataContents [][]byte
	for i := 0; i < len(cb.BlockMetadataIndex_name); i++ {
		metadataContents = append(metadataContents, []byte{})
	}
	block.Metadata = &cb.BlockMetadata{Metadata: metadataContents}

	if b.configUpdate != nil {
		block.Data.Data = append(block.Data.Data, b.configUpdate.Build())
	} else {
		var txValidationCodes []uint8
		for _, tx := range b.transactions {
			txBytes, txValidationCode := tx.Build()
			block.Data.Data = append(block.Data.Data, txBytes)
			txValidationCodes = append(txValidationCodes, uint8(txValidationCode))
		}
		txsfltr := ledger_util.NewTxValidationFlags(len(block.Data.Data))
		for i := 0; i < len(block.Data.Data); i++ {
			txsfltr[i] = txValidationCodes[i]
		}
		block.Metadata.Metadata[cb.BlockMetadataIndex_TRANSACTIONS_FILTER] = txsfltr
	}

	blockbytes := cutil.ConcatenateBytes(block.Data.Data...)
	block.Header.DataHash = computeSHA256(blockbytes)

	return block
}

// ConfigUpdate adds a config update
func (b *BlockBuilder) ConfigUpdate() *ConfigUpdateBuilder {
	if len(b.transactions) > 0 {
		panic("Cannot mix config updates with endorsement transactions")
	}
	cb := NewConfigUpdateBuilder(b.channelID)
	b.configUpdate = cb
	return cb
}

// ConfigUpdateBuilder builds a mock config update envelope
type ConfigUpdateBuilder struct {
	channelID string
}

// NewConfigUpdateBuilder returns a new mock ConfigUpdateBuilder
func NewConfigUpdateBuilder(channelID string) *ConfigUpdateBuilder {
	return &ConfigUpdateBuilder{
		channelID: channelID,
	}
}

// Build builds a config update envelope
func (b *ConfigUpdateBuilder) Build() []byte {
	chdr := &cb.ChannelHeader{
		Type:    int32(cb.HeaderType_CONFIG_UPDATE),
		Version: 1,
		Timestamp: &timestamp.Timestamp{
			Seconds: time.Now().Unix(),
			Nanos:   0,
		},
		ChannelId: b.channelID}
	hdr := &cb.Header{ChannelHeader: protoutil.MarshalOrPanic(chdr)}
	payload := &cb.Payload{Header: hdr}

	env := &cb.Envelope{}

	var err error
	env.Payload, err = protoutil.GetBytesPayload(payload)
	if err != nil {
		panic(err.Error())
	}
	ebytes, err := protoutil.GetBytesEnvelope(env)
	if err != nil {
		panic(err.Error())
	}

	return ebytes
}

// Transaction adds a new transaction
func (b *BlockBuilder) Transaction(txID string, validationCode pb.TxValidationCode) *TxBuilder {
	if b.configUpdate != nil {
		panic("Cannot mix config updates with endorsement transactions")
	}
	tx := NewTxBuilder(b.channelID, txID, validationCode)
	b.transactions = append(b.transactions, tx)
	return tx
}

// TxBuilder builds a mock Transaction
type TxBuilder struct {
	channelID        string
	txID             string
	validationCode   pb.TxValidationCode
	chaincodeActions []*ChaincodeActionBuilder
}

// NewTxBuilder returns a new mock TxBuilder
func NewTxBuilder(channelID, txID string, validationCode pb.TxValidationCode) *TxBuilder {
	return &TxBuilder{
		channelID:      channelID,
		txID:           txID,
		validationCode: validationCode,
	}
}

// Build builds a transaction
func (b *TxBuilder) Build() ([]byte, pb.TxValidationCode) {
	chdr := &cb.ChannelHeader{
		Type:    int32(cb.HeaderType_ENDORSER_TRANSACTION),
		Version: 1,
		Timestamp: &timestamp.Timestamp{
			Seconds: time.Now().Unix(),
			Nanos:   0,
		},
		ChannelId: b.channelID,
		TxId:      b.txID}
	hdr := &cb.Header{ChannelHeader: protoutil.MarshalOrPanic(chdr)}
	payload := &cb.Payload{Header: hdr}

	env := &cb.Envelope{}

	tx := &pb.Transaction{}

	for _, ccAction := range b.chaincodeActions {
		tx.Actions = append(tx.Actions, &pb.TransactionAction{
			Payload: ccAction.Build(),
		})
	}

	var err error
	payload.Data, err = protoutil.GetBytesTransaction(tx)
	if err != nil {
		panic(err.Error())
	}
	env.Payload, err = protoutil.GetBytesPayload(payload)
	if err != nil {
		panic(err.Error())
	}
	ebytes, err := protoutil.GetBytesEnvelope(env)
	if err != nil {
		panic(err.Error())
	}

	return ebytes, b.validationCode
}

// ChaincodeAction adds a chaincode action to the transaction
func (b *TxBuilder) ChaincodeAction(ccID string) *ChaincodeActionBuilder {
	cc := NewChaincodeActionBuilder(ccID, b.txID)
	b.chaincodeActions = append(b.chaincodeActions, cc)
	return cc
}

// ChaincodeActionBuilder builds a mock Chaincode Action
type ChaincodeActionBuilder struct {
	ccID         string
	txID         string
	response     *pb.Response
	ccEvent      *pb.ChaincodeEvent
	nsRWSet      *NamespaceRWSetBuilder
	collNSRWSets []*NamespaceRWSetBuilder
}

// NewChaincodeActionBuilder returns a new ChaincodeActionBuilder
func NewChaincodeActionBuilder(ccID, txID string) *ChaincodeActionBuilder {
	return &ChaincodeActionBuilder{
		ccID:    ccID,
		txID:    txID,
		nsRWSet: NewNamespaceRWSetBuilder(ccID),
	}
}

// Build builds the chaincode action
func (b *ChaincodeActionBuilder) Build() []byte {
	ccID := &pb.ChaincodeID{
		Name: b.ccID,
	}

	var ccEventBytes []byte
	if b.ccEvent != nil {
		var err error
		ccEventBytes, err = proto.Marshal(b.ccEvent)
		if err != nil {
			panic(err.Error())
		}
	}

	txRWSet := &rwsetutil.TxRwSet{}
	txRWSet.NsRwSets = append(txRWSet.NsRwSets, b.nsRWSet.Build())
	for _, collRWSet := range b.collNSRWSets {
		txRWSet.NsRwSets = append(txRWSet.NsRwSets, collRWSet.Build())
	}

	nsRWSetBytes, err := txRWSet.ToProtoBytes()
	if err != nil {
		panic(err.Error())
	}

	proposalResponsePayload, err := protoutil.GetBytesProposalResponsePayload(
		[]byte("proposal_hash"), b.response, nsRWSetBytes, ccEventBytes, ccID)
	if err != nil {
		panic(err.Error())
	}

	ccaPayload := &pb.ChaincodeActionPayload{
		Action: &pb.ChaincodeEndorsedAction{
			ProposalResponsePayload: proposalResponsePayload,
		},
	}

	payload, err := protoutil.GetBytesChaincodeActionPayload(ccaPayload)
	if err != nil {
		panic(err.Error())
	}
	return payload
}

// Response sets the chaincode response
func (b *ChaincodeActionBuilder) Response(response *pb.Response) *ChaincodeActionBuilder {
	b.response = response
	return b
}

// Read adds a KV read to the read/write set
func (b *ChaincodeActionBuilder) Read(key string, version *kvrwset.Version) *ChaincodeActionBuilder {
	b.nsRWSet.Read(key, version)
	return b
}

// Write adds a KV write to the read/write set
func (b *ChaincodeActionBuilder) Write(key string, value []byte) *ChaincodeActionBuilder {
	b.nsRWSet.Write(key, value)
	return b
}

// Delete adds a KV write (with delete=true) to the read/write set
func (b *ChaincodeActionBuilder) Delete(key string) *ChaincodeActionBuilder {
	b.nsRWSet.Delete(key)
	return b
}

// ChaincodeEvent adds a chaincode event to the chaincode action
func (b *ChaincodeActionBuilder) ChaincodeEvent(eventName string, payload []byte) *ChaincodeActionBuilder {
	b.ccEvent = &pb.ChaincodeEvent{
		ChaincodeId: b.ccID,
		TxId:        b.txID,
		EventName:   eventName,
		Payload:     payload,
	}
	return b
}

// Collection starts a new collection read/write set
func (b *ChaincodeActionBuilder) Collection(coll string) *NamespaceRWSetBuilder {
	nsRWSetBuilder := NewNamespaceRWSetBuilder(b.ccID + "~" + coll)
	b.collNSRWSets = append(b.collNSRWSets, nsRWSetBuilder)
	return nsRWSetBuilder
}

// NamespaceRWSetBuilder builds a mock read/write set for a given namespace
type NamespaceRWSetBuilder struct {
	namespace string
	reads     []*kvrwset.KVRead
	writes    []*kvrwset.KVWrite
}

// NewNamespaceRWSetBuilder returns a new namespace read/write set builder
func NewNamespaceRWSetBuilder(ns string) *NamespaceRWSetBuilder {
	return &NamespaceRWSetBuilder{
		namespace: ns,
	}
}

// Build builds a namespace read/write set
func (b *NamespaceRWSetBuilder) Build() *rwsetutil.NsRwSet {
	return &rwsetutil.NsRwSet{
		NameSpace: b.namespace,
		KvRwSet: &kvrwset.KVRWSet{
			Reads:  b.reads,
			Writes: b.writes,
		},
	}
}

// Read adds a KV read to the read/write set
func (b *NamespaceRWSetBuilder) Read(key string, version *kvrwset.Version) *NamespaceRWSetBuilder {
	b.reads = append(b.reads, &kvrwset.KVRead{
		Key:     key,
		Version: version,
	})
	return b
}

// Write adds a KV write to the read/write set
func (b *NamespaceRWSetBuilder) Write(key string, value []byte) *NamespaceRWSetBuilder {
	b.writes = append(b.writes, &kvrwset.KVWrite{
		Key:   key,
		Value: value,
	})
	return b
}

// Delete adds a KV write (with delete=true) to the read/write set
func (b *NamespaceRWSetBuilder) Delete(key string) *NamespaceRWSetBuilder {
	b.writes = append(b.writes, &kvrwset.KVWrite{
		Key:      key,
		IsDelete: true,
	})
	return b
}

func computeSHA256(data []byte) (hash []byte) {
	h := sha256.New()
	_, err := h.Write(data)
	if err != nil {
		panic("unable to create digest")
	}
	return h.Sum(nil)
}
