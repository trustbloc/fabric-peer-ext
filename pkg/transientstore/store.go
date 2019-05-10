/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transientstore

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/bluele/gcache"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/transientstore"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	pb "github.com/hyperledger/fabric/protos/transientstore"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/transientstore/common"
)

var nilByte = byte('\x00')

// ErrStoreEmpty is used to indicate that there are no entries in transient store
var ErrStoreEmpty = errors.New("Transient store is empty")

// Store manages the storage of private write sets for a ledgerId.
// Ideally, a ledger can remove the data from this storage when it is committed to
// the permanent storage or the pruning of some data items is enforced by the policy
// the internal storage mechanism used in this specific store is gcache of type 'simple'
// cache to allow 'unlimited' growth of data and avoid eviction due to size. The assumption
// is the ledger will purge data from this storage once transactions are committed or deemed invalid.

type store struct {
	// cache contains a key of txid and a value represented as a map of composite key/TxRWSet values (real data store)
	cache gcache.Cache
	// blockHeightCache contains a key of blockHeight and value represented as a slice of txids
	blockHeightCache gcache.Cache
	// txidCache contains a key of txid and value represented as a slice of blockHeights
	txidCache gcache.Cache
}

func newStore() *store {
	s := &store{}
	s.cache = gcache.New(0).LoaderFunc(loadPvtRWSetMap).Build()
	s.blockHeightCache = gcache.New(0).LoaderFunc(loadBlockHeight).Build()
	s.txidCache = gcache.New(0).LoaderFunc(loadTxid).Build()
	return s
}

// Persist stores the private write set of a transaction in the transient store
// based on txid and the block height the private data was received at
func (s *store) Persist(txid string, blockHeight uint64, privateSimulationResults *rwset.TxPvtReadWriteSet) error {
	logger.Debugf("Persisting private data to transient store for txid [%s] at block height [%d]", txid, blockHeight)

	uuid := util.GenerateUUID()
	compositeKeyPvtRWSet := common.CreateCompositeKeyForPvtRWSet(txid, uuid, blockHeight)
	privateSimulationResultsBytes, err := proto.Marshal(privateSimulationResults)
	if err != nil {
		return err
	}

	s.setTxPvtWRSetToCache(txid, compositeKeyPvtRWSet, privateSimulationResultsBytes)

	s.setTxidToBlockHeightCache(txid, blockHeight)

	s.updateTxidCache(txid, blockHeight)
	return nil
}

// PersistWithConfig stores the private write set of a transaction along with the collection config
// in the transient store based on txid and the block height the private data was received at
func (s *store) PersistWithConfig(txid string, blockHeight uint64, privateSimulationResultsWithConfig *pb.TxPvtReadWriteSetWithConfigInfo) error {
	if privateSimulationResultsWithConfig != nil {
		logger.Debugf("Persisting private data to transient store for txid [%s] at block height [%d] with [%d] config(s)", txid, blockHeight, len(privateSimulationResultsWithConfig.CollectionConfigs))
	} else {
		logger.Debugf("Persisting private data to transient store for txid [%s] at block height [%d] with nil config", txid, blockHeight)
	}

	uuid := util.GenerateUUID()
	compositeKeyPvtRWSet := common.CreateCompositeKeyForPvtRWSet(txid, uuid, blockHeight)
	privateSimulationResultsWithConfigBytes, err := proto.Marshal(privateSimulationResultsWithConfig)
	if err != nil {
		return err
	}

	// emulating original Fabric's new proto (post v1.2) by appending nilByte
	// TODO remove this when Fabric stops appending nilByte
	privateSimulationResultsWithConfigBytes = append([]byte{nilByte}, privateSimulationResultsWithConfigBytes...)

	s.setTxPvtWRSetToCache(txid, compositeKeyPvtRWSet, privateSimulationResultsWithConfigBytes)

	s.setTxidToBlockHeightCache(txid, blockHeight)

	s.updateTxidCache(txid, blockHeight)
	return nil
}

func (s *store) updateTxidCache(txid string, blockHeight uint64) {
	value, err := s.txidCache.Get(txid)
	if err != nil {
		if err != gcache.KeyNotFoundError {
			panic(fmt.Sprintf("Get from cache must never return an error other than KeyNotFoundError err:%s", err))
		}
	}
	uintVal := value.(*blockHeightsSlice)
	if found, _ := uintVal.findBlockHeightEntryInSlice(blockHeight); !found {
		uintVal.add(blockHeight)
	}

	err = s.txidCache.Set(txid, uintVal)
	if err != nil {
		panic(fmt.Sprintf("Storing blockheight '%d' for txid key '%s' in transientstore cache must never fail, err:%s", blockHeight, txid, err))
	}
}

