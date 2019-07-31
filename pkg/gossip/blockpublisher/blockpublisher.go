/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockpublisher

import (
	"strings"
	"sync"
	"sync/atomic"

	"github.com/bluele/gcache"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	ledgerutil "github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/extensions/gossip/api"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/config"
)

const (
	lsccID              = "lscc"
	upgradeEvent        = "upgrade"
	collectionSeparator = "~"
)

var logger = flogging.MustGetLogger("ext_blockpublisher")

type write struct {
	blockNum  uint64
	txID      string
	txNum     uint64
	namespace string
	w         *kvrwset.KVWrite
}

type read struct {
	blockNum  uint64
	txID      string
	txNum     uint64
	namespace string
	r         *kvrwset.KVRead
}

type lsccWrite struct {
	blockNum uint64
	txID     string
	txNum    uint64
	w        []*kvrwset.KVWrite
}

type ccEvent struct {
	blockNum uint64
	txID     string
	txNum    uint64
	event    *pb.ChaincodeEvent
}

type configUpdate struct {
	blockNum     uint64
	configUpdate *cb.ConfigUpdate
}

// Provider maintains a cache of Block Publishers - one per channel
type Provider struct {
	cache gcache.Cache
}

// NewProvider returns a new block publisher provider
func NewProvider() *Provider {
	return &Provider{
		cache: gcache.New(0).LoaderFunc(func(channelID interface{}) (interface{}, error) {
			return New(channelID.(string)), nil
		}).Build(),
	}
}

// ForChannel returns the block publisher for the given channel
func (p *Provider) ForChannel(channelID string) api.BlockPublisher {
	publisher, err := p.cache.Get(channelID)
	if err != nil {
		// This should never happen
		panic(err.Error())
	}
	return publisher.(*Publisher)
}

// Close closes all block publishers
func (p *Provider) Close() {
	for _, publisher := range p.cache.GetALL() {
		publisher.(*Publisher).Close()
	}
}

type channels struct {
	blockChan        chan *cb.Block
	wChan            chan *write
	rChan            chan *read
	lsccChan         chan *lsccWrite
	ccEvtChan        chan *ccEvent
	configUpdateChan chan *configUpdate
}

// Publisher traverses a block and publishes KV read, KV write, and chaincode events to registered handlers
type Publisher struct {
	*channels
	channelID            string
	writeHandlers        []api.WriteHandler
	readHandlers         []api.ReadHandler
	lsccWriteHandlers    []api.LSCCWriteHandler
	ccEventHandlers      []api.ChaincodeEventHandler
	configUpdateHandlers []api.ConfigUpdateHandler
	mutex                sync.RWMutex
	doneChan             chan struct{}
	closed               uint32
	lastCommittedBlock   uint64
}

// New returns a new block Publisher for the given channel
func New(channelID string) *Publisher {
	bufferSize := config.GetBlockPublisherBufferSize()

	p := &Publisher{
		channelID: channelID,
		channels: &channels{
			blockChan:        make(chan *cb.Block, bufferSize),
			wChan:            make(chan *write, bufferSize),
			rChan:            make(chan *read, bufferSize),
			lsccChan:         make(chan *lsccWrite, bufferSize),
			ccEvtChan:        make(chan *ccEvent, bufferSize),
			configUpdateChan: make(chan *configUpdate, bufferSize),
		},
		doneChan: make(chan struct{}),
	}
	go p.listen()
	return p
}

// Close releases all resources associated with the Publisher. Calling this function
// multiple times has no effect.
func (p *Publisher) Close() {
	if atomic.CompareAndSwapUint32(&p.closed, 0, 1) {
		p.doneChan <- struct{}{}
	} else {
		logger.Debugf("[%s] Block Publisher already closed", p.channelID)
	}
}

// AddConfigUpdateHandler adds a handler for config update events
func (p *Publisher) AddConfigUpdateHandler(handler api.ConfigUpdateHandler) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	logger.Debugf("[%s] Adding config update", p.channelID)
	p.configUpdateHandlers = append(p.configUpdateHandlers, handler)
}

