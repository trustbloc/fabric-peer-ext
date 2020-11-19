/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cdbblkstorage

import (
	"sync/atomic"

	"github.com/bluele/gcache"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric/protoutil"
)

type retriever interface {
	RetrieveBlockByNumber(blockNum uint64) (*common.Block, error)
	RetrieveBlockByHash(blockHash []byte) (*common.Block, error)
}

type cache interface {
	Set(key, value interface{}) error
	Get(key interface{}) (interface{}, error)
}

type blockCache struct {
	config
	ledgerID     string
	bcInfo       atomic.Value
	configBlock  atomic.Value
	blocksByNum  cache
	blocksByHash cache
	retriever    retriever
}

func newCache(ledgerID string, retriever retriever, bcInfo *common.BlockchainInfo, cfg config) *blockCache {
	c := &blockCache{
		config:    cfg,
		ledgerID:  ledgerID,
		retriever: retriever,
	}

	c.blocksByNum = c.createBlockByNumCache()
	c.blocksByHash = c.createBlockByHashCache()

	c.bcInfo.Store(bcInfo)

	return c
}

func (c *blockCache) createBlockByNumCache() cache {
	if c.blockByNumSize > 0 {
		logger.Infof("[%s] Creating block-by-number cache of size %d", c.ledgerID, c.blockByNumSize)

		return gcache.New(int(c.blockByNumSize)).LRU().LoaderFunc(c.loadBlockByNumber).Build()
	}

	logger.Infof("[%s] Block-by-number cache is disabled", c.ledgerID)

	return &passthroughBlockByNumCache{retriever: c.retriever}
}

func (c *blockCache) createBlockByHashCache() cache {
	if c.blockByHashSize > 0 {
		logger.Infof("[%s] Creating block-by-hash cache of size %d", c.ledgerID, c.blockByNumSize)

		return gcache.New(int(c.blockByHashSize)).LRU().LoaderFunc(c.loadBlockByHash).Build()
	}

	logger.Infof("[%s] Block-by-hash cache is disabled", c.ledgerID)

	return &passthroughBlockByHashCache{retriever: c.retriever}
}

func (c *blockCache) put(b *common.Block) {
	err := c.blocksByNum.Set(b.Header.Number, b)
	if err != nil {
		// This should never happen since the only possible error is if we provided a 'Serialize' function
		logger.Errorf("[%s] Error setting block for number [%d]: %s", c.ledgerID, b.Header.Number, err)
	}

	blockHash := protoutil.BlockHeaderHash(b.Header)

	err = c.blocksByHash.Set(string(blockHash), b)
	if err != nil {
		// This should never happen since the only possible error is if we provided a 'Serialize' function
		logger.Errorf("[%s] Error setting block [%d] for hash: %s", c.ledgerID, b.Header.Number, err)
	}

	c.setConfigBlock(b)
}

func (c *blockCache) getBlockchainInfo() *common.BlockchainInfo {
	return c.bcInfo.Load().(*common.BlockchainInfo)
}

func (c *blockCache) setBlockchainInfo(bcInfo *common.BlockchainInfo) {
	logger.Debugf("[%s] Updating blockchain info - Height: %d", c.ledgerID, bcInfo.Height)

	c.bcInfo.Store(bcInfo)
}

func (c *blockCache) getConfigBlock() *common.Block {
	v := c.configBlock.Load()
	if v == nil {
		return nil
	}

	return v.(*common.Block)
}

// setConfigBlock if the block is a config block then it will be pinned, since the
// config block is retrieved frequently and it should always remain in the cache
func (c *blockCache) setConfigBlock(b *common.Block) {
	if !c.isConfigBlock(b) {
		return
	}

	logger.Debugf("[%s] Caching config block [%d]", c.ledgerID, b.Header.Number)

	c.configBlock.Store(b)
}

func (c *blockCache) getBlockByNumber(bNum uint64) (*common.Block, error) {
	// Do a quick check for config block (since the last config block is always pinned)
	cfgBlock := c.getConfigBlock()
	if cfgBlock != nil && cfgBlock.Header.Number == bNum {
		logger.Debugf("[%s] Returning config block [%d] from cache", c.ledgerID, bNum)

		return cfgBlock, nil
	}

	b, err := c.blocksByNum.Get(bNum)
	if err != nil {
		return nil, err
	}

	logger.Debugf("[%s] Returning block from cache for number [%d]", c.ledgerID, bNum)

	return b.(*common.Block), nil
}

func (c *blockCache) getBlockByHash(hash []byte) (*common.Block, error) {
	b, err := c.blocksByHash.Get(string(hash))
	if err != nil {
		return nil, err
	}

	logger.Debugf("[%s] Returning block from cache for hash", c.ledgerID)

	return b.(*common.Block), nil
}

func (c *blockCache) loadBlockByNumber(bNum interface{}) (interface{}, error) {
	logger.Debugf("[%s] Loading block for num [%d] into cache", c.ledgerID, bNum)

	b, err := c.retriever.RetrieveBlockByNumber(bNum.(uint64))
	if err != nil {
		return nil, err
	}

	c.setConfigBlock(b)

	return b, nil
}

func (c *blockCache) loadBlockByHash(hash interface{}) (interface{}, error) {
	logger.Debugf("[%s] Loading block for hash [%s] into cache", c.ledgerID, hash)

	b, err := c.retriever.RetrieveBlockByHash([]byte(hash.(string)))
	if err != nil {
		return nil, err
	}

	c.setConfigBlock(b)

	return b, nil
}

func (c *blockCache) isConfigBlock(b *common.Block) bool {
	envelope, err := protoutil.ExtractEnvelope(b, 0)
	if err != nil {
		logger.Errorf("[%s] Error extracting envelope from block [%d]: %s", c.ledgerID, b.Header.Number, err)

		return false
	}

	header, err := protoutil.ChannelHeader(envelope)
	if err != nil {
		logger.Errorf("[%s] Error extracting envelope header from block [%d]: %s", c.ledgerID, b.Header.Number, err)

		return false
	}

	return common.HeaderType(header.Type) == common.HeaderType_CONFIG || common.HeaderType(header.Type) == common.HeaderType_CONFIG_UPDATE
}

type passthroughBlockByNumCache struct {
	retriever retriever
}

func (c *passthroughBlockByNumCache) Set(_, _ interface{}) error {
	// Nothing to do
	return nil
}

func (c *passthroughBlockByNumCache) Get(key interface{}) (interface{}, error) {
	return c.retriever.RetrieveBlockByNumber(key.(uint64))
}

type passthroughBlockByHashCache struct {
	retriever retriever
}

func (c *passthroughBlockByHashCache) Set(_, _ interface{}) error {
	// Nothing to do
	return nil
}

func (c *passthroughBlockByHashCache) Get(key interface{}) (interface{}, error) {
	return c.retriever.RetrieveBlockByHash([]byte(key.(string)))
}