func (s *store) getTxPvtRWSetFromCache(txid string) *pvtRWSetMap {
	value, err := s.cache.Get(txid)
	if err != nil {
		if err != gcache.KeyNotFoundError {
			panic(fmt.Sprintf("Get from cache must never return an error other than KeyNotFoundError err:%s", err))
		}
	}

	return value.(*pvtRWSetMap)
}

func (s *store) getTxidsFromBlockHeightCache(blockHeight uint64) *txidsSlice {
	blockHeightValue, err := s.blockHeightCache.Get(blockHeight)
	if err != nil {
		if err != gcache.KeyNotFoundError {
			panic(fmt.Sprintf("Get from cache must never return an error other than KeyNotFoundError err:%s", err))
		}
	}
	return blockHeightValue.(*txidsSlice)
}

func (s *store) setTxPvtWRSetToCache(txid string, compositeKeyPvtRWSet, privSimulationResults []byte) {
	txPvtRWSetMap := s.getTxPvtRWSetFromCache(txid)
	k := hex.EncodeToString(compositeKeyPvtRWSet)
	v := base64.StdEncoding.EncodeToString(privSimulationResults)
	txPvtRWSetMap.set(k, v)

	err := s.cache.Set(txid, txPvtRWSetMap)
	if err != nil {
		panic(fmt.Sprintf("Set to cache must never return an error, got error:%s", err))
	}
}

func (s *store) setTxidToBlockHeightCache(txid string, blockHeight uint64) {
	blockHeightTxids := s.getTxidsFromBlockHeightCache(blockHeight)
	found, _ := blockHeightTxids.findTxidEntryInSlice(txid)
	if !found {
		blockHeightTxids.add(txid)
		err := s.blockHeightCache.Set(blockHeight, blockHeightTxids)
		if err != nil {
			panic(fmt.Sprintf("Set to cache must never return an error, got error:%s", err))
		}
	}
}

// GetTxPvtRWSetByTxid returns an iterator due to the fact that the txid may have multiple private
// write sets persisted from different endorsers (via Gossip)
func (s *store) GetTxPvtRWSetByTxid(txid string, filter ledger.PvtNsCollFilter) (transientstore.RWSetScanner, error) {
	logger.Debugf("Calling GetTxPvtRWSetByTxid on transient store for txid [%s]", txid)
	var results []keyValue

	val, err := s.cache.Get(txid)
	if err != nil {
		if err != gcache.KeyNotFoundError {
			panic(fmt.Sprintf("Get from cache must never return an error other than KeyNotFoundError err:%s", err))
		}
		// return empty results
		return &RwsetScanner{filter: filter, results: []keyValue{}}, nil
	}

	pvtRWsm := val.(*pvtRWSetMap)
	pvtRWsm.mu.RLock()
	defer pvtRWsm.mu.RUnlock()
	for key, value := range pvtRWsm.m {
		results = append(results, keyValue{key: key, value: value})
	}

	return &RwsetScanner{filter: filter, results: results}, nil
}

// GetMinTransientBlkHt returns the lowest block height remaining in transient store
func (s *store) GetMinTransientBlkHt() (uint64, error) {
	var minTransientBlkHt uint64
	val := s.blockHeightCache.GetALL()

	for key := range val {
		k := key.(uint64)
		if minTransientBlkHt == 0 || k < minTransientBlkHt {
			minTransientBlkHt = k
		}
	}
	logger.Debugf("Called GetMinTransientBlkHt on transient store, min block height is: %d", minTransientBlkHt)
	if minTransientBlkHt == 0 { // mimic Fabric's transientstore with leveldb -> return an error
		return 0, ErrStoreEmpty
	}
	return minTransientBlkHt, nil
}

// PurgeByTxids removes private write sets of a given set of transactions from the
// transient store
func (s *store) PurgeByTxids(txids []string) error {
	logger.Debugf("Calling PurgeByTxids on transient store for txids [%v]", txids)
	s.purgeTxPvtRWSetCacheByTxids(txids)
	return s.purgeBlockHeightCacheByTxids(txids)
}

// Shutdown noop for in memory storage
func (s *store) Shutdown() {

}

func (s *store) purgeTxPvtRWSetCacheByTxids(txids []string) {
	for _, txID := range txids {
		s.cache.Remove(txID)
	}
}

