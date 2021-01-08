/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator

import (
	"fmt"
	"sync"

	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/peer"

	"github.com/trustbloc/fabric-peer-ext/pkg/common/txflags"
)

type txResults struct {
	channelID   string
	mutex       sync.RWMutex
	flags       txflags.ValidationFlags
	txIDs       []string
	blockNumber uint64
}

// newTxResults returns a new transaction results struct initialized with the given block
func newTxResults(channelID string, block *common.Block) *txResults {
	// Copy the current flags from the block
	flags := txflags.New(len(block.Data.Data))
	currentFlags := txflags.ValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
	for i := range block.Data.Data {
		flags.SetFlag(i, currentFlags.Flag(i))
	}

	return &txResults{
		channelID:   channelID,
		blockNumber: block.Header.Number,
		flags:       flags,
		txIDs:       make([]string, len(flags)),
	}
}

// Merge merges the given flags and transaction IDs and returns true if all of the flags have been validated
func (f *txResults) Merge(flags txflags.ValidationFlags, txIDs []string) (bool, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(flags) != len(f.flags) {
		return false, fmt.Errorf("the length of the provided flags %d does not match the length of the existing flags %d", len(flags), len(f.flags))
	}

	if len(txIDs) != len(f.txIDs) {
		return false, fmt.Errorf("the length of the provided Tx IDs %d does not match the length of the existing Tx IDs %d", len(txIDs), len(f.txIDs))
	}

	for i, flag := range flags {
		if peer.TxValidationCode(flag) == peer.TxValidationCode_NOT_VALIDATED {
			continue
		}

		currentFlag := f.flags.Flag(i)
		if currentFlag == peer.TxValidationCode_NOT_VALIDATED {
			txID := txIDs[i]
			code := peer.TxValidationCode(flag)

			traceLogger.Debugf("[%s] Setting result for Tx [%s] at index [%d] for block %d: %s", f.channelID, txID, i, f.blockNumber, code)

			f.flags.SetFlag(i, code)
			f.txIDs[i] = txID
		} else {
			traceLogger.Debugf("[%s] Not setting result for Tx [%s] at index [%d] for block number %d since it is already set to: %s", f.channelID, f.txIDs[i], i, f.blockNumber, currentFlag)
		}
	}

	f.markDuplicateTxIDs()

	return f.allValidated(), nil
}

// UnvalidatedMap returns a map of TX indexes of the transaction that are not yet validated
func (f *txResults) UnvalidatedMap() map[int]struct{} {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	notValidated := make(map[int]struct{})
	for i, flag := range f.flags {
		if peer.TxValidationCode(flag) == peer.TxValidationCode_NOT_VALIDATED {
			notValidated[i] = struct{}{}
		}
	}

	return notValidated
}

// AllValidated returns true if none of the flags is set to NOT_VALIDATED
func (f *txResults) AllValidated() bool {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	return f.allValidated()
}

// Flags returns the validation flags
func (f *txResults) Flags() txflags.ValidationFlags {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	return f.flags
}

// markDuplicateTxIDs checks for duplicate transaction IDs. If a duplicate is found then it
// is flagged with code TxValidationCode_DUPLICATE_TXID
func (f *txResults) markDuplicateTxIDs() {
	txIDMap := make(map[string]struct{})

	for i, txID := range f.txIDs {
		if txID == "" {
			continue
		}

		_, in := txIDMap[txID]
		if in {
			logger.Warningf("Duplicate txID found: %s", txID)

			f.flags.SetFlag(i, peer.TxValidationCode_DUPLICATE_TXID)
		} else {
			txIDMap[txID] = struct{}{}
		}
	}
}

// allValidated returns true if none of the flags is set to NOT_VALIDATED
func (f *txResults) allValidated() bool {
	for _, txStatus := range f.flags {
		if peer.TxValidationCode(txStatus) == peer.TxValidationCode_NOT_VALIDATED {
			return false
		}
	}

	return true
}
