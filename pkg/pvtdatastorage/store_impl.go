/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pvtdatastorage

import (
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/pvtdatapolicy"
	"github.com/hyperledger/fabric/core/ledger/pvtdatastorage"

	xstorageapi "github.com/hyperledger/fabric/extensions/storage/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/pvtdatastorage/cachedpvtdatastore"
	cdbpvtdatastore "github.com/trustbloc/fabric-peer-ext/pkg/pvtdatastorage/cdbpvtdatastore"
)

type cacheProvider interface {
	Create(ledgerID string, lastCommittedBlockNum uint64) xstorageapi.PrivateDataStore
}

//////// Provider functions  /////////////
//////////////////////////////////////////

// PvtDataProvider encapsulates the storage and cache providers in addition to the missing data index provider
type PvtDataProvider struct {
	storageProvider xstorageapi.PrivateDataProvider
	cacheProvider   cacheProvider
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

	return newPvtDataStore(pvtDataStore, cachePvtDataStore), nil
}

// Close cleans up the Provider
func (c *PvtDataProvider) Close() {
	c.storageProvider.Close()
}

type pvtDataStore struct {
	pvtDataDBStore    xstorageapi.PrivateDataStore
	cachePvtDataStore xstorageapi.PrivateDataStore
}

func newPvtDataStore(pvtDataDBStore xstorageapi.PrivateDataStore, cachePvtDataStore xstorageapi.PrivateDataStore) *pvtDataStore {
	return &pvtDataStore{
		pvtDataDBStore:    pvtDataDBStore,
		cachePvtDataStore: cachePvtDataStore,
	}
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
