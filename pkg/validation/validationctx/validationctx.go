/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validationctx

import (
	"context"
	"fmt"
	"sync"

	"github.com/bluele/gcache"
	"github.com/hyperledger/fabric/common/flogging"
)

var logger = flogging.MustGetLogger("ext_validation")

type blockContext struct {
	blockNum uint64
	ctx      context.Context
	cancel   context.CancelFunc
}

// Provider is a validation context provider
type Provider struct {
	channelContexts gcache.Cache
}

// NewProvider returns a new validation context provider
func NewProvider() *Provider {
	return &Provider{
		channelContexts: gcache.New(0).LoaderFunc(func(channelID interface{}) (interface{}, error) {
			return &channelContext{
				channelID: channelID.(string),
			}, nil
		}).Build(),
	}
}

// ValidationContextForBlock returns the context for the given block number
func (p *Provider) ValidationContextForBlock(channelID string, blockNum uint64) (context.Context, error) {
	return p.get(channelID).create(blockNum)
}

// CancelBlockValidation cancels any outstanding validation for the given block number
func (p *Provider) CancelBlockValidation(channelID string, blockNum uint64) {
	p.get(channelID).cancel(blockNum)
}

func (p *Provider) get(channelID string) *channelContext {
	chCtx, err := p.channelContexts.Get(channelID)
	if err != nil {
		// Should never happen
		panic(err)
	}

	return chCtx.(*channelContext)
}

type channelContext struct {
	channelID     string
	mutex         sync.Mutex
	cancelByBlock []blockContext
}

func (c *channelContext) create(blockNum uint64) (context.Context, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	i := len(c.cancelByBlock) - 1
	if i >= 0 {
		ctx := c.cancelByBlock[i]
		if ctx.blockNum >= blockNum {
			return nil, fmt.Errorf("unable to create context for block %d since it is less than or equal to the current block %d", blockNum, ctx.blockNum)
		}
	}

	logger.Debugf("[%s] Creating context for block %d", c.channelID, blockNum)

	ctx, cancel := context.WithCancel(context.Background())

	c.cancelByBlock = append(
		c.cancelByBlock,
		blockContext{
			blockNum: blockNum,
			ctx:      ctx,
			cancel:   cancel,
		},
	)

	return ctx, nil
}

func (c *channelContext) cancel(blockNum uint64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	lastIdx := -1

	// Cancel all contexts up to and including the given block number
	for i, ctx := range c.cancelByBlock {
		if blockNum < ctx.blockNum {
			break
		}

		logger.Debugf("[%s] Cancelling context for block %d", c.channelID, blockNum)

		ctx.cancel()

		lastIdx = i
	}

	if lastIdx >= 0 {
		c.cancelByBlock = c.cancelByBlock[lastIdx+1:]
	}
}
