/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"bytes"
	"math"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/core/ledger/pvtdatastorage"
	"github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/pkg/errors"
	"github.com/willf/bitset"
)

// todo add pinning script to include copied code into this file, original file from fabric is found in fabric/core/ledger/pvtdatastorage/kv_encoding.go
// todo below functions are originally unexported, the pinning script must capitalize these functions to export them

var (
	PendingCommitKey               = []byte{0}
	pvtDataKeyPrefix               = []byte{2}
	expiryKeyPrefix                = []byte{3}
	eligibleMissingDataKeyPrefix   = []byte{4}
	ineligibleMissingDataKeyPrefix = []byte{5}
	collElgKeyPrefix               = []byte{6}
	LastUpdatedOldBlocksKey        = []byte{7}
	lastCommittedBlockKey          = []byte{8}
	nilByte                        = byte(0)
)

func EncodeDataKey(key *DataKey) []byte {
	dataKeyBytes := append(pvtDataKeyPrefix, version.NewHeight(key.BlkNum, key.TxNum).ToBytes()...)
	dataKeyBytes = append(dataKeyBytes, []byte(key.Ns)...)
	dataKeyBytes = append(dataKeyBytes, nilByte)
	return append(dataKeyBytes, []byte(key.Coll)...)
}

func EncodeDataValue(collData *rwset.CollectionPvtReadWriteSet) ([]byte, error) {
	return proto.Marshal(collData)
}

func EncodeExpiryKey(expiryKey *ExpiryKey) []byte {
	// reusing version encoding scheme here
	return append(expiryKeyPrefix, version.NewHeight(expiryKey.ExpiringBlk, expiryKey.CommittingBlk).ToBytes()...)
}

func DecodeExpiryKey(expiryKeyBytes []byte) *ExpiryKey {
	height, _ := version.NewHeightFromBytes(expiryKeyBytes[1:])
	return &ExpiryKey{ExpiringBlk: height.BlockNum, CommittingBlk: height.TxNum}
}

func EncodeExpiryValue(expiryData *ExpiryData) ([]byte, error) {
	return proto.Marshal(expiryData)
}

func DecodeExpiryValue(expiryValueBytes []byte) (*ExpiryData, error) {
	expiryData := &ExpiryData{}
	err := proto.Unmarshal(expiryValueBytes, expiryData)
	return expiryData, err
}

func DecodeDatakey(datakeyBytes []byte) *DataKey {
	v, n := version.NewHeightFromBytes(datakeyBytes[1:])
	blkNum := v.BlockNum
	tranNum := v.TxNum
	remainingBytes := datakeyBytes[n+1:]
	nilByteIndex := bytes.IndexByte(remainingBytes, nilByte)
	ns := string(remainingBytes[:nilByteIndex])
	coll := string(remainingBytes[nilByteIndex+1:])
	return &DataKey{NsCollBlk: NsCollBlk{Ns: ns, Coll: coll, BlkNum: blkNum}, TxNum: tranNum}
}

func DecodeDataValue(datavalueBytes []byte) (*rwset.CollectionPvtReadWriteSet, error) {
	collPvtdata := &rwset.CollectionPvtReadWriteSet{}
	err := proto.Unmarshal(datavalueBytes, collPvtdata)
	return collPvtdata, err
}

func EncodeMissingDataKey(key *MissingDataKey) []byte {
	if key.IsEligible {
		keyBytes := append(eligibleMissingDataKeyPrefix, util.EncodeReverseOrderVarUint64(key.BlkNum)...)
		keyBytes = append(keyBytes, []byte(key.Ns)...)
		keyBytes = append(keyBytes, nilByte)
		return append(keyBytes, []byte(key.Coll)...)
	}

	keyBytes := append(ineligibleMissingDataKeyPrefix, []byte(key.Ns)...)
	keyBytes = append(keyBytes, nilByte)
	keyBytes = append(keyBytes, []byte(key.Coll)...)
	keyBytes = append(keyBytes, nilByte)
	return append(keyBytes, []byte(util.EncodeReverseOrderVarUint64(key.BlkNum))...)
}