// AddWriteHandler adds a new handler for KV writes
func (p *Publisher) AddWriteHandler(handler api.WriteHandler) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	logger.Debugf("[%s] Adding write", p.channelID)
	p.writeHandlers = append(p.writeHandlers, handler)
}

// AddReadHandler adds a new handler for KV reads
func (p *Publisher) AddReadHandler(handler api.ReadHandler) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	logger.Debugf("[%s] Adding read", p.channelID)
	p.readHandlers = append(p.readHandlers, handler)
}

// AddCCEventHandler adds a new handler for chaincode events
func (p *Publisher) AddCCEventHandler(handler api.ChaincodeEventHandler) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	logger.Debugf("[%s] Adding chaincode event", p.channelID)
	p.ccEventHandlers = append(p.ccEventHandlers, handler)
}

// AddCCUpgradeHandler adds a handler for chaincode upgrade events
func (p *Publisher) AddCCUpgradeHandler(handler api.ChaincodeUpgradeHandler) {
	logger.Debugf("[%s] Adding chaincode upgrade", p.channelID)
	p.AddCCEventHandler(newChaincodeUpgradeHandler(handler))
}

// AddLSCCWriteHandler adds a handler for chaincode instantiation/upgrade events
func (p *Publisher) AddLSCCWriteHandler(handler api.LSCCWriteHandler) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	logger.Debugf("[%s] Adding LSCC write handler", p.channelID)
	p.lsccWriteHandlers = append(p.lsccWriteHandlers, handler)
}

// Publish publishes a block
func (p *Publisher) Publish(block *cb.Block) {
	defer atomic.StoreUint64(&p.lastCommittedBlock, block.Header.Number)
	newBlockEvent(p.channelID, block, p.channels).publish()
}

// LedgerHeight returns ledger height based on last block published
func (p *Publisher) LedgerHeight() uint64 {
	return atomic.LoadUint64(&p.lastCommittedBlock) + 1
}

func (p *Publisher) listen() {
	for {
		select {
		case w := <-p.wChan:
			p.handleWrite(w)
		case r := <-p.rChan:
			p.handleRead(r)
		case lscc := <-p.lsccChan:
			p.handleLSCCWrite(lscc)
		case ccEvt := <-p.ccEvtChan:
			p.handleCCEvent(ccEvt)
		case cu := <-p.configUpdateChan:
			p.handleConfigUpdate(cu)
		case <-p.doneChan:
			logger.Debugf("[%s] Exiting block Publisher", p.channelID)
			return
		}
	}
}

func (p *Publisher) handleRead(r *read) {
	logger.Debugf("[%s] Handling read: [%s]", p.channelID, r)
	for _, handleRead := range p.getReadHandlers() {
		if err := handleRead(api.TxMetadata{BlockNum: r.blockNum, ChannelID: p.channelID, TxID: r.txID, TxNum: r.txNum}, r.namespace, r.r); err != nil {
			logger.Warningf("[%s] Error returned from KV read handler: %s", p.channelID, err)
		}
	}
}

func (p *Publisher) handleWrite(w *write) {
	logger.Debugf("[%s] Handling write: [%s]", p.channelID, w)
	for _, handleWrite := range p.getWriteHandlers() {
		if err := handleWrite(api.TxMetadata{BlockNum: w.blockNum, ChannelID: p.channelID, TxID: w.txID, TxNum: w.txNum}, w.namespace, w.w); err != nil {
			logger.Warningf("[%s] Error returned from KV write handler: %s", p.channelID, err)
		}
	}
}

