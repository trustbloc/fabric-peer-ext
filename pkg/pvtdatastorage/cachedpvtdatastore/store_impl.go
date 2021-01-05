/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cachedpvtdatastore

import (
	"fmt"
	"sync/atomic"

	"github.com/bluele/gcache"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/pvtdatapolicy"
	"github.com/hyperledger/fabric/core/ledger/pvtdatastorage"
	xstorageapi "github.com/hyperledger/fabric/extensions/storage/api"
	"github.com/pkg/errors"

	"github.com/trustbloc/fabric-peer-ext/pkg/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/pvtdatastorage/common"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

var logger = flogging.MustGetLogger("cachedpvtdatastore")

// Provider creates in-memory private data stores
type Provider struct {
}

type store struct {
	ledgerid           string
	btlPolicy          pvtdatapolicy.BTLPolicy
	cache              gcache.Cache
	lastCommittedBlock uint64
	empty              uint32
}

type pendingPvtData struct {
	batchPending bool
	dataEntries  []*common.DataEntry
}

//////// Provider functions  /////////////
//////////////////////////////////////////

// NewProvider instantiates a private data storage Provider backed by cache
func NewProvider() *Provider {
	logger.Debugf("constructing cached private data storage Provider")
	return &Provider{}
}

// Create creates a new cache store
func (p *Provider) Create(ledgerID string, lastCommittedBlockNum uint64) xstorageapi.PrivateDataStore {
	s := &store{cache: gcache.New(config.GetPvtDataCacheSize()).ARC().Build(), ledgerid: ledgerID,
		lastCommittedBlock: lastCommittedBlockNum,
	}

	if lastCommittedBlockNum == 0 {
		s.empty = 1
	}

	logger.Debugf("Pvtdata cache store opened. Initial state: isEmpty [%t], lastCommittedBlock [%d]",
		s.isEmpty, lastCommittedBlockNum)

	return s
}

//////// store functions  ////////////////
//////////////////////////////////////////

func (s *store) Init(btlPolicy pvtdatapolicy.BTLPolicy) {
	s.btlPolicy = btlPolicy
}

// Prepare implements the function in the interface `Store`
func (s *store) Commit(blockNum uint64, pvtData []*ledger.TxPvtData, missingPvtData ledger.TxMissingPvtDataMap) error {
	if !roles.IsCommitter() {
		panic("calling Prepare on a peer that is not a committer")
	}

	expectedBlockNum := s.nextBlockNum()
	if expectedBlockNum != blockNum {
		return pvtdatastorage.NewErrIllegalCall(fmt.Sprintf("Expected block number=%d, received block number=%d", expectedBlockNum, blockNum))
	}

	storeEntries, err := common.PrepareStoreEntries(blockNum, pvtData, s.btlPolicy, missingPvtData)
	if err != nil {
		return err
	}

	pendingPvtData := &pendingPvtData{batchPending: true}
	if len(storeEntries.DataEntries) > 0 {
		pendingPvtData.dataEntries = storeEntries.DataEntries
	}
	logger.Debugf("Saved %d private data write sets for block [%d]", len(pvtData), blockNum)

	committingBlockNum := s.nextBlockNum()
	logger.Debugf("Committing private data for block [%d]", committingBlockNum)

	if pendingPvtData.dataEntries != nil {
		err := s.cache.Set(committingBlockNum, pendingPvtData.dataEntries)
		if err != nil {
			return errors.WithMessage(err, fmt.Sprintf("writing private data to cache failed [%d]", committingBlockNum))
		}
	}

	atomic.StoreUint32(&s.empty, 0)
	atomic.StoreUint64(&s.lastCommittedBlock, committingBlockNum)

	logger.Debugf("Committed private data for block [%d]", committingBlockNum)
	return nil
}

// CommitPvtDataOfOldBlocks implements the function in the interface `Store`
func (s *store) CommitPvtDataOfOldBlocks(blocksPvtData map[uint64][]*ledger.TxPvtData, _ ledger.MissingPvtDataInfo) error {
	return errors.New("not supported")
}

// GetLastUpdatedOldBlocksPvtData implements the function in the interface `Store`
func (s *store) GetLastUpdatedOldBlocksPvtData() (map[uint64][]*ledger.TxPvtData, error) {
	return nil, errors.New("not supported")
}

// ResetLastUpdatedOldBlocksList implements the function in the interface `Store`
func (s *store) ResetLastUpdatedOldBlocksList() error {
	return errors.New("not supported")
}

// GetPvtDataByBlockNum implements the function in the interface `Store`.
// If the store is empty or the last committed block number is smaller then the
// requested block number, an 'ErrOutOfRange' is thrown
func (s *store) GetPvtDataByBlockNum(blockNum uint64, filter ledger.PvtNsCollFilter) ([]*ledger.TxPvtData, error) {
	logger.Debugf("Get private data for block [%d], filter=%#v", blockNum, filter)
	if s.isEmpty() {
		return nil, pvtdatastorage.NewErrOutOfRange("The store is empty")
	}

	lastCommittedBlock := s.getLastCommittedBlock()

	if blockNum > lastCommittedBlock {
		return nil, pvtdatastorage.NewErrOutOfRange(fmt.Sprintf("Last committed block=%d, block requested=%d", lastCommittedBlock, blockNum))
	}

	value, err := s.cache.Get(blockNum)
	if err != nil {
		if err != gcache.KeyNotFoundError {
			panic(fmt.Sprintf("Get must never return an error other than KeyNotFoundError err:%s", err))
		}
		return nil, nil
	}

	dataEntries := value.([]*common.DataEntry)

	return s.getBlockPvtData(dataEntries, filter, blockNum)

}

// ProcessCollsEligibilityEnabled implements the function in the interface `Store`
func (s *store) ProcessCollsEligibilityEnabled(committingBlk uint64, nsCollMap map[string][]string) error {
	return errors.New("not supported")
}

//GetMissingPvtDataInfoForMostRecentBlocks implements the function in the interface `Store`
func (s *store) GetMissingPvtDataInfoForMostRecentBlocks(maxBlock int) (ledger.MissingPvtDataInfo, error) {
	return nil, errors.New("not supported")
}

// LastCommittedBlockHeight implements the function in the interface `Store`
func (s *store) LastCommittedBlockHeight() (uint64, error) {
	if s.isEmpty() {
		return 0, nil
	}
	return s.getLastCommittedBlock() + 1, nil
}

func v11RetrievePvtdata(dataEntries []*common.DataEntry, filter ledger.PvtNsCollFilter) ([]*ledger.TxPvtData, error) {
	var blkPvtData []*ledger.TxPvtData
	for _, dataEntry := range dataEntries {
		value, err := common.EncodeDataValue(dataEntry.Value)
		if err != nil {
			return nil, errors.Wrapf(err, "EncodeDataValue failed")
		}
		pvtDatum, err := common.V11DecodeKV(common.EncodeDataKey(dataEntry.Key), value, filter)
		if err != nil {
			return nil, err
		}
		blkPvtData = append(blkPvtData, pvtDatum)
	}
	return blkPvtData, nil
}

func (s *store) nextBlockNum() uint64 {
	if s.isEmpty() {
		return 0
	}
	return s.getLastCommittedBlock() + 1
}

func (s *store) getLastCommittedBlock() uint64 {
	return atomic.LoadUint64(&s.lastCommittedBlock)
}

func (s *store) isEmpty() bool {
	return atomic.LoadUint32(&s.empty) == 1
}

func (s *store) getBlockPvtData(dataEntries []*common.DataEntry, filter ledger.PvtNsCollFilter, blockNum uint64) ([]*ledger.TxPvtData, error) {
	var blockPvtdata []*ledger.TxPvtData
	var currentTxNum uint64
	var currentTxWsetAssember *common.TxPvtdataAssembler
	firstItr := true

	for _, dataEntry := range dataEntries {
		dataKeyBytes := common.EncodeDataKey(dataEntry.Key)
		ok, err := common.V11Format(dataKeyBytes)
		if err != nil {
			return nil, err
		}
		if ok {
			return v11RetrievePvtdata(dataEntries, filter)
		}
		expired, err := common.IsExpired(dataEntry.Key.NsCollBlk, s.btlPolicy, s.getLastCommittedBlock())
		if err != nil {
			return nil, err
		}
		if expired || !common.PassesFilter(dataEntry.Key, filter) {
			continue
		}

		if firstItr {
			currentTxNum = dataEntry.Key.TxNum
			currentTxWsetAssember = common.NewTxPvtdataAssembler(blockNum, currentTxNum)
			firstItr = false
		}

		if dataEntry.Key.TxNum != currentTxNum {
			blockPvtdata = append(blockPvtdata, currentTxWsetAssember.GetTxPvtdata())
			currentTxNum = dataEntry.Key.TxNum
			currentTxWsetAssember = common.NewTxPvtdataAssembler(blockNum, currentTxNum)
		}
		currentTxWsetAssember.Add(dataEntry.Key.Ns, dataEntry.Value)
	}
	if currentTxWsetAssember != nil {
		blockPvtdata = append(blockPvtdata, currentTxWsetAssember.GetTxPvtdata())
	}
	return blockPvtdata, nil
}
