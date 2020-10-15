/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pvtdatastorage

import (
	"github.com/hyperledger/fabric/common/ledger/util/leveldbhelper"
	"github.com/hyperledger/fabric/core/ledger"
	couchdb "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"
	"github.com/hyperledger/fabric/core/ledger/pvtdatastorage"
	"github.com/pkg/errors"

	"github.com/trustbloc/fabric-peer-ext/pkg/pvtdatastorage/common"
)

// CommitPvtDataOfOldBlocks commits the pvtData (i.e., previously missing data) of old blocks.
// The parameter `blocksPvtData` refers a list of old block's pvtdata which are missing in the pvtstore.
// Given a list of old block's pvtData, `CommitPvtDataOfOldBlocks` performs the following three
// operations
// (1) construct update entries (i.e., dataEntries, expiryEntries, missingDataEntries)
//     from the above created data entries
// (2) create a db update batch from the update entries
// (3) commit the update batch to the pvtStore
func (s *store) CommitPvtDataOfOldBlocks(blocksPvtData map[uint64][]*ledger.TxPvtData, unreconciledMissingData ledger.MissingPvtDataInfo) error {
	s.purgerLock.Lock()
	defer s.purgerLock.Unlock()

	if s.isLastUpdatedOldBlocksSet {
		return pvtdatastorage.NewErrIllegalCall(`The lastUpdatedOldBlocksList is set. It means that the
		stateDB may not be in sync with the pvtStore`)
	}

	// create a list of blocks' pvtData which are being stored. If this list is
	// found during the recovery, the stateDB may not be in sync with the pvtData
	// and needs recovery. In a normal flow, once the stateDB is synced, the
	// block list would be deleted.
	updatedBlksListMap := make(map[uint64]bool)

	for blockNum := range blocksPvtData {
		updatedBlksListMap[blockNum] = true
	}

	docs, batch, err := s.constructDBUpdateBatch(blocksPvtData, unreconciledMissingData)
	if err != nil {
		return err
	}

	if err := s.addLastUpdatedOldBlocksList(batch, updatedBlksListMap); err != nil {
		return err
	}

	if len(docs) > 0 {
		if _, err := s.db.BatchUpdateDocuments(docs); err != nil {
			return errors.Wrapf(err, "BatchUpdateDocuments failed")
		}
	}

	if err := s.missingKeysIndexDB.WriteBatch(batch, true); err != nil {
		return errors.Wrapf(err, "WriteBatch failed")
	}

	return nil
}

func (s *store) constructDBUpdateBatch(blocksPvtData map[uint64][]*ledger.TxPvtData, unreconciledMissingData ledger.MissingPvtDataInfo) ([]*couchdb.CouchDoc, *leveldbhelper.UpdateBatch, error) {
	p := common.NewOldBlockDataProcessor(s.btlPolicy, s.getExpiryDataOfExpiryKey, s.preparePvtDataDoc, s.missingKeysIndexDB)

	if err := p.PrepareDataAndExpiryEntries(blocksPvtData); err != nil {
		return nil, nil, err
	}

	if err := p.PrepareMissingDataEntriesToReflectReconciledData(); err != nil {
		return nil, nil, err
	}

	if err := p.PrepareMissingDataEntriesToReflectPriority(unreconciledMissingData); err != nil {
		return nil, nil, err
	}

	return p.ConstructDBUpdateBatch()
}