func (p *Publisher) handleLSCCWrite(w *lsccWrite) {
	logger.Debugf("[%s] Handling LSCC write: [%s]", p.channelID, w)

	if len(p.getLSCCWriteHandlers()) == 0 {
		// No handlers registered
		return
	}

	ccID, ccData, ccp, err := getCCInfo(w.w)
	if err != nil {
		logger.Warningf("[%s] Error getting chaincode info: %s", p.channelID, err)
		return
	}

	for _, handler := range p.getLSCCWriteHandlers() {
		if err := handler(api.TxMetadata{BlockNum: w.blockNum, ChannelID: p.channelID, TxID: w.txID, TxNum: w.txNum}, ccID, ccData, ccp); err != nil {
			logger.Warningf("[%s] Error returned from LSCC write handler: %s", p.channelID, err)
		}
	}
}

func (p *Publisher) handleCCEvent(event *ccEvent) {
	logger.Debugf("[%s] Handling chaincode event: [%s]", p.channelID, event)
	for _, handleCCEvent := range p.getCCEventHandlers() {
		if err := handleCCEvent(api.TxMetadata{BlockNum: event.blockNum, ChannelID: p.channelID, TxID: event.txID, TxNum: event.txNum}, event.event); err != nil {
			logger.Warningf("[%s] Error returned from CC event handler: %s", p.channelID, err)
		}
	}
}

func (p *Publisher) handleConfigUpdate(cu *configUpdate) {
	logger.Debugf("[%s] Handling config update [%s]", p.channelID, cu)
	for _, handleConfigUpdate := range p.getConfigUpdateHandlers() {
		if err := handleConfigUpdate(cu.blockNum, cu.configUpdate); err != nil {
			logger.Warningf("[%s] Error returned from config update handler: %s", p.channelID, err)
		}
	}
}

func (p *Publisher) getReadHandlers() []api.ReadHandler {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	handlers := make([]api.ReadHandler, len(p.readHandlers))
	copy(handlers, p.readHandlers)
	return handlers
}

func (p *Publisher) getWriteHandlers() []api.WriteHandler {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	handlers := make([]api.WriteHandler, len(p.writeHandlers))
	copy(handlers, p.writeHandlers)
	return handlers
}

func (p *Publisher) getLSCCWriteHandlers() []api.LSCCWriteHandler {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	handlers := make([]api.LSCCWriteHandler, len(p.lsccWriteHandlers))
	copy(handlers, p.lsccWriteHandlers)
	return handlers
}

func (p *Publisher) getCCEventHandlers() []api.ChaincodeEventHandler {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	handlers := make([]api.ChaincodeEventHandler, len(p.ccEventHandlers))
	copy(handlers, p.ccEventHandlers)
	return handlers
}

func (p *Publisher) getConfigUpdateHandlers() []api.ConfigUpdateHandler {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	handlers := make([]api.ConfigUpdateHandler, len(p.configUpdateHandlers))
	copy(handlers, p.configUpdateHandlers)
	return handlers
}

type blockEvent struct {
	*channels
	channelID string
	block     *cb.Block
}

func newBlockEvent(channelID string, block *cb.Block, channels *channels) *blockEvent {
	return &blockEvent{
		channels:  channels,
		channelID: channelID,
		block:     block,
	}
}

func (p *blockEvent) publish() {
	logger.Debugf("[%s] Publishing block #%d", p.channelID, p.block.Header.Number)
	for txNum := range p.block.Data.Data {
		envelope, err := protoutil.ExtractEnvelope(p.block, txNum)
		if err != nil {
			logger.Warningf("[%s] Error extracting envelope at index %d in block %d: %s", p.channelID, txNum, p.block.Header.Number, err)
		} else {
			err = p.visitEnvelope(uint64(txNum), envelope)
			if err != nil {
				logger.Warningf("[%s] Error checking envelope at index %d in block %d: %s", p.channelID, txNum, p.block.Header.Number, err)
			}
		}
	}
}