func decodeMissingDataKey(keyBytes []byte) *MissingDataKey {
	key := &MissingDataKey{NsCollBlk: NsCollBlk{}}
	if keyBytes[0] == eligibleMissingDataKeyPrefix[0] {
		blkNum, numBytesConsumed := util.DecodeReverseOrderVarUint64(keyBytes[1:])

		splittedKey := bytes.Split(keyBytes[numBytesConsumed+1:], []byte{nilByte})
		key.Ns = string(splittedKey[0])
		key.Coll = string(splittedKey[1])
		key.BlkNum = blkNum
		key.IsEligible = true
		return key
	}

	splittedKey := bytes.SplitN(keyBytes[1:], []byte{nilByte}, 3) //encoded bytes for blknum may contain empty bytes
	key.Ns = string(splittedKey[0])
	key.Coll = string(splittedKey[1])
	key.BlkNum, _ = util.DecodeReverseOrderVarUint64(splittedKey[2])
	key.IsEligible = false
	return key
}

func EncodeMissingDataValue(bitmap *bitset.BitSet) ([]byte, error) {
	return bitmap.MarshalBinary()
}

func DecodeMissingDataValue(bitmapBytes []byte) (*bitset.BitSet, error) {
	bitmap := &bitset.BitSet{}
	if err := bitmap.UnmarshalBinary(bitmapBytes); err != nil {
		return nil, err
	}
	return bitmap, nil
}

func encodeCollElgKey(blkNum uint64) []byte {
	return append(collElgKeyPrefix, util.EncodeReverseOrderVarUint64(blkNum)...)
}

func decodeCollElgKey(b []byte) uint64 {
	blkNum, _ := util.DecodeReverseOrderVarUint64(b[1:])
	return blkNum
}

func encodeCollElgVal(m *pvtdatastorage.CollElgInfo) ([]byte, error) {
	return proto.Marshal(m)
}

func decodeCollElgVal(b []byte) (*pvtdatastorage.CollElgInfo, error) {
	m := &pvtdatastorage.CollElgInfo{}
	if err := proto.Unmarshal(b, m); err != nil {
		return nil, errors.WithStack(err)
	}
	return m, nil
}

func createRangeScanKeysForIneligibleMissingData(maxBlkNum uint64, ns, coll string) (startKey, endKey []byte) {
	startKey = EncodeMissingDataKey(
		&MissingDataKey{
			NsCollBlk:  NsCollBlk{Ns: ns, Coll: coll, BlkNum: maxBlkNum},
			IsEligible: false,
		},
	)
	endKey = EncodeMissingDataKey(
		&MissingDataKey{
			NsCollBlk:  NsCollBlk{Ns: ns, Coll: coll, BlkNum: 0},
			IsEligible: false,
		},
	)
	return
}

func createRangeScanKeysForEligibleMissingDataEntries(blkNum uint64) (startKey, endKey []byte) {
	startKey = append(eligibleMissingDataKeyPrefix, util.EncodeReverseOrderVarUint64(blkNum)...)
	endKey = append(eligibleMissingDataKeyPrefix, util.EncodeReverseOrderVarUint64(0)...)

	return startKey, endKey
}

func createRangeScanKeysForCollElg() (startKey, endKey []byte) {
	return encodeCollElgKey(math.MaxUint64),
		encodeCollElgKey(0)
}

func datakeyRange(blockNum uint64) (startKey, endKey []byte) {
	startKey = append(pvtDataKeyPrefix, version.NewHeight(blockNum, 0).ToBytes()...)
	endKey = append(pvtDataKeyPrefix, version.NewHeight(blockNum, math.MaxUint64).ToBytes()...)
	return
}

func eligibleMissingdatakeyRange(blkNum uint64) (startKey, endKey []byte) {
	startKey = append(eligibleMissingDataKeyPrefix, util.EncodeReverseOrderVarUint64(blkNum)...)
	endKey = append(eligibleMissingDataKeyPrefix, util.EncodeReverseOrderVarUint64(blkNum-1)...)
	return
}
