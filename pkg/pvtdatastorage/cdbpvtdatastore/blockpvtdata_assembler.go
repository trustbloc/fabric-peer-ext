/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pvtdatastorage

import (
	"encoding/hex"

	"github.com/hyperledger/fabric/core/ledger"
	"github.com/trustbloc/fabric-peer-ext/pkg/pvtdatastorage/common"
)

type blockPvtDataAssembler struct {
	blockPvtdata          []*ledger.TxPvtData
	currentTxNum          uint64
	currentTxWsetAssember *common.TxPvtdataAssembler
	firstItr              bool

	results            map[string][]byte
	filter             ledger.PvtNsCollFilter
	blockNum           uint64
	sortedKeys         []string
	lastCommittedBlock uint64
	checkIsExpired     func(dataKey *common.DataKey, filter ledger.PvtNsCollFilter, lastCommittedBlock uint64) (bool, error)
}

func newBlockPvtDataAssembler(
	results map[string][]byte,
	filter ledger.PvtNsCollFilter,
	blockNum, lastCommittedBlock uint64,
	sortedKeys []string,
	checkIsExpired func(dataKey *common.DataKey, filter ledger.PvtNsCollFilter, lastCommittedBlock uint64) (bool, error),
) *blockPvtDataAssembler {
	return &blockPvtDataAssembler{
		firstItr:           true,
		results:            results,
		filter:             filter,
		blockNum:           blockNum,
		lastCommittedBlock: lastCommittedBlock,
		sortedKeys:         sortedKeys,
		checkIsExpired:     checkIsExpired,
	}
}

func (a *blockPvtDataAssembler) assemble() ([]*ledger.TxPvtData, error) {
	for _, key := range a.sortedKeys {
		dataKeyBytes, err := hex.DecodeString(key)
		if err != nil {
			return nil, err
		}

		ok, err := v11Format(dataKeyBytes)
		if err != nil {
			return nil, err
		}

		if ok {
			return common.V11RetrievePvtdata(a.results, a.filter)
		}

		err = a.assembleKeyValue(dataKeyBytes, a.results[key])
		if err != nil {
			return nil, err
		}
	}

	if a.currentTxWsetAssember != nil {
		a.blockPvtdata = append(a.blockPvtdata, a.currentTxWsetAssember.GetTxPvtdata())
	}

	return a.blockPvtdata, nil
}

func (a *blockPvtDataAssembler) assembleKeyValue(dataKeyBytes, dataValueBytes []byte) error {
	dataKey, err := common.DecodeDatakey(dataKeyBytes)
	if err != nil {
		return err
	}

	expired, err := a.checkIsExpired(dataKey, a.filter, a.lastCommittedBlock)
	if err != nil {
		return err
	}
	if expired {
		return nil
	}

	dataValue, err := common.DecodeDataValue(dataValueBytes)
	if err != nil {
		return err
	}

	if a.firstItr {
		a.currentTxNum = dataKey.TxNum
		a.currentTxWsetAssember = common.NewTxPvtdataAssembler(a.blockNum, a.currentTxNum)
		a.firstItr = false
	}

	if dataKey.TxNum != a.currentTxNum {
		a.blockPvtdata = append(a.blockPvtdata, a.currentTxWsetAssember.GetTxPvtdata())
		a.currentTxNum = dataKey.TxNum
		a.currentTxWsetAssember = common.NewTxPvtdataAssembler(a.blockNum, a.currentTxNum)
	}
	a.currentTxWsetAssember.Add(dataKey.Ns, dataValue)

	return nil
}

// v11Format may be overridden by unit tests
var v11Format = func(datakeyBytes []byte) (bool, error) {
	return common.V11Format(datakeyBytes)
}
