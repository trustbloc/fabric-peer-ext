/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pvtdatastore

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/hyperledger/fabric/protos/transientstore"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("ext_pvtdatastore")

type transientStore interface {
	// PersistWithConfig stores the private write set of a transaction along with the collection config
	// in the transient store based on txid and the block height the private data was received at
	PersistWithConfig(txid string, blockHeight uint64, privateSimulationResultsWithConfig *transientstore.TxPvtReadWriteSetWithConfigInfo) error
}

// Store persists private data from collection R/W sets
type Store struct {
	channelID      string
	transientStore transientStore
	collDataStore  storeapi.Store
}

// New returns a new Store
func New(channelID string, transientStore transientStore, collDataStore storeapi.Store) *Store {
	return &Store{
		channelID:      channelID,
		transientStore: transientStore,
		collDataStore:  collDataStore,
	}
}

// StorePvtData used to persist private date into transient and/or extended collection store(s)
func (c *Store) StorePvtData(txID string, privData *transientstore.TxPvtReadWriteSetWithConfigInfo, blkHeight uint64) error {
	// Filter out all of the R/W sets that are not to be persisted to the ledger, i.e. we just
	// want the regular private data R/W sets
	pvtRWSet, err := newPvtRWSetFilter(c.channelID, txID, blkHeight, privData).apply()
	if err != nil {
		logger.Errorf("[%s:%d:%s] Unable to extract transient r/w set from private data: %s", c.channelID, blkHeight, txID, err)
		return err
	}

	// Persist the extended collection data (this includes Transient Data, etc.)
	err = c.collDataStore.Persist(txID, privData)
	if err != nil {
		logger.Errorf("[%s:%d:%s] Unable to persist collection data: %s", c.channelID, blkHeight, txID, err)
		return err
	}

	if pvtRWSet == nil {
		logger.Debugf("[%s:%d:%s] Nothing to persist to transient store", c.channelID, blkHeight, txID)
		return nil
	}

	logger.Debugf("[%s:%d:%s] Persisting private data to transient store", c.channelID, blkHeight, txID)

	return c.transientStore.PersistWithConfig(
		txID, blkHeight,
		&transientstore.TxPvtReadWriteSetWithConfigInfo{
			PvtRwset:          pvtRWSet,
			CollectionConfigs: privData.CollectionConfigs,
			EndorsedAt:        privData.EndorsedAt,
		})
}

// pvtRWSetFilter filters out all of the R/W sets that are not to be persisted to the ledger,
// i.e. we just want the regular private data R/W sets
type pvtRWSetFilter struct {
	channelID string
	txID      string
	blkHeight uint64
	privData  *transientstore.TxPvtReadWriteSetWithConfigInfo
}

func newPvtRWSetFilter(channelID, txID string, blkHeight uint64, privData *transientstore.TxPvtReadWriteSetWithConfigInfo) *pvtRWSetFilter {
	return &pvtRWSetFilter{
		channelID: channelID,
		txID:      txID,
		blkHeight: blkHeight,
		privData:  privData,
	}
}

func (f *pvtRWSetFilter) apply() (*rwset.TxPvtReadWriteSet, error) {
	txPvtRWSet, err := rwsetutil.TxPvtRwSetFromProtoMsg(f.privData.PvtRwset)
	if err != nil {
		return nil, errors.New("error getting pvt RW set from bytes")
	}

	nsPvtRwSet, modified, err := f.extractPvtRWSets(txPvtRWSet.NsPvtRwSet)
	if err != nil {
		return nil, err
	}
	if !modified {
		logger.Debugf("[%s:%d:%s] Rewrite to NsPvtRwSet not required for transient store", f.channelID, f.blkHeight, f.txID)
		return f.privData.PvtRwset, nil
	}
	if len(nsPvtRwSet) == 0 {
		logger.Debugf("[%s:%d:%s] Didn't find any private data to persist to transient store", f.channelID, f.blkHeight, f.txID)
		return nil, nil
	}

	logger.Debugf("[%s:%d:%s] Rewriting NsPvtRwSet for transient store", f.channelID, f.blkHeight, f.txID)
	txPvtRWSet.NsPvtRwSet = nsPvtRwSet
	newPvtRwset, err := txPvtRWSet.ToProtoMsg()
	if err != nil {
		return nil, errors.WithMessage(err, "error marshalling private data r/w set")
	}

	logger.Debugf("[%s:%d:%s] Rewriting private data r/w set since it was modified", f.channelID, f.blkHeight, f.txID)
	return newPvtRwset, nil
}

