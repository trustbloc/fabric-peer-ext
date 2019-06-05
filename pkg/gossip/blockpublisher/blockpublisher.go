/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockpublisher

import (
	"sync"
	"sync/atomic"

	"github.com/bluele/gcache"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
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
	lsccID       = "lscc"
	upgradeEvent = "upgrade"
)

var logger = flogging.MustGetLogger("ext_blockpublisher")

type write struct {
	blockNum  uint64
	txID      string
	namespace string
	w         *kvrwset.KVWrite
}

type read struct {
	blockNum  uint64
	txID      string
	namespace string
	r         *kvrwset.KVRead
}

type ccEvent struct {
	blockNum uint64
	txID     string
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

// Publisher traverses a block and publishes KV read, KV write, and chaincode events to registered handlers
type Publisher struct {
	channelID            string
	writeHandlers        []api.WriteHandler
	readHandlers         []api.ReadHandler
	ccEventHandlers      []api.ChaincodeEventHandler
	configUpdateHandlers []api.ConfigUpdateHandler
	mutex                sync.RWMutex
	blockChan            chan *cb.Block
	wChan                chan *write
	rChan                chan *read
	ccEvtChan            chan *ccEvent
	configUpdateChan     chan *configUpdate
	doneChan             chan struct{}
	closed               uint32
}

// New returns a new block Publisher for the given channel
func New(channelID string) *Publisher {
	bufferSize := config.GetBlockPublisherBufferSize()

	p := &Publisher{
		channelID:        channelID,
		blockChan:        make(chan *cb.Block, bufferSize),
		wChan:            make(chan *write, bufferSize),
		rChan:            make(chan *read, bufferSize),
		ccEvtChan:        make(chan *ccEvent, bufferSize),
		configUpdateChan: make(chan *configUpdate, bufferSize),
		doneChan:         make(chan struct{}),
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

// Publish publishes a block
func (p *Publisher) Publish(block *cb.Block) {
	newBlockEvent(p.channelID, block, p.wChan, p.rChan, p.ccEvtChan, p.configUpdateChan).publish()
}

func (p *Publisher) listen() {
	for {
		select {
		case w := <-p.wChan:
			p.handleWrite(w)
		case r := <-p.rChan:
			p.handleRead(r)
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
		if err := handleRead(r.blockNum, p.channelID, r.txID, r.namespace, r.r); err != nil {
			logger.Warningf("[%s] Error returned from KV read handler: %s", p.channelID, err)
		}
	}
}

func (p *Publisher) handleWrite(w *write) {
	logger.Debugf("[%s] Handling write: [%s]", p.channelID, w)
	for _, handleWrite := range p.getWriteHandlers() {
		if err := handleWrite(w.blockNum, p.channelID, w.txID, w.namespace, w.w); err != nil {
			logger.Warningf("[%s] Error returned from KV write handler: %s", p.channelID, err)
		}
	}
}

func (p *Publisher) handleCCEvent(event *ccEvent) {
	logger.Debugf("[%s] Handling chaincode event: [%s]", p.channelID, event)
	for _, handleCCEvent := range p.getCCEventHandlers() {
		if err := handleCCEvent(event.blockNum, p.channelID, event.txID, event.event); err != nil {
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
	channelID        string
	block            *cb.Block
	wChan            chan<- *write
	rChan            chan<- *read
	ccEvtChan        chan<- *ccEvent
	configUpdateChan chan<- *configUpdate
}

func newBlockEvent(channelID string, block *cb.Block, wChan chan<- *write, rChan chan<- *read, ccEvtChan chan<- *ccEvent, configUpdateChan chan<- *configUpdate) *blockEvent {
	return &blockEvent{
		channelID:        channelID,
		block:            block,
		wChan:            wChan,
		rChan:            rChan,
		ccEvtChan:        ccEvtChan,
		configUpdateChan: configUpdateChan,
	}
}

func (p *blockEvent) publish() {
	logger.Debugf("[%s] Publishing block #%d", p.channelID, p.block.Header.Number)
	for i := range p.block.Data.Data {
		envelope, err := protoutil.ExtractEnvelope(p.block, i)
		if err != nil {
			logger.Warningf("[%s] Error extracting envelope at index %d in block %d: %s", p.channelID, i, p.block.Header.Number, err)
		} else {
			err = p.visitEnvelope(i, envelope)
			if err != nil {
				logger.Warningf("[%s] Error checking envelope at index %d in block %d: %s", p.channelID, i, p.block.Header.Number, err)
			}
		}
	}
}

func (p *blockEvent) visitEnvelope(i int, envelope *cb.Envelope) error {
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
		code := txFilter.Flag(i)
		if code != pb.TxValidationCode_VALID {
			logger.Debugf("[%s] Transaction at index %d in block %d is not valid. Status code: %s", p.channelID, i, p.block.Header.Number, code)
			return nil
		}
		tx, err := protoutil.GetTransaction(payload.Data)
		if err != nil {
			return err
		}
		newTxEvent(p.channelID, p.block.Header.Number, chdr.TxId, tx, p.wChan, p.rChan, p.ccEvtChan).publish()
		return nil
	}

	if cb.HeaderType(chdr.Type) == cb.HeaderType_CONFIG_UPDATE {
		envelope := &cb.ConfigUpdateEnvelope{}
		if err := proto.Unmarshal(payload.Data, envelope); err != nil {
			return err
		}
		newConfigUpdateEvent(p.channelID, p.block.Header.Number, envelope, p.configUpdateChan).publish()
		return nil
	}

	return nil
}

type txEvent struct {
	channelID string
	blockNum  uint64
	txID      string
	tx        *pb.Transaction
	wChan     chan<- *write
	rChan     chan<- *read
	ccEvtChan chan<- *ccEvent
}

func newTxEvent(channelID string, blockNum uint64, txID string, tx *pb.Transaction, wChan chan<- *write, rChan chan<- *read, ccEvtChan chan<- *ccEvent) *txEvent {
	return &txEvent{
		channelID: channelID,
		blockNum:  blockNum,
		txID:      txID,
		tx:        tx,
		wChan:     wChan,
		rChan:     rChan,
		ccEvtChan: ccEvtChan,
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
	for _, w := range nsRWSet.KvRwSet.Writes {
		p.wChan <- &write{
			blockNum:  p.blockNum,
			txID:      p.txID,
			namespace: nsRWSet.NameSpace,
			w:         w,
		}
	}
}

type configUpdateEvent struct {
	channelID        string
	blockNum         uint64
	envelope         *cb.ConfigUpdateEnvelope
	configUpdateChan chan<- *configUpdate
}

func newConfigUpdateEvent(channelID string, blockNum uint64, envelope *cb.ConfigUpdateEnvelope, configUpdateChan chan<- *configUpdate) *configUpdateEvent {
	return &configUpdateEvent{
		channelID:        channelID,
		blockNum:         blockNum,
		envelope:         envelope,
		configUpdateChan: configUpdateChan,
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
	return func(blockNum uint64, channelID string, txID string, event *pb.ChaincodeEvent) error {
		logger.Debugf("[%s] Handling chaincode event: %s", channelID, event)
		if event.ChaincodeId != lsccID {
			logger.Debugf("[%s] Chaincode event is not from 'lscc'", channelID)
			return nil
		}
		if event.EventName != upgradeEvent {
			logger.Debugf("[%s] Chaincode event from 'lscc' is not an upgrade event", channelID)
			return nil
		}

		ccData := &pb.LifecycleEvent{}
		err := proto.Unmarshal(event.Payload, ccData)
		if err != nil {
			return errors.WithMessage(err, "error unmarshalling chaincode upgrade event")
		}

		logger.Debugf("[%s] Handling chaincode upgrade of chaincode [%s]", channelID, ccData.ChaincodeName)
		return handleUpgrade(blockNum, txID, ccData.ChaincodeName)
	}
}
