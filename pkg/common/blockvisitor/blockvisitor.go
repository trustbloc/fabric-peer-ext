/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockvisitor

import (
	"strings"
	"sync/atomic"

	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/ledger/rwset/kvrwset"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	ledgerutil "github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/util"
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

// CollHashWrite contains all information related to a KV collection hash write on the block
type CollHashWrite struct {
	BlockNum   uint64
	TxID       string
	TxNum      uint64
	Namespace  string
	Collection string
	Write      *kvrwset.KVWriteHash
}

// CollHashRead contains all information related to a KV collection hash read on the block
type CollHashRead struct {
	BlockNum   uint64
	TxID       string
	TxNum      uint64
	Namespace  string
	Collection string
	Read       *kvrwset.KVReadHash
}

// LSCCWrite contains all information related to an LSCC write on the block
type LSCCWrite struct {
	BlockNum uint64
	TxID     string
	TxNum    uint64
	CCID     string
	CCData   *ccprovider.ChaincodeData
	CCP      *pb.CollectionConfigPackage
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

// CollHashReadHandler publishes KV collection hash read events
type CollHashReadHandler func(read *CollHashRead) error

// CollHashWriteHandler publishes KV collection hash write events
type CollHashWriteHandler func(write *CollHashWrite) error

// LSCCWriteHandler publishes LSCC write events
type LSCCWriteHandler func(lsccWrite *LSCCWrite) error

// ConfigUpdateHandler publishes config updates
type ConfigUpdateHandler func(update *ConfigUpdate) error

// Handlers contains the full set of handlers
type Handlers struct {
	HandleCCEvent       CCEventHandler
	HandleRead          ReadHandler
	HandleWrite         WriteHandler
	HandleCollHashRead  CollHashReadHandler
	HandleCollHashWrite CollHashWriteHandler
	HandleLSCCWrite     LSCCWriteHandler
	HandleConfigUpdate  ConfigUpdateHandler
}

// ErrorHandler allows clients to handle errors
type ErrorHandler func(err error, ctx *Context) error

// Options contains all block Visitor options
type Options struct {
	*Handlers
	HandleError ErrorHandler
}

// Opt is a block visitor option
type Opt func(options *Options)

// WithErrorHandler provides an error handler that's invoked when
// an error occurs in one of the handlers.
func WithErrorHandler(handler ErrorHandler) Opt {
	return func(options *Options) {
		options.HandleError = handler
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

// WithCollHashReadHandler sets the collection hash read handler
func WithCollHashReadHandler(handler CollHashReadHandler) Opt {
	return func(options *Options) {
		options.HandleCollHashRead = handler
	}
}

// WithCollHashWriteHandler sets the collection hash write handler
func WithCollHashWriteHandler(handler CollHashWriteHandler) Opt {
	return func(options *Options) {
		options.HandleCollHashWrite = handler
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
			HandleError: defaultHandleError,
		},
	}

	for _, opt := range opts {
		opt(visitor.Options)
	}

	return visitor
}

// defaultHandleError logs the error and continues
func defaultHandleError(err error, ctx *Context) error {
	logger.Warnf("Error of type %s occurred: %s", ctx.Category, err)
	return nil
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

// SetLastCommittedBlockNum initializes the visitor with the given block number - only if the given block number
// is greater than the current last committed block number.
func (p *Visitor) SetLastCommittedBlockNum(blockNum uint64) {
	for n := atomic.LoadUint64(&p.lastCommittedBlock); n < blockNum; {
		if atomic.CompareAndSwapUint64(&p.lastCommittedBlock, n, blockNum) {
			logger.Debugf("Changed lastCommittedBlockNumber from %d to %d", n, blockNum)
			break
		}
	}
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
			if e := p.HandleError(err, p.newUnmarshalErrContext()); e != nil {
				return newVisitorError(e)
			}

			logger.Warningf("[%s] Error extracting envelope at index %d in block %d: %s", p.channelID, txNum, p.block.Header.Number, err)
			continue
		}

		if err = p.visitEnvelope(uint64(txNum), envelope); err != nil {
			if haltOnError(err) {
				return errors.Wrapf(err, "error visiting envelope at index %d in block %d", txNum, p.block.Header.Number)
			}

			logger.Warningf("[%s] Error visiting envelope at index %d in block %d: %s", p.channelID, txNum, p.block.Header.Number, err)
			continue
		}
	}
	return nil
}

func (p *blockEvent) visitEnvelope(txNum uint64, envelope *cb.Envelope) error {
	payload, chdr, err := extractPayloadAndHeader(envelope)
	if err != nil {
		if e := p.HandleError(err, p.newUnmarshalErrContext(withTxNum(txNum))); e != nil {
			return newVisitorError(e)
		}

		return nil
	}

	switch cb.HeaderType(chdr.Type) {
	case cb.HeaderType_ENDORSER_TRANSACTION:
		return p.visitTx(txNum, payload, chdr)
	case cb.HeaderType_CONFIG_UPDATE:
		return p.visitConfigUpdate(payload)
	default:
		return nil
	}
}

func (p *blockEvent) visitTx(txNum uint64, payload *cb.Payload, chdr *cb.ChannelHeader) error {
	txFilter := ledgerutil.TxValidationFlags(p.block.Metadata.Metadata[cb.BlockMetadataIndex_TRANSACTIONS_FILTER])
	code := txFilter.Flag(int(txNum))
	if code != pb.TxValidationCode_VALID {
		logger.Debugf("[%s] Transaction at index %d in block %d is not valid. Status code: %s", p.channelID, txNum, p.block.Header.Number, code)
		return nil
	}

	tx, err := getTransaction(payload.Data)
	if err != nil {
		if e := p.HandleError(err, p.newUnmarshalErrContext(withTxNum(txNum), withTxID(chdr.TxId))); e != nil {
			return newVisitorError(e)
		}

		return nil
	}

	return newTxEvent(p.channelID, p.block.Header.Number, txNum, chdr.TxId, tx, p.Options).visit()
}

func (p *blockEvent) visitConfigUpdate(payload *cb.Payload) error {
	envelope := &cb.ConfigUpdateEnvelope{}
	if err := unmarshal(payload.Data, envelope); err != nil {
		return err
	}

	return newConfigUpdateEvent(p.channelID, p.block.Header.Number, envelope, p.Options).visit()
}

func (p *blockEvent) newUnmarshalErrContext(opts ...ctxOpt) *Context {
	return newContext(UnmarshalErr, p.channelID, p.block.Header.Number, opts...)
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
			if haltOnError(err) {
				return err
			}

			logger.Warningf("[%s] Error checking TxAction at index %d: %s", p.channelID, i, err)
		}
	}

	return nil
}