func (p *blockEvent) visitEnvelope(txNum uint64, envelope *cb.Envelope) error {
	payload, err := protoutil.ExtractPayload(envelope)
	if err != nil {
		return err
	}

	chdr, err := protoutil.UnmarshalChannelHeader(payload.Header.ChannelHeader)
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
		tx, err := protoutil.GetTransaction(payload.Data)
		if err != nil {
			return err
		}
		newTxEvent(p.channelID, p.block.Header.Number, txNum, chdr.TxId, tx, p.channels).publish()
		return nil
	}

	if cb.HeaderType(chdr.Type) == cb.HeaderType_CONFIG_UPDATE {
		envelope := &cb.ConfigUpdateEnvelope{}
		if err := proto.Unmarshal(payload.Data, envelope); err != nil {
			return err
		}
		newConfigUpdateEvent(p.channelID, p.block.Header.Number, envelope, p.channels).publish()
		return nil
	}

	return nil
}

type txEvent struct {
	*channels
	channelID string
	blockNum  uint64
	txNum     uint64
	txID      string
	tx        *pb.Transaction
}

func newTxEvent(channelID string, blockNum uint64, txNum uint64, txID string, tx *pb.Transaction, channels *channels) *txEvent {
	return &txEvent{
		channelID: channelID,
		blockNum:  blockNum,
		txNum:     txNum,
		txID:      txID,
		tx:        tx,
		channels:  channels,
	}
}

func (p *txEvent) publish() {
	logger.Debugf("[%s] Publishing Tx %s in block #%d", p.channelID, p.txID, p.blockNum)
	for i, action := range p.tx.Actions {
		err := p.visitTXAction(action)
		if err != nil {
			logger.Warningf("[%s] Error checking TxAction at index %d: %s", p.channelID, i, err)
		}
	}
}

func (p *txEvent) visitTXAction(action *pb.TransactionAction) error {
	chaPayload, err := protoutil.GetChaincodeActionPayload(action.Payload)
	if err != nil {
		return err
	}
	return p.visitChaincodeActionPayload(chaPayload)
}

func (p *txEvent) visitChaincodeActionPayload(chaPayload *pb.ChaincodeActionPayload) error {
	cpp := &pb.ChaincodeProposalPayload{}
	err := proto.Unmarshal(chaPayload.ChaincodeProposalPayload, cpp)
	if err != nil {
		return err
	}

	return p.visitAction(chaPayload.Action)
}

func (p *txEvent) visitAction(action *pb.ChaincodeEndorsedAction) error {
	prp := &pb.ProposalResponsePayload{}
	err := proto.Unmarshal(action.ProposalResponsePayload, prp)
	if err != nil {
		return err
	}
	return p.visitProposalResponsePayload(prp)
}

func (p *txEvent) visitProposalResponsePayload(prp *pb.ProposalResponsePayload) error {
	chaincodeAction := &pb.ChaincodeAction{}
	err := proto.Unmarshal(prp.Extension, chaincodeAction)
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
		p.visitTxReadWriteSet(txRWSet)
	}

	if len(chaincodeAction.Events) > 0 {
		evt := &pb.ChaincodeEvent{}
		if err := proto.Unmarshal(chaincodeAction.Events, evt); err != nil {
			logger.Warningf("[%s] Invalid chaincode event for chaincode [%s]", p.channelID, chaincodeAction.ChaincodeId)
			return errors.WithMessagef(err, "invalid chaincode event for chaincode [%s]", chaincodeAction.ChaincodeId)
		}
		p.ccEvtChan <- &ccEvent{
			blockNum: p.blockNum,
			txID:     p.txID,
			event:    evt,
		}
	}

	return nil
}

func (p *txEvent) visitTxReadWriteSet(txRWSet *rwsetutil.TxRwSet) {
	for _, nsRWSet := range txRWSet.NsRwSets {
		p.visitNsReadWriteSet(nsRWSet)
	}
}

func (p *txEvent) visitNsReadWriteSet(nsRWSet *rwsetutil.NsRwSet) {
	for _, r := range nsRWSet.KvRwSet.Reads {
		p.rChan <- &read{
			blockNum:  p.blockNum,
			txID:      p.txID,
			namespace: nsRWSet.NameSpace,
			r:         r,
		}
	}
	if nsRWSet.NameSpace == lsccID {
		p.publishLSCCWrite(nsRWSet.KvRwSet.Writes)
	} else {
		p.publishWrites(nsRWSet.NameSpace, nsRWSet.KvRwSet.Writes)
	}
}

