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
	cb "github.com/hyperledger/fabric-protos-go/common"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/extensions/gossip/api"
	"github.com/pkg/errors"
	collcommon "github.com/trustbloc/fabric-peer-ext/pkg/collections/common"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/blockvisitor"
	"github.com/trustbloc/fabric-peer-ext/pkg/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/resource"
)

var logger = flogging.MustGetLogger("ext_blockpublisher")

const upgradeEvent = "upgrade"

// Provider maintains a cache of Block Publishers - one per channel
type Provider struct {
	cache          gcache.Cache
	ledgerProvider collcommon.LedgerProvider
}

// NewProvider returns a new block publisher provider
func NewProvider() *Provider {
	p := &Provider{
		cache: gcache.New(0).LoaderFunc(func(channelID interface{}) (interface{}, error) {
			return New(channelID.(string)), nil
		}).Build(),
	}
	resource.Mgr.Register(p.Initialize)
	return p
}

// Initialize is called by the resource manager to initialize the block publisher provider
func (p *Provider) Initialize(ledgerProvider collcommon.LedgerProvider) *Provider {
	logger.Infof("Initializing block publisher provider")
	p.ledgerProvider = ledgerProvider
	return p
}

// ChannelJoined is invoked when the peer joins a channel.
// This function initializes the value of lastCommittedBlock for the channel.
func (p *Provider) ChannelJoined(channelID string) {
	bcInfo, err := p.ledgerProvider.GetLedger(channelID).GetBlockchainInfo()
	if err != nil {
		logger.Errorf("Error getting blockchain info for channel [%s]", channelID)
		return
	}

	logger.Infof("Channel [%s] joined. Initializing block publisher with ledger height %d", channelID, bcInfo.Height)
	publisher := p.ForChannel(channelID).(*Publisher)
	publisher.SetLastCommittedBlockNum(bcInfo.Height - 1)
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
	*blockvisitor.Visitor
	*channels
	writeHandlers        []api.WriteHandler
	readHandlers         []api.ReadHandler
	lsccWriteHandlers    []api.LSCCWriteHandler
	ccEventHandlers      []api.ChaincodeEventHandler
	configUpdateHandlers []api.ConfigUpdateHandler
	mutex                sync.RWMutex
	doneChan             chan struct{}
	closed               uint32
}

// New returns a new block Publisher for the given channel
func New(channelID string) *Publisher {
	channels := newChannels(config.GetBlockPublisherBufferSize())

	p := &Publisher{
		channels: channels,
		doneChan: make(chan struct{}),
		Visitor: blockvisitor.New(
			channelID,
			blockvisitor.WithNoStopOnError(),
			blockvisitor.WithCCEventHandler(channels.sendCCEvent),
			blockvisitor.WithReadHandler(channels.sendRead),
			blockvisitor.WithWriteHandler(channels.sendWrite),
			blockvisitor.WithLSCCWriteHandler(channels.sendLSCCWrite),
			blockvisitor.WithConfigUpdateHandler(channels.sendConfigUpdate),
		),
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
		logger.Debugf("[%s] Block Publisher already closed", p.ChannelID())
	}
}

// AddConfigUpdateHandler adds a handler for config update events
func (p *Publisher) AddConfigUpdateHandler(handler api.ConfigUpdateHandler) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	logger.Debugf("[%s] Adding config update", p.ChannelID())
	p.configUpdateHandlers = append(p.configUpdateHandlers, handler)
}

// AddWriteHandler adds a new handler for KV writes
func (p *Publisher) AddWriteHandler(handler api.WriteHandler) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	logger.Debugf("[%s] Adding write handler", p.ChannelID())
	p.writeHandlers = append(p.writeHandlers, handler)
}

// AddReadHandler adds a new handler for KV reads
func (p *Publisher) AddReadHandler(handler api.ReadHandler) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	logger.Debugf("[%s] Adding read handler", p.ChannelID())
	p.readHandlers = append(p.readHandlers, handler)
}

// AddCCEventHandler adds a new handler for chaincode events
func (p *Publisher) AddCCEventHandler(handler api.ChaincodeEventHandler) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	logger.Debugf("[%s] Adding chaincode event handler", p.ChannelID())
	p.ccEventHandlers = append(p.ccEventHandlers, handler)
}

// AddCCUpgradeHandler adds a handler for chaincode upgrade events
func (p *Publisher) AddCCUpgradeHandler(handler api.ChaincodeUpgradeHandler) {
	logger.Debugf("[%s] Adding chaincode upgrade handler", p.ChannelID())
	p.AddCCEventHandler(newChaincodeUpgradeHandler(handler))
}

// AddLSCCWriteHandler adds a handler for chaincode instantiation/upgrade events
func (p *Publisher) AddLSCCWriteHandler(handler api.LSCCWriteHandler) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	logger.Debugf("[%s] Adding LSCC write handler", p.ChannelID())
	p.lsccWriteHandlers = append(p.lsccWriteHandlers, handler)
}

// Publish publishes a block
func (p *Publisher) Publish(block *cb.Block) {
	if err := p.Visit(block); err != nil {
		// Errors are never expected
		panic(err.Error())
	}
}

func (p *Publisher) listen() {
	for {
		select {
		case w := <-p.wChan:
			panicOnError(p.handleWrite(w))
		case r := <-p.rChan:
			panicOnError(p.handleRead(r))
		case lscc := <-p.lsccChan:
			panicOnError(p.handleLSCCWrite(lscc))
		case ccEvt := <-p.ccEvtChan:
			panicOnError(p.handleCCEvent(ccEvt))
		case cu := <-p.configUpdateChan:
			panicOnError(p.handleConfigUpdate(cu))
		case <-p.doneChan:
			logger.Debugf("[%s] Exiting block Publisher", p.ChannelID())
			return
		}
	}
}

