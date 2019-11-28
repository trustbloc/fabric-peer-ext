/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockvisitor

import (
	"fmt"
	"sync"
	"testing"

	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/ledger/rwset/kvrwset"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

const (
	channelID = "testchannel"

	txID1 = "tx1"
	txID2 = "tx2"
	txID3 = "tx3"

	ccID1 = "cc1"
	ccID2 = "cc2"

	coll1 = "collection1"
	coll2 = "collection2"

	key1 = "key1"
	key2 = "key2"
	key3 = "key3"

	ccEvent1 = "ccevent1"
)

func TestVisitor_HandleEndorsementEvents(t *testing.T) {
	numReads := 0
	numWrites := 0
	numLSCCWrites := 0
	numCCEvents := 0

	p := New(channelID,
		WithCCEventHandler(func(ccEvent *CCEvent) error {
			numCCEvents++
			return nil
		}),
		WithReadHandler(func(read *Read) error {
			numReads++
			return nil
		}),
		WithWriteHandler(func(write *Write) error {
			numWrites++
			return nil
		}),
		WithLSCCWriteHandler(func(lsccWrite *LSCCWrite) error {
			numLSCCWrites++
			return nil
		}),
	)
	require.NotNil(t, p)
	require.Equal(t, channelID, p.ChannelID())

	block := mockBlockWithTransactions(t)
	t.Run("No handlers", func(t *testing.T) {
		p := New(channelID)
		require.NoError(t, p.Visit(block))
		require.NotNil(t, p)
	})

	t.Run("With handlers", func(t *testing.T) {
		require.NoError(t, p.Visit(block))
		assert.Equal(t, 2, numReads)
		assert.Equal(t, 5, numWrites)
		assert.Equal(t, 2, numCCEvents)
		assert.Equal(t, 2, numLSCCWrites)
		assert.EqualValues(t, 1101, p.LedgerHeight())
	})
}

func TestVisitor_PublishConfigUpdateEvents(t *testing.T) {
	b := mocks.NewBlockBuilder(channelID, 1100)
	b.ConfigUpdate()

	t.Run("No handlers", func(t *testing.T) {
		p := New(channelID)
		require.NoError(t, p.Visit(b.Build()))
	})

	t.Run("With handlers", func(t *testing.T) {
		numConfigUpdates := 0

		p := New(channelID,
			WithConfigUpdateHandler(func(update *ConfigUpdate) error {
				numConfigUpdates++
				return nil
			}),
		)
		require.NotNil(t, p)

		require.NoError(t, p.Visit(b.Build()))
		assert.Equal(t, 1, numConfigUpdates)
		assert.EqualValues(t, 1101, p.LedgerHeight())
	})
}

func TestVisitor_LSCCWriteEvent(t *testing.T) {
	numLSCCWrites := 0

	var info ccInfo

	p := New(channelID,
		WithLSCCWriteHandler(func(lsccWrite *LSCCWrite) error {
			numLSCCWrites++
			info.set(lsccWrite.CCID, lsccWrite.CCData, lsccWrite.CCP)
			return nil
		}),
	)
	require.NotNil(t, p)

	b := mocks.NewBlockBuilder(channelID, 1100)

	ccData := &ccprovider.ChaincodeData{
		Name: ccID1,
	}
	ccDataBytes, err := proto.Marshal(ccData)
	require.NoError(t, err)

	ccp := &pb.CollectionConfigPackage{
		Config: []*pb.CollectionConfig{
			{
				Payload: &pb.CollectionConfig_StaticCollectionConfig{
					StaticCollectionConfig: &pb.StaticCollectionConfig{
						Name: coll1,
						Type: pb.CollectionType_COL_TRANSIENT,
					},
				},
			},
			{
				Payload: &pb.CollectionConfig_StaticCollectionConfig{
					StaticCollectionConfig: &pb.StaticCollectionConfig{
						Name: coll2,
						Type: pb.CollectionType_COL_OFFLEDGER,
					},
				},
			},
		},
	}
	ccpBytes, err := proto.Marshal(ccp)
	require.NoError(t, err)

	b.Transaction(txID1, pb.TxValidationCode_VALID).
		ChaincodeAction(LsccID).
		Write(ccID1, ccDataBytes).
		Write(ccID1+CollectionSeparator+"collection", ccpBytes)

	require.NoError(t, p.Visit(b.Build()))
	require.Equal(t, ccID1, info.getCCName())
	require.NotNil(t, info.getCCData())
	require.Equal(t, ccData.Name, info.getCCData().Name)
	require.NotNil(t, info.getCCP())
	require.Equal(t, 2, len(info.getCCP().Config))

	config1 := info.getCCP().Config[0].GetStaticCollectionConfig()
	require.NotNil(t, config1)
	require.Equal(t, coll1, config1.Name)
	require.Equal(t, pb.CollectionType_COL_TRANSIENT, config1.Type)

	config2 := info.getCCP().Config[1].GetStaticCollectionConfig()
	require.NotNil(t, config2)
	require.Equal(t, coll2, config2.Name)
	require.Equal(t, pb.CollectionType_COL_OFFLEDGER, config2.Type)
}

func TestVisitor_LSCCWriteEventMarshalError(t *testing.T) {
	numLSCCWrites := 0

	var info ccInfo

	p := New(channelID,
		WithLSCCWriteHandler(func(lsccWrite *LSCCWrite) error {
			numLSCCWrites++
			info.set(lsccWrite.CCID, lsccWrite.CCData, lsccWrite.CCP)
			return nil
		}),
	)
	require.NotNil(t, p)

	ccData := &ccprovider.ChaincodeData{
		Name: ccID1,
	}
	ccDataBytes, err := proto.Marshal(ccData)
	require.NoError(t, err)

	ccp := &pb.CollectionConfigPackage{
		Config: []*pb.CollectionConfig{
			{
				Payload: &pb.CollectionConfig_StaticCollectionConfig{
					StaticCollectionConfig: &pb.StaticCollectionConfig{
						Name: coll1,
					},
				},
			},
		},
	}
	ccpBytes, err := proto.Marshal(ccp)
	require.NoError(t, err)

	t.Run("CCData unmarshal error", func(t *testing.T) {
		b := mocks.NewBlockBuilder(channelID, 1100)

		b.Transaction(txID1, pb.TxValidationCode_VALID).
			ChaincodeAction(LsccID).
			Write(ccID1, []byte("invalid cc data")).
			Write(ccID1+CollectionSeparator+"collection", ccpBytes)

		err := p.Visit(b.Build())
		require.Error(t, err)
		require.Contains(t, err.Error(), "error unmarshaling chaincode data")
		require.Empty(t, info.ccName)
		require.Nil(t, info.ccData)
		require.Nil(t, info.ccp)
	})

	t.Run("CCP unmarshal error", func(t *testing.T) {
		b := mocks.NewBlockBuilder(channelID, 1100)

		b.Transaction(txID1, pb.TxValidationCode_VALID).
			ChaincodeAction(LsccID).
			Write(ccID1, ccDataBytes).
			Write(ccID1+CollectionSeparator+"collection", []byte("invalid ccp"))

		err := p.Visit(b.Build())
		require.Error(t, err)
		require.Contains(t, err.Error(), "error unmarshaling collection configuration")
		require.Empty(t, info.ccName)
		require.Nil(t, info.ccData)
		require.Nil(t, info.ccp)
	})
}

func TestVisitor_Error(t *testing.T) {
	p := New(channelID)
	pNoStop := New(channelID, WithNoStopOnError())

	block := mockBlockWithTransactions(t)

	t.Run("Unmarshal error", func(t *testing.T) {
		errExpected := errors.New("injected Unmarshal error")
		restore := unmarshal
		unmarshal = func(buf []byte, pb proto.Message) error { return errExpected }
		defer func() { unmarshal = restore }()

		err := p.Visit(block)
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.NoError(t, pNoStop.Visit(block))
	})

	t.Run("ExtractEnvelope error", func(t *testing.T) {
		errExpected := errors.New("injected ExtractEnvelope error")
		restore := extractEnvelope
		extractEnvelope = func(block *cb.Block, index int) (envelope *cb.Envelope, e error) { return nil, errExpected }
		defer func() { extractEnvelope = restore }()

		err := p.Visit(block)
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.NoError(t, pNoStop.Visit(block))
	})

	t.Run("ExtractPayload error", func(t *testing.T) {
		errExpected := errors.New("injected ExtractPayload error")
		restore := extractPayload
		extractPayload = func(envelope *cb.Envelope) (payload *cb.Payload, e error) { return nil, errExpected }
		defer func() { extractPayload = restore }()

		err := p.Visit(block)
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.NoError(t, pNoStop.Visit(block))
	})

	t.Run("UnmarshalChannelHeader error", func(t *testing.T) {
		errExpected := errors.New("injected UnmarshalChannelHeader error")
		restore := unmarshalChannelHeader
		unmarshalChannelHeader = func(bytes []byte) (header *cb.ChannelHeader, e error) { return nil, errExpected }
		defer func() { unmarshalChannelHeader = restore }()

		err := p.Visit(block)
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.NoError(t, pNoStop.Visit(block))
	})

	t.Run("GetTransaction error", func(t *testing.T) {
		errExpected := errors.New("injected GetTransaction error")
		restore := getTransaction
		getTransaction = func(txBytes []byte) (transaction *pb.Transaction, e error) { return nil, errExpected }
		defer func() { getTransaction = restore }()

		err := p.Visit(block)
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.NoError(t, pNoStop.Visit(block))
	})

	t.Run("GetChaincodeActionPayload error", func(t *testing.T) {
		errExpected := errors.New("injected GetChaincodeActionPayload error")
		restore := getChaincodeActionPayload
		getChaincodeActionPayload = func(capBytes []byte) (payload *pb.ChaincodeActionPayload, e error) { return nil, errExpected }
		defer func() { getChaincodeActionPayload = restore }()

		err := p.Visit(block)
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.NoError(t, pNoStop.Visit(block))
	})
}

func TestVisitor_NoStopOnError(t *testing.T) {
	expectedErr := fmt.Errorf("injected error")
	numReads := 0
	numWrites := 0
	numLSCCWrites := 0
	numCCEvents := 0

	p := New(channelID, WithNoStopOnError(),
		WithCCEventHandler(func(ccEvent *CCEvent) error {
			numCCEvents++
			return expectedErr
		}),
		WithReadHandler(func(read *Read) error {
			numReads++
			return expectedErr
		}),
		WithWriteHandler(func(write *Write) error {
			numWrites++
			return expectedErr
		}),
		WithLSCCWriteHandler(func(lsccWrite *LSCCWrite) error {
			numLSCCWrites++
			return expectedErr
		}),
	)
	require.NotNil(t, p)

	require.NoError(t, p.Visit(mockBlockWithTransactions(t)))
	assert.Equal(t, 2, numReads)
	assert.Equal(t, 5, numWrites)
	assert.Equal(t, 2, numCCEvents)
	assert.EqualValues(t, 1101, p.LedgerHeight())
}

func TestVisitor_SetLastCommittedBlockNum(t *testing.T) {
	v := New(channelID)
	v.SetLastCommittedBlockNum(999)
	require.Equal(t, uint64(1000), v.LedgerHeight())

	v.SetLastCommittedBlockNum(1)
	require.Equalf(t, uint64(1000), v.LedgerHeight(), "should not have been able to set the block number to be lower than the current number")

	v.SetLastCommittedBlockNum(1000)
	require.Equal(t, uint64(1001), v.LedgerHeight())
}

type ccInfo struct {
	mutex  sync.RWMutex
	ccName string
	ccData *ccprovider.ChaincodeData
	ccp    *pb.CollectionConfigPackage
}

func (info *ccInfo) set(ccName string, ccData *ccprovider.ChaincodeData, ccp *pb.CollectionConfigPackage) {
	info.mutex.Lock()
	defer info.mutex.Unlock()

	info.ccName = ccName
	info.ccData = ccData
	info.ccp = ccp
}

func (info *ccInfo) getCCName() string {
	info.mutex.RLock()
	defer info.mutex.RUnlock()
	return info.ccName
}

func (info *ccInfo) getCCData() *ccprovider.ChaincodeData {
	info.mutex.RLock()
	defer info.mutex.RUnlock()
	return info.ccData
}

func (info *ccInfo) getCCP() *pb.CollectionConfigPackage {
	info.mutex.RLock()
	defer info.mutex.RUnlock()
	return info.ccp
}

func mockBlockWithTransactions(t *testing.T) *cb.Block {
	var (
		value1 = []byte("value1")
		value2 = []byte("value2")
		value3 = []byte("value3")

		v1 = &kvrwset.Version{
			BlockNum: 1000,
			TxNum:    0,
		}
		v2 = &kvrwset.Version{
			BlockNum: 1001,
			TxNum:    1,
		}
	)

	b := mocks.NewBlockBuilder(channelID, 1100)

	tb1 := b.Transaction(txID1, pb.TxValidationCode_VALID)
	tb1.ChaincodeAction(ccID1).
		Write(key1, value1).
		Read(key1, v1).
		ChaincodeEvent(ccEvent1, []byte("ccpayload"))
	tb1.ChaincodeAction(ccID2).
		Write(key2, value2).
		Read(key2, v2)

	tb2 := b.Transaction(txID2, pb.TxValidationCode_VALID)
	cc2_1 := tb2.ChaincodeAction(ccID1).
		Write(key2, value2)
	cc2_1.Collection(coll1).
		Write(key1, value2)
	cc2_1.Collection(coll2).
		Delete(key1)

	// This transaction should not be published
	tb3 := b.Transaction(txID3, pb.TxValidationCode_MVCC_READ_CONFLICT)
	tb3.ChaincodeAction(ccID1).
		Write(key3, value3).
		ChaincodeEvent(ccEvent1, []byte("ccpayload"))

	lceBytes, err := proto.Marshal(&pb.LifecycleEvent{ChaincodeName: ccID2})
	require.NoError(t, err)
	require.NotNil(t, lceBytes)

	ccData := &ccprovider.ChaincodeData{
		Name: ccID1,
	}
	ccDataBytes, err := proto.Marshal(ccData)
	require.NoError(t, err)

	b.Transaction(txID1, pb.TxValidationCode_VALID).
		ChaincodeAction(LsccID).
		Write(ccID1, ccDataBytes)

	tb4 := b.Transaction(txID2, pb.TxValidationCode_VALID)
	tb4.ChaincodeAction(LsccID).
		Write(ccID1, ccDataBytes).
		ChaincodeEvent(ccEvent1, nil)

	return b.Build()
}
