/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockvisitor

import (
	"strings"
	"sync/atomic"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	ledgerutil "github.com/hyperledger/fabric/core/ledger/util"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/pkg/errors"
)

const (
	// LsccID is the ID of the Lifecycle system chaincode
	LsccID = "lscc"
	// CollectionSeparator is used as a separator between the namespace (chaincode ID) and collection name
	CollectionSeparator = "~"
)

var logger = flogging.MustGetLogger("ext_blockvisitor")

// Write contains all information related to a KV write on the block
type Write struct {
	BlockNum  uint64
	TxID      string
	TxNum     uint64
	Namespace string
	Write     *kvrwset.KVWrite
}

// Read contains all information related to a KV read on the block
type Read struct {
	BlockNum  uint64
	TxID      string
	TxNum     uint64
	Namespace string
	Read      *kvrwset.KVRead
}

// LSCCWrite contains all information related to an LSCC write on the block
type LSCCWrite struct {
	BlockNum uint64
	TxID     string
	TxNum    uint64
	CCID     string
	CCData   *ccprovider.ChaincodeData
	CCP      *cb.CollectionConfigPackage
}

// CCEvent contains all information related to a chaincode Event on the block
type CCEvent struct {
	BlockNum uint64
	TxID     string
	TxNum    uint64
	Event    *pb.ChaincodeEvent
}

// ConfigUpdate contains all information related to a config-update on the block
type ConfigUpdate struct {
	BlockNum     uint64
	ConfigUpdate *cb.ConfigUpdate
}

// CCEventHandler publishes chaincode events
type CCEventHandler func(ccEvent *CCEvent) error

// ReadHandler publishes KV read events
type ReadHandler func(read *Read) error

// WriteHandler publishes KV write events
type WriteHandler func(write *Write) error

// LSCCWriteHandler publishes LSCC write events
type LSCCWriteHandler func(lsccWrite *LSCCWrite) error

// ConfigUpdateHandler publishes config updates
type ConfigUpdateHandler func(update *ConfigUpdate) error

// Handlers contains the full set of handlers
type Handlers struct {
	HandleCCEvent      CCEventHandler
	HandleRead         ReadHandler
	HandleWrite        WriteHandler
	HandleLSCCWrite    LSCCWriteHandler
	HandleConfigUpdate ConfigUpdateHandler
}

// Options contains all block Visitor options
type Options struct {
	*Handlers
	StopOnError bool
}

// Opt is a block visitor option
type Opt func(options *Options)

// WithNoStopOnError indicates that processing should proceed even
// though an error occurs in one of the handlers.
func WithNoStopOnError() Opt {
	return func(options *Options) {
		options.StopOnError = false
	}
}

// WithCCEventHandler sets the chaincode event handler
func WithCCEventHandler(handler CCEventHandler) Opt {
	return func(options *Options) {
		options.HandleCCEvent = handler
	}
}

// WithReadHandler sets the read handler
func WithReadHandler(handler ReadHandler) Opt {
	return func(options *Options) {
		options.HandleRead = handler
	}
}

// WithWriteHandler sets the write handler
func WithWriteHandler(handler WriteHandler) Opt {
	return func(options *Options) {
		options.HandleWrite = handler
	}
}

// WithLSCCWriteHandler sets the LSCC write handler
func WithLSCCWriteHandler(handler LSCCWriteHandler) Opt {
	return func(options *Options) {
		options.HandleLSCCWrite = handler
	}
}

// WithConfigUpdateHandler sets the config-update handler
func WithConfigUpdateHandler(handler ConfigUpdateHandler) Opt {
	return func(options *Options) {
		options.HandleConfigUpdate = handler
	}
}

// Visitor traverses a block and publishes KV Read, KV Write, LSCC write, and chaincode events to registered handlers
type Visitor struct {
	*Options
	channelID          string
	lastCommittedBlock uint64
}

// New returns a new block Visitor for the given channel
func New(channelID string, opts ...Opt) *Visitor {
	visitor := &Visitor{
		channelID: channelID,
		Options: &Options{
			Handlers:    noopHandlers(),
			StopOnError: true,
		},
	}

	for _, opt := range opts {
		opt(visitor.Options)
	}

	return visitor
}

