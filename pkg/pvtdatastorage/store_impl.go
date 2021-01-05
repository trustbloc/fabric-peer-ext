/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pvtdatastorage

import (
	cb "github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/pvtdatapolicy"
	"github.com/hyperledger/fabric/core/ledger/pvtdatastorage"
	"github.com/hyperledger/fabric/extensions/gossip/api"
	"github.com/hyperledger/fabric/extensions/gossip/blockpublisher"
	xstorageapi "github.com/hyperledger/fabric/extensions/storage/api"

	"github.com/trustbloc/fabric-peer-ext/pkg/pvtdatastorage/cachedpvtdatastore"
	cdbpvtdatastore "github.com/trustbloc/fabric-peer-ext/pkg/pvtdatastorage/cdbpvtdatastore"
	"github.com/trustbloc/fabric-peer-ext/pkg/pvtdatastorage/common"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

var logger = flogging.MustGetLogger("ext_pvtdatastore")

//////// Provider functions  /////////////
//////////////////////////////////////////

// PvtDataProvider encapsulates the storage and cache providers in addition to the missing data index provider
type PvtDataProvider struct {
	storageProvider common.Provider
	cacheProvider   common.CacheProvider
}

// NewProvider creates a new PvtDataStoreProvider that combines a cache provider and a backing storage provider
func NewProvider(conf *pvtdatastorage.PrivateDataConfig, ledgerconfig *ledger.Config) (*PvtDataProvider, error) {
	// create couchdb pvt date store provider
	storageProvider, err := cdbpvtdatastore.NewProvider(conf, ledgerconfig)
	if err != nil {
		return nil, err
	}
	// create cache pvt date store provider
	cacheProvider := cachedpvtdatastore.NewProvider()

	p := PvtDataProvider{
		storageProvider: storageProvider,
		cacheProvider:   cacheProvider,
	}
	return &p, nil
}

// OpenStore creates a pvt data store instance for the given ledger ID
func (c *PvtDataProvider) OpenStore(ledgerID string) (xstorageapi.PrivateDataStore, error) {
	pvtDataStore, err := c.storageProvider.OpenStore(ledgerID)
	if err != nil {
		return nil, err
	}

	lastCommittedBlockHeight, err := pvtDataStore.LastCommittedBlockHeight()
	if err != nil {
		return nil, err
	}

	cachePvtDataStore := c.cacheProvider.Create(ledgerID, lastCommittedBlockHeight-1)

	return newPvtDataStore(ledgerID, pvtDataStore, cachePvtDataStore, blockpublisher.ForChannel(ledgerID)), nil
}

// Close cleans up the Provider
func (c *PvtDataProvider) Close() {
	c.storageProvider.Close()
}

type pvtDataStore struct {
	ledgerID          string
	pvtDataDBStore    common.StoreExt
	cachePvtDataStore common.StoreExt
}

func newPvtDataStore(ledgerID string, pvtDataDBStore common.StoreExt, cachePvtDataStore common.StoreExt, bp api.BlockPublisher) *pvtDataStore {
	s := &pvtDataStore{
		ledgerID:          ledgerID,
		pvtDataDBStore:    pvtDataDBStore,
		cachePvtDataStore: cachePvtDataStore,
	}

	if !roles.IsCommitter() {
		// This peer is not a committer and therefore Commit will never be called. So, the last committed
		// block number needs to be updated when a committed block is published.
		bp.AddBlockHandler(s.updateLastCommittedBlockNum)
	}

	return s
}

//////// store functions  ////////////////
//////////////////////////////////////////
func (c *pvtDataStore) Init(btlPolicy pvtdatapolicy.BTLPolicy) {
	c.cachePvtDataStore.Init(btlPolicy)
	c.pvtDataDBStore.Init(btlPolicy)
}

// Prepare pvt data in cache and send pvt data to background prepare/commit go routine
func (c *pvtDataStore) Commit(blockNum uint64, pvtData []*ledger.TxPvtData, pvtMissingDataMap ledger.TxMissingPvtDataMap) error {
	// Prepare data in cache
	err := c.cachePvtDataStore.Commit(blockNum, pvtData, pvtMissingDataMap)
	if err != nil {
		return err
	}
	// Prepare data in storage
	return c.pvtDataDBStore.Commit(blockNum, pvtData, pvtMissingDataMap)
}

//GetPvtDataByBlockNum implements the function in the interface `Store`
func (c *pvtDataStore) GetPvtDataByBlockNum(blockNum uint64, filter ledger.PvtNsCollFilter) ([]*ledger.TxPvtData, error) {
	result, err := c.cachePvtDataStore.GetPvtDataByBlockNum(blockNum, filter)
	if err != nil {
		return nil, err
	}
	if len(result) > 0 {
		return result, nil
	}

	// data is not in cache will try to get it from storage
	return c.pvtDataDBStore.GetPvtDataByBlockNum(blockNum, filter)
}

//LastCommittedBlockHeight implements the function in the interface `Store`
func (c *pvtDataStore) LastCommittedBlockHeight() (uint64, error) {
	return c.pvtDataDBStore.LastCommittedBlockHeight()
}

//GetMissingPvtDataInfoForMostRecentBlocks implements the function in the interface `Store`
func (c *pvtDataStore) GetMissingPvtDataInfoForMostRecentBlocks(maxBlock int) (ledger.MissingPvtDataInfo, error) {
	return c.pvtDataDBStore.GetMissingPvtDataInfoForMostRecentBlocks(maxBlock)
}

//ProcessCollsEligibilityEnabled implements the function in the interface `Store`
func (c *pvtDataStore) ProcessCollsEligibilityEnabled(committingBlk uint64, nsCollMap map[string][]string) error {
	return c.pvtDataDBStore.ProcessCollsEligibilityEnabled(committingBlk, nsCollMap)
}

//CommitPvtDataOfOldBlocks implements the function in the interface `Store`
func (c *pvtDataStore) CommitPvtDataOfOldBlocks(blocksPvtData map[uint64][]*ledger.TxPvtData, unreconciled ledger.MissingPvtDataInfo) error {
	return c.pvtDataDBStore.CommitPvtDataOfOldBlocks(blocksPvtData, unreconciled)
}

//GetLastUpdatedOldBlocksPvtData implements the function in the interface `Store`
func (c *pvtDataStore) GetLastUpdatedOldBlocksPvtData() (map[uint64][]*ledger.TxPvtData, error) {
	return c.pvtDataDBStore.GetLastUpdatedOldBlocksPvtData()
}

//ResetLastUpdatedOldBlocksList implements the function in the interface `Store`
func (c *pvtDataStore) ResetLastUpdatedOldBlocksList() error {
	return c.pvtDataDBStore.ResetLastUpdatedOldBlocksList()
}

func (c *pvtDataStore) updateLastCommittedBlockNum(block *cb.Block) error {
	blockNum := block.Header.Number

	logger.Debugf("[%s] Updating last committed block number to %d", c.ledgerID, blockNum)

	c.pvtDataDBStore.UpdateLastCommittedBlockNum(blockNum)
	c.cachePvtDataStore.UpdateLastCommittedBlockNum(blockNum)

	return nil
}
