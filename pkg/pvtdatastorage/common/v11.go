/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/ledger/rwset"
	"github.com/hyperledger/fabric/core/ledger"

	"github.com/trustbloc/fabric-peer-ext/pkg/common/version"
)

// todo add pinning script to include copied code into this file, original file from fabric is found in fabric/core/ledger/pvtdatastorage/v11.go
// todo below functions are originally unexported, the pinning script must capitalize these functions to export them

type blkTranNumKey []byte

func V11Format(datakeyBytes []byte) (bool, error) {
	_, n, err := version.NewHeightFromBytes(datakeyBytes[1:])
	if err != nil {
		return false, err
	}
	remainingBytes := datakeyBytes[n+1:]
	return len(remainingBytes) == 0, nil
}

func v11DecodePK(key blkTranNumKey) (blockNum uint64, tranNum uint64, err error) {
	height, _, err := version.NewHeightFromBytes(key[1:])
	if err != nil {
		return 0, 0, err
	}
	return height.BlockNum, height.TxNum, nil
}

func v11DecodePvtRwSet(encodedBytes []byte) (*rwset.TxPvtReadWriteSet, error) {
	writeset := &rwset.TxPvtReadWriteSet{}
	return writeset, proto.Unmarshal(encodedBytes, writeset)
}

func V11DecodeKV(k, v []byte, filter ledger.PvtNsCollFilter) (*ledger.TxPvtData, error) {
	_, tNum, err := v11DecodePK(k)
	if err != nil {
		return nil, err
	}
	var pvtWSet *rwset.TxPvtReadWriteSet
	if pvtWSet, err = v11DecodePvtRwSet(v); err != nil {
		return nil, err
	}
	filteredWSet := v11TrimPvtWSet(pvtWSet, filter)
	return &ledger.TxPvtData{SeqInBlock: tNum, WriteSet: filteredWSet}, nil
}

func v11TrimPvtWSet(pvtWSet *rwset.TxPvtReadWriteSet, filter ledger.PvtNsCollFilter) *rwset.TxPvtReadWriteSet {
	if filter == nil {
		return pvtWSet
	}

	var filteredNsRwSet []*rwset.NsPvtReadWriteSet
	for _, ns := range pvtWSet.NsPvtRwset {
		var filteredCollRwSet []*rwset.CollectionPvtReadWriteSet
		for _, coll := range ns.CollectionPvtRwset {
			if filter.Has(ns.Namespace, coll.CollectionName) {
				filteredCollRwSet = append(filteredCollRwSet, coll)
			}
		}
		if filteredCollRwSet != nil {
			filteredNsRwSet = append(filteredNsRwSet,
				&rwset.NsPvtReadWriteSet{
					Namespace:          ns.Namespace,
					CollectionPvtRwset: filteredCollRwSet,
				},
			)
		}
	}
	var filteredTxPvtRwSet *rwset.TxPvtReadWriteSet
	if filteredNsRwSet != nil {
		filteredTxPvtRwSet = &rwset.TxPvtReadWriteSet{
			DataModel:  pvtWSet.GetDataModel(),
			NsPvtRwset: filteredNsRwSet,
		}
	}
	return filteredTxPvtRwSet
}

func V11RetrievePvtdata(pvtDataResults map[string][]byte, filter ledger.PvtNsCollFilter) ([]*ledger.TxPvtData, error) {
	var blkPvtData []*ledger.TxPvtData
	for key, val := range pvtDataResults {
		pvtDatum, err := V11DecodeKV([]byte(key), val, filter)
		if err != nil {
			return nil, err
		}
		blkPvtData = append(blkPvtData, pvtDatum)
	}
	return blkPvtData, nil
}