// publishLSCCWrite publishes an LSCC write event which is a result of a chaincode instantiate/upgrade.
// The event consists of two writes: CC data and collection configs.
func (p *txEvent) publishLSCCWrite(writes []*kvrwset.KVWrite) {
	p.lsccChan <- &lsccWrite{
		blockNum: p.blockNum,
		txID:     p.txID,
		w:        writes,
	}
}

func (p *txEvent) publishWrites(ns string, writes []*kvrwset.KVWrite) {
	for _, w := range writes {
		p.wChan <- &write{
			blockNum:  p.blockNum,
			txID:      p.txID,
			namespace: ns,
			w:         w,
		}
	}
}

type configUpdateEvent struct {
	*channels
	channelID string
	blockNum  uint64
	envelope  *cb.ConfigUpdateEnvelope
}

func newConfigUpdateEvent(channelID string, blockNum uint64, envelope *cb.ConfigUpdateEnvelope, channels *channels) *configUpdateEvent {
	return &configUpdateEvent{
		channels:  channels,
		channelID: channelID,
		blockNum:  blockNum,
		envelope:  envelope,
	}
}

func (p *configUpdateEvent) publish() {
	cu := &cb.ConfigUpdate{}
	if err := proto.Unmarshal(p.envelope.ConfigUpdate, cu); err != nil {
		logger.Warningf("[%s] Error unmarshalling config update: %s", p.channelID, err)
		return
	}

	logger.Debugf("[%s] Publishing Config Update [%s] in block #%d", p.channelID, cu, p.blockNum)

	p.configUpdateChan <- &configUpdate{
		blockNum:     p.blockNum,
		configUpdate: cu,
	}
}

func newChaincodeUpgradeHandler(handleUpgrade api.ChaincodeUpgradeHandler) api.ChaincodeEventHandler {
	return func(txnMetadata api.TxMetadata, event *pb.ChaincodeEvent) error {
		logger.Debugf("[%s] Handling chaincode event: %s", txnMetadata.ChannelID, event)
		if event.ChaincodeId != lsccID {
			logger.Debugf("[%s] Chaincode event is not from 'lscc'", txnMetadata.ChannelID)
			return nil
		}
		if event.EventName != upgradeEvent {
			logger.Debugf("[%s] Chaincode event from 'lscc' is not an upgrade event", txnMetadata.ChannelID)
			return nil
		}

		ccData := &pb.LifecycleEvent{}
		err := proto.Unmarshal(event.Payload, ccData)
		if err != nil {
			return errors.WithMessage(err, "error unmarshalling chaincode upgrade event")
		}

		logger.Debugf("[%s] Handling chaincode upgrade of chaincode [%s]", txnMetadata.ChannelID, ccData.ChaincodeName)
		return handleUpgrade(txnMetadata, ccData.ChaincodeName)
	}
}

func getCCInfo(writes []*kvrwset.KVWrite) (string, *ccprovider.ChaincodeData, *cb.CollectionConfigPackage, error) {
	var ccID string
	var ccp *cb.CollectionConfigPackage
	var ccData *ccprovider.ChaincodeData

	for _, kvWrite := range writes {
		if isCollectionConfigKey(kvWrite.Key) {
			ccp = &cb.CollectionConfigPackage{}
			if err := proto.Unmarshal(kvWrite.Value, ccp); err != nil {
				return "", nil, nil, errors.WithMessagef(err, "error unmarshaling collection configuration")
			}
		} else {
			ccID = kvWrite.Key
			ccData = &ccprovider.ChaincodeData{}
			if err := proto.Unmarshal(kvWrite.Value, ccData); err != nil {
				return "", nil, nil, errors.WithMessagef(err, "error unmarshaling chaincode data")
			}
		}
	}

	return ccID, ccData, ccp, nil
}

func isCollectionConfigKey(key string) bool {
	return strings.Contains(key, collectionSeparator)
}
