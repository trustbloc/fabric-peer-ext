/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package state

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bluele/gcache"
	extcommon "github.com/trustbloc/fabric-peer-ext/pkg/common"
	extdiscovery "github.com/trustbloc/fabric-peer-ext/pkg/common/discovery"
	"github.com/trustbloc/fabric-peer-ext/pkg/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/gossip/appdata"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
	extstatedb "github.com/trustbloc/fabric-peer-ext/pkg/statedb"
)

const cacheUpdatesDataType = "cache-updates"

var noCacheUpdates = &noopCacheUpdater{}
var errNotFound = fmt.Errorf("cache updates not found for block")

// SaveCacheUpdates is called on the committing peer to save the state cache updates so that the results
// may be published to an endorsing peer when asked for via a Gossip request.
func SaveCacheUpdates(channelID string, blockNum uint64, updates []byte) {
	logger.Debugf("[%s] Storing cache updates for block [%d]", channelID, blockNum)

	updateHandler.getCacheUpdater(channelID).Put(blockNum, updates)
}

// UpdateCache is called on the endorsing peer to pre-populate the state cache.
// When a validated block is received from the committing peer, a Gossip request is made to the committing
// peer for cache-updates for that block. When the cache-updates are received then the cache on the endorsing
// peer is updated (pre-populated).
func UpdateCache(channelID string, blockNum uint64) error {
	return updateHandler.getCacheUpdater(channelID).UpdateCache(blockNum)
}

type cacheUpdater interface {
	UpdateCache(blockNum uint64) error
	Put(blockNum uint64, updates []byte)
	GetCacheUpdatesForBlock(blockNum uint64) ([]byte, error)
}

type cacheUpdatesRequest struct {
	ChannelID string
	BlockNum  uint64
}

type gossipCacheUpdater struct {
	stateRetriever
	channelID string
	stateDB   extstatedb.StateDB
	updates   gcache.Cache
	mspID     string
	timeout   time.Duration
}

func (u *gossipCacheUpdater) UpdateCache(blockNum uint64) error {
	logger.Debugf("[%s] Requesting cache updates for block [%d] from committing peer", u.channelID, blockNum)

	reqBytes, err := jsonMarshal(&cacheUpdatesRequest{
		ChannelID: u.channelID,
		BlockNum:  blockNum,
	})
	if err != nil {
		return err
	}

	ctxt, cancel := context.WithTimeout(context.Background(), u.timeout)
	defer cancel()

	req := &appdata.Request{
		DataType: cacheUpdatesDataType,
		Payload:  reqBytes,
	}

	responses, err := u.Retrieve(ctxt,
		req,
		func(response []byte) (extcommon.Values, error) {
			return extcommon.Values{response}, nil
		},
		func(values extcommon.Values) bool {
			return len(values) > 0 && values.AllSet()
		},
		appdata.WithPeerFilter(func(peer *extdiscovery.Member) bool {
			logger.Debugf("Checking peer: %s, MSP: %s, Roles: %s", peer, peer.MSPID, peer.Roles())
			return peer.MSPID == u.mspID && peer.HasRole(roles.CommitterRole)
		}),
	)
	if err != nil {
		logger.Infof("[%s] Got error requesting cache updates for block [%d] from remote peer: %s", u.channelID, blockNum, err)

		return err
	}

	if len(responses) == 0 {
		logger.Debugf("[%s] No cache update responses received from committing peer for block [%d]", u.channelID, blockNum)

		return nil
	}

	cacheUpdates := responses[0].([]byte)

	logger.Debugf("[%s] Updating state cache for block [%d]. Cache updates size: %d bytes", u.channelID, blockNum, len(cacheUpdates))

	return u.stateDB.UpdateCache(blockNum, cacheUpdates)
}

func (u *gossipCacheUpdater) GetCacheUpdatesForBlock(blockNum uint64) ([]byte, error) {
	updates, err := u.updates.Get(blockNum)
	if err != nil {
		if err == gcache.KeyNotFoundError {
			return nil, errNotFound
		}

		// Should never happen
		return nil, err
	}

	logger.Debugf("[%s] Returning cache updates for block [%d]", u.channelID, blockNum)

	return updates.([]byte), nil
}

func (u *gossipCacheUpdater) Put(blockNum uint64, updates []byte) {
	logger.Debugf("[%s] Saving cache updates for block [%d]", u.channelID, blockNum)

	if err := u.updates.Set(blockNum, updates); err != nil {
		logger.Errorf("[%s] Error setting cache updates", u.channelID)
	}
}

type noopCacheUpdater struct {
}

func (u *noopCacheUpdater) UpdateCache(uint64) error {
	// Nothing to do
	return nil
}

func (u *noopCacheUpdater) GetCacheUpdatesForBlock(uint64) ([]byte, error) {
	// Nothing to do
	return nil, errNotFound
}

func (u *noopCacheUpdater) Put(uint64, []byte) {
	// Nothing to do
}

var isPrePopulateStateCache = func() bool {
	return config.IsPrePopulateStateCache() && roles.IsClustered()
}

var jsonMarshal = func(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