func (s *store) purgeTxRWSetCacheByBlockHeight(txids []string, maxBlockNumToRetain uint64) error {
	for _, txID := range txids {
		txMap := s.getTxPvtRWSetFromCache(txID)
		for _, txK := range txMap.keys() {
			hexKey, err := hex.DecodeString(txK)
			if err != nil {
				return err
			}
			_, blkHeight := common.SplitCompositeKeyOfPvtRWSet(hexKey)
			if blkHeight < maxBlockNumToRetain {
				txMap.delete(txK)
			}
		}
		if txMap.length() == 0 {
			s.cache.Remove(txID)
		}

	}
	return nil
}

func (s *store) purgeBlockHeightCacheByTxids(txids []string) error {
	sort.Strings(txids)
	var blkHeightTxids *txidsSlice
	var blkHeightKeys []uint64
	// step 1 fetch block heights for txids
	blkHeightKeys, err := s.getBlockHeightKeysFromTxidCache(txids)
	if err != nil {
		return err
	}
	// step 2 remove txids from blockHeightCache
	for _, blkHgtKey := range blkHeightKeys {
		value, err := s.blockHeightCache.Get(blkHgtKey)
		if err != nil {
			if err == gcache.KeyNotFoundError {
				continue
			}
			return err
		}
		blkHeightTxids = value.(*txidsSlice)
		for _, txID := range txids {
			for { // ensure to remove duplicates
				if isFound, i := blkHeightTxids.findTxidEntryInSlice(txID); isFound {
					blkHeightTxids.removeTxidEntryAtIndex(i)
				} else {
					break
				}
			}
		}

		if blkHeightTxids.length() == 0 {
			s.blockHeightCache.Remove(blkHgtKey)
		} else {
			err := s.blockHeightCache.Set(blkHgtKey, blkHeightTxids)
			if err != nil {
				return err
			}
		}
	}

	// step 3 remove blockHeights from txidCache
	return s.purgeTxidsCacheByBlockHeight(txids, blkHeightKeys)
}

func (s *store) getBlockHeightKeysFromTxidCache(txids []string) ([]uint64, error) {
	var blkHeightKeys []uint64
	for _, t := range txids {
		blkHgts, err := s.txidCache.Get(t)
		if err != nil {
			if err == gcache.KeyNotFoundError {
				continue
			}
			return nil, err
		}
		if blkHgts.(*blockHeightsSlice).length() > 0 {
			blkHeightKeys = append(blkHeightKeys, blkHgts.(*blockHeightsSlice).getBlockHeights()...)
		}
	}
	blkHeightKeys = sliceUniqueUint64(blkHeightKeys)
	return blkHeightKeys, nil
}

// PurgeByHeight will remove all ReadWriteSets with block height below maxBlockNumToRetain
func (s *store) PurgeByHeight(maxBlockNumToRetain uint64) error {
	logger.Debugf("Calling PurgeByHeight on transient store for maxBlockNumToRetain [%d]", maxBlockNumToRetain)
	txIDs := make([]string, 0)
	blkHgts := make([]uint64, 0)
	for key, value := range s.blockHeightCache.GetALL() {
		k := key.(uint64)
		if k < maxBlockNumToRetain {
			txIDs = append(txIDs, value.(*txidsSlice).getTxids()...)
			blkHgts = append(blkHgts, k)
			s.blockHeightCache.Remove(k)
		}
	}
	txIDs = sliceUniqueString(txIDs)
	blkHgts = sliceUniqueUint64(blkHgts)
	err := s.purgeTxRWSetCacheByBlockHeight(txIDs, maxBlockNumToRetain)
	if err != nil {
		return err
	}
	return s.purgeTxidsCacheByBlockHeight(txIDs, blkHgts)

}

func (s *store) purgeTxidsCacheByBlockHeight(txids []string, blockHeights []uint64) error {
	for _, txid := range txids {
		blkHgt, err := s.txidCache.Get(txid)
		if err != nil {
			if err == gcache.KeyNotFoundError {
				continue
			}
			return err
		}
		blkHgtSliceByTxid := blkHgt.(*blockHeightsSlice)

		for _, b := range blockHeights {
			if isFound, i := blkHgtSliceByTxid.findBlockHeightEntryInSlice(b); isFound {
				blkHgtSliceByTxid.removeBlockHeightEntryAtIndex(i)
			}
		}

		if blkHgtSliceByTxid.length() == 0 {
			s.txidCache.Remove(txid)
		}
	}
	return nil
}

type keyValue struct {
	key   string
	value string
}

// RwsetScanner provides an iterator for EndorserPvtSimulationResults from transientstore
type RwsetScanner struct {
	filter  ledger.PvtNsCollFilter
	results []keyValue
	next    int
}

