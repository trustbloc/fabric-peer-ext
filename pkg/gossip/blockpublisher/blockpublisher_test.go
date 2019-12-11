/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockpublisher

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/ledger/rwset/kvrwset"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/extensions/gossip/api"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/blockvisitor"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

const (
	channel1 = "channel1"
	channel2 = "channel2"

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

func TestProvider(t *testing.T) {
	l := &mocks.Ledger{BlockchainInfo: &cb.BlockchainInfo{Height: 1000}}
	l.BcInfoError = errors.New("injected error")
	lp := &mocks.LedgerProvider{}
	lp.GetLedgerReturns(l)

	provider := NewProvider().Initialize(lp)
	require.NotNil(t, provider)
	provider.ChannelJoined(channel1)
	require.NotPanicsf(t, func() { provider.ChannelJoined(channel1) }, "should not have panicked event though GetBlockchainInfo returned error")

	l.BcInfoError = nil
	require.NotPanics(t, func() { provider.ChannelJoined(channel1) })

	p1 := provider.ForChannel(channel1)
	require.NotNil(t, p1)
	require.Equal(t, uint64(1000), p1.LedgerHeight())

	p2 := provider.ForChannel(channel2)
	require.NotNil(t, p2)

	require.NotEqual(t, p1, p2)

	provider.Close()
	// Call Close again
	require.NotPanics(t, func() { provider.Close() })
}

func TestPublisher_Get(t *testing.T) {
	p := New("mychannel")
	require.NotNil(t, p)
	p.Close()
}

func TestPublisher_Close(t *testing.T) {
	p := New(channel1)
	require.NotNil(t, p)

	p.Close()

	assert.NotPanics(t, func() {
		p.Close()
	}, "Expecting Close to not panic when called multiple times")
}

func TestPublisher_PublishEndorsementEvents(t *testing.T) {
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

	p := New(channel1)
	require.NotNil(t, p)
	defer p.Close()

	handler1 := mocks.NewMockBlockHandler()
	p.AddReadHandler(handler1.HandleRead)
	p.AddWriteHandler(handler1.HandleWrite)

	handler2 := mocks.NewMockBlockHandler()
	p.AddReadHandler(handler2.HandleRead)
	p.AddCCEventHandler(handler2.HandleChaincodeEvent)

	handler3 := mocks.NewMockBlockHandler()
	p.AddCCUpgradeHandler(handler3.HandleChaincodeUpgradeEvent)
	p.AddLSCCWriteHandler(handler3.HandleLSCCWrite)

	b := mocks.NewBlockBuilder(channel1, 1100)

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
		ChaincodeAction(blockvisitor.LsccID).
		Write(ccID1, ccDataBytes).
		ChaincodeEvent(upgradeEvent, lceBytes)

	tb4 := b.Transaction(txID2, pb.TxValidationCode_VALID)
	tb4.ChaincodeAction(blockvisitor.LsccID).
		Write(ccID1, ccDataBytes).
		ChaincodeEvent(ccEvent1, nil)

	p.Publish(b.Build())

	// Wait a bit for the events to be published
	time.Sleep(500 * time.Millisecond)

	assert.Equal(t, 2, handler1.NumReads())
	assert.Equal(t, 5, handler1.NumWrites())
	assert.Equal(t, 0, handler1.NumCCEvents())
	assert.Equal(t, 0, handler1.NumCCUpgradeEvents())

	assert.Equal(t, 2, handler2.NumReads())
	assert.Equal(t, 0, handler2.NumWrites())
	assert.Equal(t, 3, handler2.NumCCEvents())
	assert.Equal(t, 0, handler2.NumCCUpgradeEvents())

	assert.Equal(t, 0, handler3.NumReads())
	assert.Equal(t, 0, handler3.NumWrites())
	assert.Equal(t, 0, handler3.NumCCEvents())
	assert.Equal(t, 1, handler3.NumCCUpgradeEvents())
	assert.Equal(t, 2, handler3.NumLSCCWrites())

	assert.EqualValues(t, 1101, p.LedgerHeight())
}

func TestPublisher_PublishConfigUpdateEvents(t *testing.T) {
	p := New(channel1)
	require.NotNil(t, p)
	defer p.Close()

	handler := mocks.NewMockBlockHandler()
	p.AddConfigUpdateHandler(handler.HandleConfigUpdate)

	b := mocks.NewBlockBuilder(channel1, 1100)
	b.ConfigUpdate()

	p.Publish(b.Build())

	// Wait a bit for the events to be published
	time.Sleep(500 * time.Millisecond)

	assert.Equal(t, 1, handler.NumConfigUpdates())

	assert.EqualValues(t, 1101, p.LedgerHeight())
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

func TestPublisher_LSCCWriteEvent(t *testing.T) {
	p := New(channel1)
	require.NotNil(t, p)
	defer p.Close()

	var info ccInfo

	p.AddLSCCWriteHandler(func(txMetadata api.TxMetadata, chaincodeName string, ccData *ccprovider.ChaincodeData, ccp *pb.CollectionConfigPackage) error {
		info.set(chaincodeName, ccData, ccp)
		return nil
	})

	b := mocks.NewBlockBuilder(channel1, 1100)

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
		ChaincodeAction(blockvisitor.LsccID).
		Write(ccID1, ccDataBytes).
		Write(ccID1+blockvisitor.CollectionSeparator+"collection", ccpBytes)

	p.Publish(b.Build())

	// Wait a bit for the events to be published
	time.Sleep(500 * time.Millisecond)

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

func TestPublisher_Error(t *testing.T) {
	var (
		value1 = []byte("value1")
		value2 = []byte("value2")

		v1 = &kvrwset.Version{
			BlockNum: 1000,
			TxNum:    3,
		}
		v2 = &kvrwset.Version{
			BlockNum: 1001,
			TxNum:    5,
		}
	)

	p := New(channel1)
	require.NotNil(t, p)
	defer p.Close()

	expectedErr := fmt.Errorf("injected error")

	handler1 := mocks.NewMockBlockHandler().WithError(expectedErr)
	p.AddReadHandler(handler1.HandleRead)
	p.AddWriteHandler(handler1.HandleWrite)
	p.AddCCEventHandler(handler1.HandleChaincodeEvent)
	p.AddCCUpgradeHandler(handler1.HandleChaincodeUpgradeEvent)

	b := mocks.NewBlockBuilder(channel1, 1100)

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

	lceBytes, err := proto.Marshal(&pb.LifecycleEvent{ChaincodeName: ccID2})
	require.NoError(t, err)
	require.NotNil(t, lceBytes)

	b.Transaction(txID1, pb.TxValidationCode_VALID).
		ChaincodeAction(blockvisitor.LsccID).
		ChaincodeEvent(upgradeEvent, lceBytes)
	tb4 := b.Transaction(txID2, pb.TxValidationCode_VALID)
	tb4.ChaincodeAction(blockvisitor.LsccID).
		ChaincodeEvent(ccEvent1, nil)

	p.Publish(b.Build())

	// Wait a bit for the events to be published
	time.Sleep(500 * time.Millisecond)

	assert.Equal(t, 2, handler1.NumReads())
	assert.Equal(t, 5, handler1.NumWrites())
	assert.Equal(t, 3, handler1.NumCCEvents())
	assert.Equal(t, 1, handler1.NumCCUpgradeEvents())

	assert.EqualValues(t, 1101, p.LedgerHeight())
}