// ChannelID returns the channel ID for the vistor
func (p *Visitor) ChannelID() string {
	return p.channelID
}

// Visit traverses the given block and invokes the applicable handlers
func (p *Visitor) Visit(block *cb.Block) error {
	defer atomic.StoreUint64(&p.lastCommittedBlock, block.Header.Number)
	return newBlockEvent(p.channelID, block, p.Options).visit()
}

// LedgerHeight returns ledger height based on last block published
func (p *Visitor) LedgerHeight() uint64 {
	return atomic.LoadUint64(&p.lastCommittedBlock) + 1
}

type blockEvent struct {
	*Options
	channelID string
	block     *cb.Block
}

func newBlockEvent(channelID string, block *cb.Block, options *Options) *blockEvent {
	return &blockEvent{
		Options:   options,
		channelID: channelID,
		block:     block,
	}
}

func (p *blockEvent) visit() error {
	logger.Debugf("[%s] Publishing block #%d", p.channelID, p.block.Header.Number)
	for txNum := range p.block.Data.Data {
		envelope, err := extractEnvelope(p.block, txNum)
		if err != nil {
			if p.StopOnError {
				return errors.Wrapf(err, "error extracting envelope at index %d in block %d", txNum, p.block.Header.Number)
			}
			logger.Warningf("[%s] Error extracting envelope at index %d in block %d: %s", p.channelID, txNum, p.block.Header.Number, err)
			continue
		}
		if err = p.visitEnvelope(uint64(txNum), envelope); err != nil {
			if p.StopOnError {
				return errors.Wrapf(err, "error visiting envelope at index %d in block %d", txNum, p.block.Header.Number)
			}
			logger.Warningf("[%s] Error visiting envelope at index %d in block %d: %s", p.channelID, txNum, p.block.Header.Number, err)
		}
	}
	return nil
}

func (p *blockEvent) visitEnvelope(txNum uint64, envelope *cb.Envelope) error {
	payload, err := extractPayload(envelope)
	if err != nil {
		return err
	}

	chdr, err := unmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		return err
	}

	if cb.HeaderType(chdr.Type) == cb.HeaderType_ENDORSER_TRANSACTION {
		txFilter := ledgerutil.TxValidationFlags(p.block.Metadata.Metadata[cb.BlockMetadataIndex_TRANSACTIONS_FILTER])
		code := txFilter.Flag(int(txNum))
		if code != pb.TxValidationCode_VALID {
			logger.Debugf("[%s] Transaction at index %d in block %d is not valid. Status code: %s", p.channelID, txNum, p.block.Header.Number, code)
			return nil
		}
		tx, err := getTransaction(payload.Data)
		if err != nil {
			return err
		}
		return newTxEvent(p.channelID, p.block.Header.Number, txNum, chdr.TxId, tx, p.Options).visit()
	}

	if cb.HeaderType(chdr.Type) == cb.HeaderType_CONFIG_UPDATE {
		envelope := &cb.ConfigUpdateEnvelope{}
		if err := unmarshal(payload.Data, envelope); err != nil {
			return err
		}
		return newConfigUpdateEvent(p.channelID, p.block.Header.Number, envelope, p.Handlers).visit()
	}

	return nil
}

type txEvent struct {
	*Options
	channelID string
	blockNum  uint64
	txNum     uint64
	txID      string
	tx        *pb.Transaction
}

func newTxEvent(channelID string, blockNum uint64, txNum uint64, txID string, tx *pb.Transaction, options *Options) *txEvent {
	return &txEvent{
		Options:   options,
		channelID: channelID,
		blockNum:  blockNum,
		txNum:     txNum,
		txID:      txID,
		tx:        tx,
	}
}