// Next moves the iterator to the next key/value pair.
// It returns whether the iterator is exhausted.
// TODO: Once the related gossip changes are made as per FAB-5096, remove this function
func (scanner *RwsetScanner) Next() (*transientstore.EndorserPvtSimulationResults, error) {
	kv, ok := scanner.nextKV()
	if !ok {
		return nil, nil
	}

	keyBytes, err := hex.DecodeString(kv.key)
	if err != nil {
		return nil, err
	}
	_, blockHeight := common.SplitCompositeKeyOfPvtRWSet(keyBytes)
	logger.Debugf("scanner next blockHeight %d", blockHeight)
	txPvtRWSet := &rwset.TxPvtReadWriteSet{}
	valueBytes, err := base64.StdEncoding.DecodeString(kv.value)
	if err != nil {
		return nil, errors.Wrapf(err, "error from DecodeString for transientDataField")
	}

	if err := proto.Unmarshal(valueBytes, txPvtRWSet); err != nil {
		return nil, err
	}
	filteredTxPvtRWSet := common.TrimPvtWSet(txPvtRWSet, scanner.filter)
	logger.Debugf("scanner next filteredTxPvtRWSet %v", filteredTxPvtRWSet)

	return &transientstore.EndorserPvtSimulationResults{
		ReceivedAtBlockHeight: blockHeight,
		PvtSimulationResults:  filteredTxPvtRWSet,
	}, nil

}

// NextWithConfig moves the iterator to the next key/value pair with configs.
// It returns whether the iterator is exhausted.
// TODO: Once the related gossip changes are made as per FAB-5096, rename this function to Next
func (scanner *RwsetScanner) NextWithConfig() (*transientstore.EndorserPvtSimulationResultsWithConfig, error) {
	kv, ok := scanner.nextKV()
	if !ok {
		return nil, nil
	}

	keyBytes, err := hex.DecodeString(kv.key)
	if err != nil {
		return nil, err
	}
	_, blockHeight := common.SplitCompositeKeyOfPvtRWSet(keyBytes)
	logger.Debugf("scanner NextWithConfig blockHeight %d", blockHeight)

	valueBytes, err := base64.StdEncoding.DecodeString(kv.value)
	if err != nil {
		return nil, errors.Wrapf(err, "error from DecodeString for transientDataField")
	}

	txPvtRWSet := &rwset.TxPvtReadWriteSet{}
	var filteredTxPvtRWSet *rwset.TxPvtReadWriteSet
	txPvtRWSetWithConfig := &pb.TxPvtReadWriteSetWithConfigInfo{}

	if valueBytes[0] == nilByte {
		// new proto, i.e., TxPvtReadWriteSetWithConfigInfo
		if er := proto.Unmarshal(valueBytes[1:], txPvtRWSetWithConfig); er != nil {
			return nil, er
		}

		logger.Debugf("scanner NextWithConfig txPvtRWSetWithConfig %v", txPvtRWSetWithConfig)

		filteredTxPvtRWSet = common.TrimPvtWSet(txPvtRWSetWithConfig.GetPvtRwset(), scanner.filter)
		logger.Debugf("scanner NextWithConfig filteredTxPvtRWSet %v", filteredTxPvtRWSet)
		configs, err := common.TrimPvtCollectionConfigs(txPvtRWSetWithConfig.CollectionConfigs, scanner.filter)
		if err != nil {
			return nil, err
		}
		logger.Debugf("scanner NextWithConfig configs %v", configs)
		txPvtRWSetWithConfig.CollectionConfigs = configs
	} else {
		// old proto, i.e., TxPvtReadWriteSet
		if e := proto.Unmarshal(valueBytes, txPvtRWSet); e != nil {
			return nil, e
		}
		filteredTxPvtRWSet = common.TrimPvtWSet(txPvtRWSet, scanner.filter)
	}

	txPvtRWSetWithConfig.PvtRwset = filteredTxPvtRWSet

	return &transientstore.EndorserPvtSimulationResultsWithConfig{
		ReceivedAtBlockHeight:          blockHeight,
		PvtSimulationResultsWithConfig: txPvtRWSetWithConfig,
	}, nil
}

func (scanner *RwsetScanner) nextKV() (keyValue, bool) {
	i := scanner.next
	if i >= len(scanner.results) {
		return keyValue{}, false
	}
	scanner.next++
	return scanner.results[i], true
}

// Close releases resource held by the iterator
func (scanner *RwsetScanner) Close() {
}