func (p *Publisher) handleRead(r *blockvisitor.Read) error {
	logger.Debugf("[%s] Handling read: [%s]", p.ChannelID(), r)
	for _, handleRead := range p.getReadHandlers() {
		if err := handleRead(api.TxMetadata{BlockNum: r.BlockNum, ChannelID: p.ChannelID(), TxID: r.TxID, TxNum: r.TxNum}, r.Namespace, r.Read); err != nil {
			if p.StopOnError {
				return err
			}
			logger.Warningf("[%s] Error returned from KV Read handler: %s", p.ChannelID(), err)
		}
	}
	return nil
}

func (p *Publisher) handleWrite(w *blockvisitor.Write) error {
	logger.Debugf("[%s] Handling write: [%s]", p.ChannelID(), w)
	for _, handleWrite := range p.getWriteHandlers() {
		if err := handleWrite(api.TxMetadata{BlockNum: w.BlockNum, ChannelID: p.ChannelID(), TxID: w.TxID, TxNum: w.TxNum}, w.Namespace, w.Write); err != nil {
			if p.StopOnError {
				return err
			}
			logger.Warningf("[%s] Error returned from KV Write handler: %s", p.ChannelID(), err)
		}
	}
	return nil
}

func (p *Publisher) handleLSCCWrite(w *blockvisitor.LSCCWrite) error {
	logger.Debugf("[%s] Handling LSCC write: [%s]", p.ChannelID(), w)

	for _, handler := range p.getLSCCWriteHandlers() {
		if err := handler(
			api.TxMetadata{BlockNum: w.BlockNum, ChannelID: p.ChannelID(), TxID: w.TxID, TxNum: w.TxNum}, w.CCID, w.CCData, w.CCP); err != nil {
			if p.StopOnError {
				return err
			}
			logger.Warningf("[%s] Error returned from LSCC Write handler: %s", p.ChannelID(), err)
		}
	}
	return nil
}

func (p *Publisher) handleCCEvent(event *blockvisitor.CCEvent) error {
	logger.Debugf("[%s] Handling chaincode event: [%s]", p.ChannelID(), event)
	for _, handleCCEvent := range p.getCCEventHandlers() {
		if err := handleCCEvent(api.TxMetadata{BlockNum: event.BlockNum, ChannelID: p.ChannelID(), TxID: event.TxID, TxNum: event.TxNum}, event.Event); err != nil {
			if p.StopOnError {
				return err
			}
			logger.Warningf("[%s] Error returned from CC event handler: %s", p.ChannelID(), err)
		}
	}
	return nil
}

func (p *Publisher) handleConfigUpdate(cu *blockvisitor.ConfigUpdate) error {
	logger.Debugf("[%s] Handling config update [%s]", p.ChannelID(), cu)
	for _, handleConfigUpdate := range p.getConfigUpdateHandlers() {
		if err := handleConfigUpdate(cu.BlockNum, cu.ConfigUpdate); err != nil {
			if p.StopOnError {
				return err
			}
			logger.Warningf("[%s] Error returned from config update handler: %s", p.ChannelID(), err)
		}
	}
	return nil
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

type channels struct {
	blockChan        chan *cb.Block
	wChan            chan *blockvisitor.Write
	rChan            chan *blockvisitor.Read
	lsccChan         chan *blockvisitor.LSCCWrite
	ccEvtChan        chan *blockvisitor.CCEvent
	configUpdateChan chan *blockvisitor.ConfigUpdate
}

func newChannels(bufferSize int) *channels {
	return &channels{
		blockChan:        make(chan *cb.Block, bufferSize),
		wChan:            make(chan *blockvisitor.Write, bufferSize),
		rChan:            make(chan *blockvisitor.Read, bufferSize),
		lsccChan:         make(chan *blockvisitor.LSCCWrite, bufferSize),
		ccEvtChan:        make(chan *blockvisitor.CCEvent, bufferSize),
		configUpdateChan: make(chan *blockvisitor.ConfigUpdate, bufferSize),
	}
}

func (c *channels) sendRead(r *blockvisitor.Read) error {
	c.rChan <- r
	return nil
}

func (c *channels) sendWrite(w *blockvisitor.Write) error {
	c.wChan <- w
	return nil
}

func (c *channels) sendLSCCWrite(w *blockvisitor.LSCCWrite) error {
	c.lsccChan <- w
	return nil
}

func (c *channels) sendCCEvent(event *blockvisitor.CCEvent) error {
	c.ccEvtChan <- event
	return nil
}

func (c *channels) sendConfigUpdate(cu *blockvisitor.ConfigUpdate) error {
	c.configUpdateChan <- cu
	return nil
}

func newChaincodeUpgradeHandler(handleUpgrade api.ChaincodeUpgradeHandler) api.ChaincodeEventHandler {
	return func(txnMetadata api.TxMetadata, event *pb.ChaincodeEvent) error {
		logger.Debugf("[%s] Handling chaincode event: %s", txnMetadata.ChannelID, event)
		if event.ChaincodeId != blockvisitor.LsccID {
			logger.Debugf("[%s] Chaincode event is not from 'lscc'", txnMetadata.ChannelID)
			return nil
		}
		if event.EventName != upgradeEvent {
			logger.Debugf("[%s] Chaincode event from 'lscc' is not an upgrade Event", txnMetadata.ChannelID)
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

// panicOnError panics if there's an error.
func panicOnError(err error) {
	if err != nil {
		panic(err.Error())
	}
}