func (p *txEvent) visitTXAction(action *pb.TransactionAction) error {
	chaPayload, err := getChaincodeActionPayload(action.Payload)
	if err != nil {
		if e := p.HandleError(err, p.newContext(UnmarshalErr)); e != nil {
			return newVisitorError(e)
		}

		return nil
	}

	return p.visitChaincodeActionPayload(chaPayload)
}

func (p *txEvent) visitChaincodeActionPayload(chaPayload *pb.ChaincodeActionPayload) error {
	cpp := &pb.ChaincodeProposalPayload{}
	err := unmarshal(chaPayload.ChaincodeProposalPayload, cpp)
	if err != nil {
		if e := p.HandleError(err, p.newContext(UnmarshalErr)); e != nil {
			return newVisitorError(e)
		}

		return nil
	}

	return p.visitAction(chaPayload.Action)
}

func (p *txEvent) visitAction(action *pb.ChaincodeEndorsedAction) error {
	prp := &pb.ProposalResponsePayload{}
	err := unmarshal(action.ProposalResponsePayload, prp)
	if err != nil {
		if e := p.HandleError(err, p.newContext(UnmarshalErr)); e != nil {
			return newVisitorError(e)
		}

		return nil
	}

	return p.visitProposalResponsePayload(prp)
}

