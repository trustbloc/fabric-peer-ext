/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pvtdatastorage

import (
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/pvtdatapolicy"
	"github.com/hyperledger/fabric/core/ledger/pvtdatastorage"
	"github.com/trustbloc/fabric-peer-ext/pkg/pvtdatastorage/cachedpvtdatastore"
	cdbpvtdatastore "github.com/trustbloc/fabric-peer-ext/pkg/pvtdatastorage/cdbpvtdatastore"
)

//////// Provider functions  /////////////
//////////////////////////////////////////

// PvtDataProvider encapsulates the storage and cache providers in addition to the missing data index provider
type PvtDataProvider struct {
	storageProvider pvtdatastorage.Provider
	cacheProvider   pvtdatastorage.Provider
}

// NewProvider creates a new PvtDataStoreProvider that combines a cache provider and a backing storage provider
func NewProvider(conf *ledger.PrivateData, ledgerconfig *ledger.Config) *PvtDataProvider {
	// create couchdb pvt date store provider
	storageProvider, err := cdbpvtdatastore.NewProvider(conf, ledgerconfig)
	if err != nil {
		panic(err)
	}
	// create cache pvt date store provider
	cacheProvider := cachedpvtdatastore.NewProvider()

	p := PvtDataProvider{
		storageProvider: storageProvider,
		cacheProvider:   cacheProvider,
	}
	return &p
}

// OpenStore creates a pvt data store instance for the given ledger ID
func (c *PvtDataProvider) OpenStore(ledgerID string) (pvtdatastorage.Store, error) {
	pvtDataStore, err := c.storageProvider.OpenStore(ledgerID)
	if err != nil {
		return nil, err
	}
	cachePvtDataStore, err := c.cacheProvider.OpenStore(ledgerID)
	if err != nil {
		return nil, err
	}

	return newPvtDataStore(pvtDataStore, cachePvtDataStore)
}

// Close cleans up the Provider
func (c *PvtDataProvider) Close() {
	c.storageProvider.Close()
	c.cacheProvider.Close()

}

type pvtDataStore struct {
	pvtDataDBStore    pvtdatastorage.Store
	cachePvtDataStore pvtdatastorage.Store
}

func newPvtDataStore(pvtDataDBStore pvtdatastorage.Store, cachePvtDataStore pvtdatastorage.Store) (*pvtDataStore, error) {
	isEmpty, err := pvtDataDBStore.IsEmpty()
	if err != nil {
		return nil, err
	}
	// InitLastCommittedBlock for cache if pvtdata storage not empty
	if !isEmpty {
		lastCommittedBlockHeight, err := pvtDataDBStore.LastCommittedBlockHeight()
		if err != nil {
			return nil, err
		}
		err = cachePvtDataStore.InitLastCommittedBlock(lastCommittedBlockHeight - 1)
		if err != nil {
			return nil, err
		}
	}
	c := pvtDataStore{
		pvtDataDBStore:    pvtDataDBStore,
		cachePvtDataStore: cachePvtDataStore,
	}
	return &c, nil
}

//////// store functions  ////////////////
//////////////////////////////////////////
func (c *pvtDataStore) Init(btlPolicy pvtdatapolicy.BTLPolicy) {
	c.cachePvtDataStore.Init(btlPolicy)
	c.pvtDataDBStore.Init(btlPolicy)
}

// Prepare pvt data in cache and send pvt data to background prepare/commit go routine
func (c *pvtDataStore) Prepare(blockNum uint64, pvtData []*ledger.TxPvtData, pvtMissingDataMap ledger.TxMissingPvtDataMap) error {
	// Prepare data in cache
	err := c.cachePvtDataStore.Prepare(blockNum, pvtData, pvtMissingDataMap)
	if err != nil {
		return err
	}
	// Prepare data in storage
	return c.pvtDataDBStore.Prepare(blockNum, pvtData, pvtMissingDataMap)
}

// Commit pvt data in cache and call background pvtDataWriter go routine to commit data
func (c *pvtDataStore) Commit() error {
	// Commit data in cache
	err := c.cachePvtDataStore.Commit()
	if err != nil {
		return err
	}
	// Commit data in storage
	return c.pvtDataDBStore.Commit()
}

//InitLastCommittedBlock initialize last committed block
func (c *pvtDataStore) InitLastCommittedBlock(blockNum uint64) error {
	// InitLastCommittedBlock data in cache
	err := c.cachePvtDataStore.InitLastCommittedBlock(blockNum)
	if err != nil {
		return err
	}
	// InitLastCommittedBlock data in storage
	return c.pvtDataDBStore.InitLastCommittedBlock(blockNum)
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

//HasPendingBatch implements the function in the interface `Store`
func (c *pvtDataStore) HasPendingBatch() (bool, error) {
	return c.pvtDataDBStore.HasPendingBatch()
}

//LastCommittedBlockHeight implements the function in the interface `Store`
func (c *pvtDataStore) LastCommittedBlockHeight() (uint64, error) {
	return c.pvtDataDBStore.LastCommittedBlockHeight()
}

//IsEmpty implements the function in the interface `Store`
func (c *pvtDataStore) IsEmpty() (bool, error) {
	return c.pvtDataDBStore.IsEmpty()
}

// Rollback pvt data in cache and call background pvtDataWriter go routine to rollback data
func (c *pvtDataStore) Rollback() error {
	// Rollback data in cache
	err := c.cachePvtDataStore.Rollback()
	if err != nil {
		return err
	}
	// Rollback data in storage
	return c.pvtDataDBStore.Rollback()
}

//Shutdown implements the function in the interface `Store`
func (c *pvtDataStore) Shutdown() {
	c.cachePvtDataStore.Shutdown()
	c.pvtDataDBStore.Shutdown()
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
func (c *pvtDataStore) CommitPvtDataOfOldBlocks(blocksPvtData map[uint64][]*ledger.TxPvtData) error {
	return c.pvtDataDBStore.CommitPvtDataOfOldBlocks(blocksPvtData)
}

//GetLastUpdatedOldBlocksPvtData implements the function in the interface `Store`
func (c *pvtDataStore) GetLastUpdatedOldBlocksPvtData() (map[uint64][]*ledger.TxPvtData, error) {
	return c.pvtDataDBStore.GetLastUpdatedOldBlocksPvtData()
}

//ResetLastUpdatedOldBlocksList implements the function in the interface `Store`
func (c *pvtDataStore) ResetLastUpdatedOldBlocksList() error {
	return c.pvtDataDBStore.ResetLastUpdatedOldBlocksList()
}