func (f *pvtRWSetFilter) extractPvtRWSets(srcNsPvtRwSets []*rwsetutil.NsPvtRwSet) ([]*rwsetutil.NsPvtRwSet, bool, error) {
	modified := false
	var nsPvtRWSets []*rwsetutil.NsPvtRwSet
	for _, nsRWSet := range srcNsPvtRwSets {
		collPvtRwSets, err := f.extractPvtCollPvtRWSets(nsRWSet)
		if err != nil {
			return nil, false, err
		}
		if len(collPvtRwSets) != len(nsRWSet.CollPvtRwSets) {
			modified = true
		}
		if len(collPvtRwSets) > 0 {
			if len(collPvtRwSets) != len(nsRWSet.CollPvtRwSets) {
				logger.Debugf("[%s:%d:%s] Rewriting collections for [%s] in transient store", f.channelID, f.blkHeight, f.txID, nsRWSet.NameSpace)
				nsRWSet.CollPvtRwSets = collPvtRwSets
				modified = true
			} else {
				logger.Debugf("[%s:%d:%s] Not touching collections for [%s] in transient store", f.channelID, f.blkHeight, f.txID, nsRWSet.NameSpace)
			}
			logger.Debugf("[%s:%d:%s] Adding NsPvtRwSet for [%s] in transient store", f.channelID, f.blkHeight, f.txID, nsRWSet.NameSpace)
			nsPvtRWSets = append(nsPvtRWSets, nsRWSet)
		} else {
			logger.Debugf("[%s:%d:%s] NOT adding NsPvtRwSet for [%s] in transient store since no private data collections found", f.channelID, f.blkHeight, f.txID, nsRWSet.NameSpace)
		}
	}
	return nsPvtRWSets, modified, nil
}

func (f *pvtRWSetFilter) extractPvtCollPvtRWSets(nsPvtRWSet *rwsetutil.NsPvtRwSet) ([]*rwsetutil.CollPvtRwSet, error) {
	var filteredCollPvtRwSets []*rwsetutil.CollPvtRwSet
	for _, collRWSet := range nsPvtRWSet.CollPvtRwSets {
		ok, e := f.isPvtData(nsPvtRWSet.NameSpace, collRWSet.CollectionName)
		if e != nil {
			return nil, errors.WithMessage(e, "error in collection config")
		}
		if !ok {
			logger.Debugf("[%s:%d:%s] Not persisting collection [%s:%s] in transient store", f.channelID, f.blkHeight, f.txID, nsPvtRWSet.NameSpace, collRWSet.CollectionName)
			continue
		}
		logger.Debugf("[%s:%d:%s] Persisting collection [%s:%s] in transient store", f.channelID, f.blkHeight, f.txID, nsPvtRWSet.NameSpace, collRWSet.CollectionName)
		filteredCollPvtRwSets = append(filteredCollPvtRwSets, collRWSet)
	}
	return filteredCollPvtRwSets, nil
}

// isPvtData returns true if the given collection is a standard private data collection
func (f *pvtRWSetFilter) isPvtData(ns, coll string) (bool, error) {
	pkg, ok := f.privData.CollectionConfigs[ns]
	if !ok {
		return false, errors.Errorf("could not find collection configs for namespace [%s]", ns)
	}

	var config *common.StaticCollectionConfig
	for _, c := range pkg.Config {
		staticConfig := c.GetStaticCollectionConfig()
		if staticConfig.Name == coll {
			config = staticConfig
			break
		}
	}

	if config == nil {
		return false, errors.Errorf("could not find collection config for collection [%s:%s]", ns, coll)
	}

	return config.Type == common.CollectionType_COL_UNKNOWN || config.Type == common.CollectionType_COL_PRIVATE, nil
}