func (p *txEvent) visitProposalResponsePayload(prp *pb.ProposalResponsePayload) error {
	chaincodeAction := &pb.ChaincodeAction{}
	err := unmarshal(prp.Extension, chaincodeAction)
	if err != nil {
		if e := p.HandleError(err, p.newContext(UnmarshalErr)); e != nil {
			return newVisitorError(e)
		}

		return nil
	}

	return p.visitChaincodeAction(chaincodeAction)
}

func (p *txEvent) visitChaincodeAction(chaincodeAction *pb.ChaincodeAction) error {
	if err := p.visitChaincodeActionResults(chaincodeAction); err != nil && haltOnError(err) {
		return err
	}

	if err := p.visitChaincodeActionEvents(chaincodeAction); err != nil && haltOnError(err) {
		return err
	}

	return nil
}

func (p *txEvent) visitChaincodeActionResults(chaincodeAction *pb.ChaincodeAction) error {
	if len(chaincodeAction.Results) == 0 {
		return nil
	}

	txRWSet := &rwsetutil.TxRwSet{}
	if err := txRWSet.FromProtoBytes(chaincodeAction.Results); err != nil {
		if e := p.HandleError(err, p.newContext(UnmarshalErr)); e != nil {
			return newVisitorError(e)
		}

		return nil
	}

	if err := p.visitTxReadWriteSet(txRWSet); err != nil && haltOnError(err) {
		return err
	}

	return nil
}

func (p *txEvent) visitChaincodeActionEvents(chaincodeAction *pb.ChaincodeAction) error {
	if len(chaincodeAction.Events) == 0 {
		return nil
	}

	evt := &pb.ChaincodeEvent{}
	if err := unmarshal(chaincodeAction.Events, evt); err != nil {
		if e := p.HandleError(err, p.newContext(UnmarshalErr)); e != nil {
			return newVisitorError(errors.WithMessagef(e, "invalid chaincode Event for chaincode [%s]", chaincodeAction.ChaincodeId))
		}

		logger.Warningf("[%s] Invalid chaincode Event for chaincode [%s]", p.channelID, chaincodeAction.ChaincodeId)
		return nil
	}

	ccEvent := &CCEvent{
		BlockNum: p.blockNum,
		TxID:     p.txID,
		TxNum:    p.txNum,
		Event:    evt,
	}

	if err := p.HandleCCEvent(ccEvent); err != nil {
		if e := p.HandleError(err, p.newContext(CCEventHandlerErr, withCCEvent(ccEvent))); e != nil {
			return newVisitorError(e)
		}
	}

	return nil
}

func (p *txEvent) visitTxReadWriteSet(txRWSet *rwsetutil.TxRwSet) error {
	for _, nsRWSet := range txRWSet.NsRwSets {
		if err := p.visitNsReadWriteSet(nsRWSet); err != nil {
			if haltOnError(err) {
				return err
			}

			logger.Warningf("[%s] Error checking TxReadWriteSet for namespace [%s]: %s", p.channelID, nsRWSet.NameSpace, err)
		}
	}

	return nil
}

func (p *txEvent) visitNsReadWriteSet(nsRWSet *rwsetutil.NsRwSet) error {
	err := p.publishReads(nsRWSet.NameSpace, nsRWSet.KvRwSet.Reads)
	if err != nil && haltOnError(err) {
		return err
	}

	if nsRWSet.NameSpace == LsccID {
		return p.publishLSCCWrite(nsRWSet.KvRwSet.Writes)
	}

	err = p.publishWrites(nsRWSet.NameSpace, nsRWSet.KvRwSet.Writes)
	if err != nil && haltOnError(err) {
		return err
	}

	err = p.visitCollHashRWSets(nsRWSet.NameSpace, nsRWSet.CollHashedRwSets)
	if err != nil {
		return err
	}

	return nil
}