func (p *txEvent) visit() error {
	logger.Debugf("[%s] Publishing Tx %s in block #%d", p.channelID, p.txID, p.blockNum)
	for i, action := range p.tx.Actions {
		err := p.visitTXAction(action)
		if err != nil {
			if p.StopOnError {
				return errors.Wrapf(err, "error checking TxAction at index %d", i)
			}
			logger.Warningf("[%s] Error checking TxAction at index %d: %s", p.channelID, i, err)
		}
	}
	return nil
}

func (p *txEvent) visitTXAction(action *pb.TransactionAction) error {
	chaPayload, err := getChaincodeActionPayload(action.Payload)
	if err != nil {
		return err
	}
	return p.visitChaincodeActionPayload(chaPayload)
}

func (p *txEvent) visitChaincodeActionPayload(chaPayload *pb.ChaincodeActionPayload) error {
	cpp := &pb.ChaincodeProposalPayload{}
	err := unmarshal(chaPayload.ChaincodeProposalPayload, cpp)
	if err != nil {
		return err
	}

	return p.visitAction(chaPayload.Action)
}

func (p *txEvent) visitAction(action *pb.ChaincodeEndorsedAction) error {
	prp := &pb.ProposalResponsePayload{}
	err := unmarshal(action.ProposalResponsePayload, prp)
	if err != nil {
		return err
	}
	return p.visitProposalResponsePayload(prp)
}

func (p *txEvent) visitProposalResponsePayload(prp *pb.ProposalResponsePayload) error {
	chaincodeAction := &pb.ChaincodeAction{}
	err := unmarshal(prp.Extension, chaincodeAction)
	if err != nil {
		return err
	}
	return p.visitChaincodeAction(chaincodeAction)
}

