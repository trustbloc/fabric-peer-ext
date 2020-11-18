/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cdbblkstorage

import (
	"fmt"
	"testing"

	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/stretchr/testify/require"

	blkstoremocks "github.com/trustbloc/fabric-peer-ext/pkg/blkstorage/cdbblkstorage/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

//go:generate counterfeiter -o ./mocks/retriever.gen.go --fake-name Retriever . retriever

const (
	channel1 = "channel1"
)

func TestCache(t *testing.T) {
	bcInfo := &common.BlockchainInfo{Height: 1001}
	block := mocks.NewBlockBuilder(channel1, 1000).Build()
	hash := protoutil.BlockHeaderHash(block.Header)

	t.Run("GetBlockchainInfo", func(t *testing.T) {
		c := newCache(channel1, &blkstoremocks.Retriever{}, bcInfo, config{})
		require.NotNil(t, c)

		bcInfo := c.getBlockchainInfo()
		require.Equal(t, bcInfo, bcInfo)

		bcInfo2 := &common.BlockchainInfo{Height: 1002}
		c.setBlockchainInfo(bcInfo2)

		require.Equal(t, bcInfo2, c.getBlockchainInfo())
	})

	t.Run("Block-by-num enabled", func(t *testing.T) {
		c := newCache(channel1, &blkstoremocks.Retriever{}, bcInfo, config{blockByNumSize: 100})
		require.NotNil(t, c)

		c.put(block)

		b, err := c.getBlockByNumber(1000)
		require.NoError(t, err)
		require.Equal(t, block, b)

		b, err = c.getBlockByHash(hash)
		require.NoError(t, err)
		require.Nil(t, b)
	})

	t.Run("Block-by-hash enabled", func(t *testing.T) {
		c := newCache(channel1, &blkstoremocks.Retriever{}, bcInfo, config{blockByHashSize: 100})
		require.NotNil(t, c)

		c.put(block)

		b, err := c.getBlockByNumber(1000)
		require.NoError(t, err)
		require.Nil(t, b)

		b, err = c.getBlockByHash(hash)
		require.NoError(t, err)
		require.Equal(t, block, b)
	})

	t.Run("Config block", func(t *testing.T) {
		c := newCache(channel1, &blkstoremocks.Retriever{}, bcInfo, config{blockByHashSize: 100, blockByNumSize: 100})
		require.NotNil(t, c)

		configBlock := c.getConfigBlock()
		require.Nil(t, configBlock)

		builder := mocks.NewBlockBuilder(channel1, 1000)
		builder.ConfigUpdate()
		configBlock = builder.Build()

		c.put(configBlock)

		b, err := c.getBlockByNumber(1000)
		require.NoError(t, err)
		require.Equal(t, configBlock, b)

		b, err = c.getBlockByHash(protoutil.BlockHeaderHash(configBlock.Header))
		require.NoError(t, err)
		require.Equal(t, configBlock, b)
	})

	t.Run("Load from retriever", func(t *testing.T) {
		retriever := &blkstoremocks.Retriever{}
		retriever.RetrieveBlockByHashReturns(block, nil)
		retriever.RetrieveBlockByNumberReturns(block, nil)

		c := newCache(channel1, retriever, bcInfo, config{blockByHashSize: 100, blockByNumSize: 100})
		require.NotNil(t, c)

		// Should load from retriever and save to cache
		b, err := c.getBlockByNumber(1000)
		require.NoError(t, err)
		require.Equal(t, block, b)

		b, err = c.getBlockByHash(hash)
		require.NoError(t, err)
		require.Equal(t, block, b)

		// Should retrieve from cache now
		b, err = c.getBlockByNumber(1000)
		require.NoError(t, err)
		require.Equal(t, block, b)

		b, err = c.getBlockByHash(hash)
		require.NoError(t, err)
		require.Equal(t, block, b)

		// Ensure the retriever was only called once for each
		require.Equal(t, 1, retriever.RetrieveBlockByNumberCallCount())
		require.Equal(t, 1, retriever.RetrieveBlockByHashCallCount())
	})

	t.Run("retriever error", func(t *testing.T) {
		errByHash := fmt.Errorf("injected getBlockByHash error")
		errByNumber := fmt.Errorf("injected RetrieveBlockByNumber error")

		retriever := &blkstoremocks.Retriever{}
		retriever.RetrieveBlockByHashReturns(nil, errByHash)
		retriever.RetrieveBlockByNumberReturns(nil, errByNumber)

		c := newCache(channel1, retriever, bcInfo, config{blockByHashSize: 100, blockByNumSize: 100})
		require.NotNil(t, c)

		b, err := c.getBlockByNumber(1000)
		require.EqualError(t, err, errByNumber.Error())
		require.Nil(t, b)

		b, err = c.getBlockByHash(hash)
		require.EqualError(t, err, errByHash.Error())
		require.Nil(t, b)
	})

	t.Run("Cache set error", func(t *testing.T) {
		errExpected := fmt.Errorf("injected cache Set error")

		c := newCache(channel1, &blkstoremocks.Retriever{}, bcInfo, config{})
		require.NotNil(t, c)

		c.blocksByNum = &mockCache{err: errExpected}
		c.blocksByHash = &mockCache{err: errExpected}

		require.NotPanics(t, func() { c.put(block) })
	})
}

type mockCache struct {
	err error
}

func (m *mockCache) Set(_, _ interface{}) error {
	return m.err
}

func (m *mockCache) Get(key interface{}) (interface{}, error) {
	return nil, m.err
}