// publishLSCCWrite publishes an LSCC Write Event which is a result of a chaincode instantiate/upgrade.
// The Event consists of two writes: CC data and collection configs.
func (p *txEvent) publishLSCCWrite(writes []*kvrwset.KVWrite) error {
	if len(writes) == 0 {
		return nil
	}

	ccID, ccData, ccp, err := getCCInfo(writes)
	if err != nil {
		if e := p.HandleError(err, p.newContext(UnmarshalErr)); e != nil {
			return newVisitorError(e)
		}

		logger.Warnf("Error getting CC info: %s", err)
		return nil
	}

	lsccWrite := &LSCCWrite{
		BlockNum: p.blockNum,
		TxID:     p.txID,
		TxNum:    p.txNum,
		CCID:     ccID,
		CCData:   ccData,
		CCP:      ccp,
	}

	if err := p.HandleLSCCWrite(lsccWrite); err != nil {
		if e := p.HandleError(err, p.newContext(LSCCWriteHandlerErr, withLSCCWrite(lsccWrite))); e != nil {
			return newVisitorError(e)
		}
	}

	return nil
}

func (p *txEvent) visitCollHashRWSets(nameSpace string, collHashedRwSets []*rwsetutil.CollHashedRwSet) error {
	for _, collRwSet := range collHashedRwSets {
		err := p.publishCollHashReads(nameSpace, collRwSet.CollectionName, collRwSet.HashedRwSet.HashedReads)
		if err != nil && haltOnError(err) {
			return err
		}

		err = p.publishCollHashWrites(nameSpace, collRwSet.CollectionName, collRwSet.HashedRwSet.HashedWrites)
		if err != nil && haltOnError(err) {
			return err
		}
	}

	return nil
}

func (p *txEvent) publishCollHashReads(ns, coll string, reads []*kvrwset.KVReadHash) error {
	for _, r := range reads {
		read := &CollHashRead{
			BlockNum:   p.blockNum,
			TxID:       p.txID,
			TxNum:      p.txNum,
			Namespace:  ns,
			Collection: coll,
			Read:       r,
		}

		if err := p.HandleCollHashRead(read); err != nil {
			if e := p.HandleError(err, p.newContext(CollHashReadHandlerErr, withCollHashRead(read))); e != nil {
				return newVisitorError(e)
			}

			logger.Warningf("[%s] Error checking collection hash read %+v", p.channelID, r, err)
		}
	}

	return nil
}

func (p *txEvent) publishCollHashWrites(ns, coll string, reads []*kvrwset.KVWriteHash) error {
	for _, w := range reads {
		write := &CollHashWrite{
			BlockNum:   p.blockNum,
			TxID:       p.txID,
			TxNum:      p.txNum,
			Namespace:  ns,
			Collection: coll,
			Write:      w,
		}

		if err := p.HandleCollHashWrite(write); err != nil {
			if e := p.HandleError(err, p.newContext(CollHashWriteHandlerErr, withCollHashWrite(write))); e != nil {
				return newVisitorError(e)
			}

			logger.Warningf("[%s] Error checking collection hash write %+v", p.channelID, w, err)
		}
	}

	return nil
}

func (p *txEvent) publishWrites(ns string, writes []*kvrwset.KVWrite) error {
	for _, w := range writes {
		write := &Write{
			BlockNum:  p.blockNum,
			TxID:      p.txID,
			TxNum:     p.txNum,
			Namespace: ns,
			Write:     w,
		}

		if err := p.HandleWrite(write); err != nil {
			if e := p.HandleError(err, p.newContext(WriteHandlerErr, withWrite(write))); e != nil {
				return newVisitorError(e)
			}
		}
	}
	return nil
}

func (p *txEvent) publishReads(ns string, reads []*kvrwset.KVRead) error {
	for _, r := range reads {
		read := &Read{
			BlockNum:  p.blockNum,
			TxID:      p.txID,
			TxNum:     p.txNum,
			Namespace: ns,
			Read:      r,
		}

		if err := p.HandleRead(read); err != nil {
			if e := p.HandleError(err, p.newContext(ReadHandlerErr, withRead(read))); e != nil {
				return newVisitorError(e)
			}

			logger.Warningf("[%s] Error checking read %+v", p.channelID, r, err)
		}
	}

	return nil
}

