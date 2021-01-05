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
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/extensions/gossip/api"
	"github.com/pkg/errors"

	collcommon "github.com/trustbloc/fabric-peer-ext/pkg/collections/common"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/blockvisitor"
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
	writeHandlers         []api.WriteHandler
	readHandlers          []api.ReadHandler
	collHashWriteHandlers []api.CollHashWriteHandler
	collHashReadHandlers  []api.CollHashReadHandler
	lsccWriteHandlers     []api.LSCCWriteHandler
	ccEventHandlers       []api.ChaincodeEventHandler
	configUpdateHandlers  []api.ConfigUpdateHandler
	blockHandlers         []api.PublishedBlockHandler
	mutex                 sync.RWMutex
	doneChan              chan struct{}
	closed                uint32
}

// New returns a new block Publisher for the given channel
func New(channelID string) *Publisher {
	channels := newChannels()

	p := &Publisher{
		channels: channels,
		doneChan: make(chan struct{}),
		Visitor: blockvisitor.New(
			channelID,
			blockvisitor.WithCCEventHandler(channels.sendCCEvent),
			blockvisitor.WithReadHandler(channels.sendRead),
			blockvisitor.WithWriteHandler(channels.sendWrite),
			blockvisitor.WithCollHashReadHandler(channels.sendCollHashRead),
			blockvisitor.WithCollHashWriteHandler(channels.sendCollHashWrite),
			blockvisitor.WithLSCCWriteHandler(channels.sendLSCCWrite),
			blockvisitor.WithConfigUpdateHandler(channels.sendConfigUpdate),
			blockvisitor.WithBlockHandler(channels.sendBlock),
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

// AddCollHashWriteHandler adds a new handler for KV collection hash writes
func (p *Publisher) AddCollHashWriteHandler(handler api.CollHashWriteHandler) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	logger.Debugf("[%s] Adding collection hash write handler", p.ChannelID())
	p.collHashWriteHandlers = append(p.collHashWriteHandlers, handler)
}

// AddCollHashReadHandler adds a new handler for KV collection hash reads
func (p *Publisher) AddCollHashReadHandler(handler api.CollHashReadHandler) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	logger.Debugf("[%s] Adding collection hash read handler", p.ChannelID())
	p.collHashReadHandlers = append(p.collHashReadHandlers, handler)
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

// AddBlockHandler adds a handler for published blocks
func (p *Publisher) AddBlockHandler(handler api.PublishedBlockHandler) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	logger.Debugf("[%s] Adding block handler", p.ChannelID())
	p.blockHandlers = append(p.blockHandlers, handler)
}

// Publish publishes a block
func (p *Publisher) Publish(block *cb.Block, pvtData ledger.TxPvtDataMap) {
	if err := p.Visit(block, pvtData); err != nil {
		// Errors are never expected
		panic(err.Error())
	}
}

func (p *Publisher) listen() { // nolint: gocyclo
	for {
		select {
		case w := <-p.wChan:
			p.handleWrite(w)
		case r := <-p.rChan:
			p.handleRead(r)
		case w := <-p.wCollHashChan:
			p.handleCollHashWrite(w)
		case r := <-p.rCollHashChan:
			p.handleCollHashRead(r)
		case lscc := <-p.lsccChan:
			p.handleLSCCWrite(lscc)
		case ccEvt := <-p.ccEvtChan:
			p.handleCCEvent(ccEvt)
		case cu := <-p.configUpdateChan:
			p.handleConfigUpdate(cu)
		case b := <-p.blockChan:
			p.handleBlock(b)
		case <-p.doneChan:
			logger.Debugf("[%s] Exiting block Publisher", p.ChannelID())
			return
		}
	}
}

func (p *Publisher) handleRead(r *blockvisitor.Read) {
	logger.Debugf("[%s] Handling read: [%s]", p.ChannelID(), r)
	for _, handleRead := range p.getReadHandlers() {
		if err := handleRead(api.TxMetadata{BlockNum: r.BlockNum, ChannelID: p.ChannelID(), TxID: r.TxID, TxNum: r.TxNum}, r.Namespace, r.Read); err != nil {
			logger.Warningf("[%s] Error returned from KV Read handler: %s", p.ChannelID(), err)
		}
	}
}

func (p *Publisher) handleWrite(w *blockvisitor.Write) {
	logger.Debugf("[%s] Handling write: [%s]", p.ChannelID(), w)
	for _, handleWrite := range p.getWriteHandlers() {
		if err := handleWrite(api.TxMetadata{BlockNum: w.BlockNum, ChannelID: p.ChannelID(), TxID: w.TxID, TxNum: w.TxNum}, w.Namespace, w.Write); err != nil {
			logger.Warningf("[%s] Error returned from KV Write handler: %s", p.ChannelID(), err)
		}
	}
}

func (p *Publisher) handleCollHashRead(r *blockvisitor.CollHashRead) {
	logger.Debugf("[%s] Handling collection hash read: [%s]", p.ChannelID(), r)
	for _, handleRead := range p.getCollHashReadHandlers() {
		if err := handleRead(api.TxMetadata{BlockNum: r.BlockNum, ChannelID: p.ChannelID(), TxID: r.TxID, TxNum: r.TxNum}, r.Namespace, r.Collection, r.Read); err != nil {
			logger.Warningf("[%s] Error returned from KV collection hash Read handler: %s", p.ChannelID(), err)
		}
	}
}

func (p *Publisher) handleCollHashWrite(w *blockvisitor.CollHashWrite) {
	logger.Debugf("[%s] Handling collection hash write: [%s]", p.ChannelID(), w)
	for _, handleWrite := range p.getCollHashWriteHandlers() {
		if err := handleWrite(api.TxMetadata{BlockNum: w.BlockNum, ChannelID: p.ChannelID(), TxID: w.TxID, TxNum: w.TxNum}, w.Namespace, w.Collection, w.Write); err != nil {
			logger.Warningf("[%s] Error returned from KV collection hash Write handler: %s", p.ChannelID(), err)
		}
	}
}

func (p *Publisher) handleLSCCWrite(w *blockvisitor.LSCCWrite) {
	logger.Debugf("[%s] Handling LSCC write: [%s]", p.ChannelID(), w)

	for _, handler := range p.getLSCCWriteHandlers() {
		if err := handler(
			api.TxMetadata{BlockNum: w.BlockNum, ChannelID: p.ChannelID(), TxID: w.TxID, TxNum: w.TxNum}, w.CCID, w.CCData, w.CCP); err != nil {
			logger.Warningf("[%s] Error returned from LSCC Write handler: %s", p.ChannelID(), err)
		}
	}
}

func (p *Publisher) handleCCEvent(event *blockvisitor.CCEvent) {
	logger.Debugf("[%s] Handling chaincode event: [%s]", p.ChannelID(), event)
	for _, handleCCEvent := range p.getCCEventHandlers() {
		if err := handleCCEvent(api.TxMetadata{BlockNum: event.BlockNum, ChannelID: p.ChannelID(), TxID: event.TxID, TxNum: event.TxNum}, event.Event); err != nil {
			logger.Warningf("[%s] Error returned from CC event handler: %s", p.ChannelID(), err)
		}
	}
}

func (p *Publisher) handleConfigUpdate(cu *blockvisitor.ConfigUpdate) {
	logger.Debugf("[%s] Handling config update [%s]", p.ChannelID(), cu)
	for _, handleConfigUpdate := range p.getConfigUpdateHandlers() {
		if err := handleConfigUpdate(cu.BlockNum, cu.ConfigUpdate); err != nil {
			logger.Warningf("[%s] Error returned from config update handler: %s", p.ChannelID(), err)
		}
	}
}

func (p *Publisher) handleBlock(block *cb.Block) {
	logger.Debugf("[%s] Handling block: [%d]", p.ChannelID(), block.Header.Number)

	for _, handleBlock := range p.getBlockHandlers() {
		if err := handleBlock(block); err != nil {
			logger.Warningf("[%s] Error returned from block handler: %s", p.ChannelID(), err)
		}
	}
}

func (p *Publisher) getReadHandlers() []api.ReadHandler {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.readHandlers
}

func (p *Publisher) getWriteHandlers() []api.WriteHandler {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.writeHandlers
}

func (p *Publisher) getCollHashReadHandlers() []api.CollHashReadHandler {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.collHashReadHandlers
}

func (p *Publisher) getCollHashWriteHandlers() []api.CollHashWriteHandler {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.collHashWriteHandlers
}

func (p *Publisher) getLSCCWriteHandlers() []api.LSCCWriteHandler {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.lsccWriteHandlers
}

func (p *Publisher) getCCEventHandlers() []api.ChaincodeEventHandler {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.ccEventHandlers
}

func (p *Publisher) getConfigUpdateHandlers() []api.ConfigUpdateHandler {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.configUpdateHandlers
}

func (p *Publisher) getBlockHandlers() []api.PublishedBlockHandler {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.blockHandlers
}

type channels struct {
	blockChan        chan *cb.Block
	wChan            chan *blockvisitor.Write
	rChan            chan *blockvisitor.Read
	wCollHashChan    chan *blockvisitor.CollHashWrite
	rCollHashChan    chan *blockvisitor.CollHashRead
	lsccChan         chan *blockvisitor.LSCCWrite
	ccEvtChan        chan *blockvisitor.CCEvent
	configUpdateChan chan *blockvisitor.ConfigUpdate
}

func newChannels() *channels {
	return &channels{
		blockChan:        make(chan *cb.Block),
		wChan:            make(chan *blockvisitor.Write),
		rChan:            make(chan *blockvisitor.Read),
		wCollHashChan:    make(chan *blockvisitor.CollHashWrite),
		rCollHashChan:    make(chan *blockvisitor.CollHashRead),
		lsccChan:         make(chan *blockvisitor.LSCCWrite),
		ccEvtChan:        make(chan *blockvisitor.CCEvent),
		configUpdateChan: make(chan *blockvisitor.ConfigUpdate),
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

func (c *channels) sendCollHashRead(r *blockvisitor.CollHashRead) error {
	c.rCollHashChan <- r
	return nil
}

func (c *channels) sendCollHashWrite(w *blockvisitor.CollHashWrite) error {
	c.wCollHashChan <- w
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

func (c *channels) sendBlock(b *cb.Block) error {
	c.blockChan <- b
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