func (p *txEvent) visitChaincodeAction(chaincodeAction *pb.ChaincodeAction) error {
	if len(chaincodeAction.Results) > 0 {
		txRWSet := &rwsetutil.TxRwSet{}
		if err := txRWSet.FromProtoBytes(chaincodeAction.Results); err != nil {
			return err
		}
		if err := p.visitTxReadWriteSet(txRWSet); err != nil {
			return err
		}
	}

	if len(chaincodeAction.Events) > 0 {
		evt := &pb.ChaincodeEvent{}
		if err := unmarshal(chaincodeAction.Events, evt); err != nil {
			logger.Warningf("[%s] Invalid chaincode Event for chaincode [%s]", p.channelID, chaincodeAction.ChaincodeId)
			return errors.WithMessagef(err, "invalid chaincode Event for chaincode [%s]", chaincodeAction.ChaincodeId)
		}
		err := p.HandleCCEvent(&CCEvent{
			BlockNum: p.blockNum,
			TxID:     p.txID,
			Event:    evt,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *txEvent) visitTxReadWriteSet(txRWSet *rwsetutil.TxRwSet) error {
	for _, nsRWSet := range txRWSet.NsRwSets {
		if err := p.visitNsReadWriteSet(nsRWSet); err != nil {
			if p.StopOnError {
				return err
			}
		}
	}
	return nil
}

func (p *txEvent) visitNsReadWriteSet(nsRWSet *rwsetutil.NsRwSet) error {
	for _, r := range nsRWSet.KvRwSet.Reads {
		if err := p.HandleRead(&Read{
			BlockNum:  p.blockNum,
			TxID:      p.txID,
			Namespace: nsRWSet.NameSpace,
			Read:      r,
		}); err != nil {
			if p.StopOnError {
				return err
			}
		}
	}
	if nsRWSet.NameSpace == LsccID {
		return p.publishLSCCWrite(nsRWSet.KvRwSet.Writes)
	}
	return p.publishWrites(nsRWSet.NameSpace, nsRWSet.KvRwSet.Writes)
}

// publishLSCCWrite publishes an LSCC Write Event which is a result of a chaincode instantiate/upgrade.
// The Event consists of two writes: CC data and collection configs.
func (p *txEvent) publishLSCCWrite(writes []*kvrwset.KVWrite) error {
	ccID, ccData, ccp, err := getCCInfo(writes)
	if err != nil {
		if p.StopOnError {
			return err
		}
		logger.Warningf("[%s] Error getting chaincode info: %s", p.channelID, err)
		return nil
	}

	return p.HandleLSCCWrite(&LSCCWrite{
		BlockNum: p.blockNum,
		TxID:     p.txID,
		CCID:     ccID,
		CCData:   ccData,
		CCP:      ccp,
	})
}

func (p *txEvent) publishWrites(ns string, writes []*kvrwset.KVWrite) error {
	for _, w := range writes {
		if err := p.HandleWrite(&Write{
			BlockNum:  p.blockNum,
			TxID:      p.txID,
			Namespace: ns,
			Write:     w,
		}); err != nil {
			if p.StopOnError {
				return err
			}
		}
	}
	return nil
}

type configUpdateEvent struct {
	*Handlers
	channelID string
	blockNum  uint64
	envelope  *cb.ConfigUpdateEnvelope
}

func newConfigUpdateEvent(channelID string, blockNum uint64, envelope *cb.ConfigUpdateEnvelope, handlers *Handlers) *configUpdateEvent {
	return &configUpdateEvent{
		Handlers:  handlers,
		channelID: channelID,
		blockNum:  blockNum,
		envelope:  envelope,
	}
}

func (p *configUpdateEvent) visit() error {
	cu := &cb.ConfigUpdate{}
	if err := unmarshal(p.envelope.ConfigUpdate, cu); err != nil {
		logger.Warningf("[%s] Error unmarshalling config update: %s", p.channelID, err)
		return err
	}

	logger.Debugf("[%s] Publishing Config Update [%s] in block #%d", p.channelID, cu, p.blockNum)

	return p.HandleConfigUpdate(&ConfigUpdate{
		BlockNum:     p.blockNum,
		ConfigUpdate: cu,
	})
}

func getCCInfo(writes []*kvrwset.KVWrite) (string, *ccprovider.ChaincodeData, *cb.CollectionConfigPackage, error) {
	var ccID string
	var ccp *cb.CollectionConfigPackage
	var ccData *ccprovider.ChaincodeData

	for _, kvWrite := range writes {
		if isCollectionConfigKey(kvWrite.Key) {
			ccp = &cb.CollectionConfigPackage{}
			if err := unmarshal(kvWrite.Value, ccp); err != nil {
				return "", nil, nil, errors.WithMessagef(err, "error unmarshaling collection configuration")
			}
		} else {
			ccID = kvWrite.Key
			ccData = &ccprovider.ChaincodeData{}
			if err := unmarshal(kvWrite.Value, ccData); err != nil {
				return "", nil, nil, errors.WithMessagef(err, "error unmarshaling chaincode data")
			}
		}
	}

	return ccID, ccData, ccp, nil
}

func noopHandlers() *Handlers {
	return &Handlers{
		HandleCCEvent:      func(*CCEvent) error { return nil },
		HandleRead:         func(*Read) error { return nil },
		HandleWrite:        func(*Write) error { return nil },
		HandleLSCCWrite:    func(*LSCCWrite) error { return nil },
		HandleConfigUpdate: func(*ConfigUpdate) error { return nil },
	}
}

func isCollectionConfigKey(key string) bool {
	return strings.Contains(key, CollectionSeparator)
}

var unmarshal = func(buf []byte, pb proto.Message) error {
	return proto.Unmarshal(buf, pb)
}

var extractEnvelope = func(block *cb.Block, index int) (*cb.Envelope, error) {
	return protoutil.ExtractEnvelope(block, index)
}

var extractPayload = func(envelope *cb.Envelope) (*cb.Payload, error) {
	return protoutil.ExtractPayload(envelope)
}

var unmarshalChannelHeader = func(bytes []byte) (*cb.ChannelHeader, error) {
	return protoutil.UnmarshalChannelHeader(bytes)
}

var getTransaction = func(txBytes []byte) (*pb.Transaction, error) {
	return protoutil.GetTransaction(txBytes)
}

var getChaincodeActionPayload = func(capBytes []byte) (*pb.ChaincodeActionPayload, error) {
	return protoutil.GetChaincodeActionPayload(capBytes)
}