func (p *txEvent) newContext(category Category, opts ...ctxOpt) *Context {
	return newContext(category, p.channelID, p.blockNum, append(opts, withTxNum(p.txNum), withTxID(p.txID))...)
}

type configUpdateEvent struct {
	*Options
	channelID string
	blockNum  uint64
	envelope  *cb.ConfigUpdateEnvelope
}

func newConfigUpdateEvent(channelID string, blockNum uint64, envelope *cb.ConfigUpdateEnvelope, options *Options) *configUpdateEvent {
	return &configUpdateEvent{
		Options:   options,
		channelID: channelID,
		blockNum:  blockNum,
		envelope:  envelope,
	}
}

func (p *configUpdateEvent) visit() error {
	cu := &cb.ConfigUpdate{}
	if err := unmarshal(p.envelope.ConfigUpdate, cu); err != nil {
		if e := p.HandleError(err, p.newContext(UnmarshalErr)); e != nil {
			return newVisitorError(e)
		}

		logger.Warningf("[%s] Error unmarshalling config update: %s", p.channelID, err)
		return err
	}

	logger.Debugf("[%s] Publishing Config Update [%s] in block #%d", p.channelID, cu, p.blockNum)

	update := &ConfigUpdate{
		BlockNum:     p.blockNum,
		ConfigUpdate: cu,
	}

	if err := p.HandleConfigUpdate(update); err != nil {
		if e := p.HandleError(err, p.newContext(ConfigUpdateHandlerErr, withConfigUpdate(update))); e != nil {
			return newVisitorError(e)
		}
	}

	return nil
}

func (p *configUpdateEvent) newContext(category Category, opts ...ctxOpt) *Context {
	return newContext(category, p.channelID, p.blockNum, opts...)
}

func extractPayloadAndHeader(envelope *cb.Envelope) (*cb.Payload, *cb.ChannelHeader, error) {
	payload, err := extractPayload(envelope)
	if err != nil {
		return nil, nil, err
	}

	chdr, err := unmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		return nil, nil, err
	}

	return payload, chdr, nil
}

func getCCInfo(writes []*kvrwset.KVWrite) (string, *ccprovider.ChaincodeData, *pb.CollectionConfigPackage, error) {
	var ccID string
	var ccp *pb.CollectionConfigPackage
	var ccData *ccprovider.ChaincodeData

	for _, kvWrite := range writes {
		if isCollectionConfigKey(kvWrite.Key) {
			ccp = &pb.CollectionConfigPackage{}
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
		HandleCCEvent:       func(*CCEvent) error { return nil },
		HandleRead:          func(*Read) error { return nil },
		HandleWrite:         func(*Write) error { return nil },
		HandleCollHashRead:  func(*CollHashRead) error { return nil },
		HandleCollHashWrite: func(*CollHashWrite) error { return nil },
		HandleLSCCWrite:     func(*LSCCWrite) error { return nil },
		HandleConfigUpdate:  func(*ConfigUpdate) error { return nil },
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
	return util.ExtractPayload(envelope)
}

var unmarshalChannelHeader = func(bytes []byte) (*cb.ChannelHeader, error) {
	return protoutil.UnmarshalChannelHeader(bytes)
}

var getTransaction = func(txBytes []byte) (*pb.Transaction, error) {
	return util.GetTransaction(txBytes)
}

var getChaincodeActionPayload = func(capBytes []byte) (*pb.ChaincodeActionPayload, error) {
	return util.GetChaincodeActionPayload(capBytes)
}

type visitorError struct {
	cause error
}

func newVisitorError(cuase error) visitorError {
	return visitorError{cause: cuase}
}

func (e visitorError) Error() string {
	return e.cause.Error()
}

func (e visitorError) Cause() error {
	return e.cause
}

// haltOnError returns true if the error was handled by an error handler
// and processing should halt because of the error
func haltOnError(err error) bool {
	_, ok := err.(visitorError)
	return ok
}
