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
	"github.com/hyperledger/fabric/core/ledger"
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
	txID4 = "tx4"
	txID5 = "tx5"

	ccID1 = "cc1"
	ccID2 = "cc2"

	coll1 = "collection1"
	coll2 = "collection2"

	key1 = "key1"
	key2 = "key2"
	key3 = "key3"

	ccEvent1 = "ccevent1"
)

var (
	hashedKey1   = []byte(key1)
	hashedValue1 = []byte("value1")
)

func TestVisitor_HandleEndorsementEvents(t *testing.T) {
	block, pvtData := mockBlockWithTransactions(t)

	t.Run("No handlers", func(t *testing.T) {
		p := New(channelID)
		require.NoError(t, p.Visit(block, nil))
		require.NotNil(t, p)
	})

	t.Run("With handlers", func(t *testing.T) {
		var ccEvents []*CCEvent
		var reads []*Read
		var writes []*Write
		var collHashReads []*CollHashRead
		var collHashWrites []*CollHashWrite
		var lsccWrites []*LSCCWrite
		var blocks []*cb.Block

		p := New(channelID,
			WithCCEventHandler(func(ccEvent *CCEvent) error {
				ccEvents = append(ccEvents, ccEvent)
				return nil
			}),
			WithReadHandler(func(read *Read) error {
				reads = append(reads, read)
				return nil
			}),
			WithWriteHandler(func(write *Write) error {
				writes = append(writes, write)
				return nil
			}),
			WithCollHashReadHandler(func(read *CollHashRead) error {
				collHashReads = append(collHashReads, read)
				return nil
			}),
			WithCollHashWriteHandler(func(write *CollHashWrite) error {
				collHashWrites = append(collHashWrites, write)
				return nil
			}),
			WithLSCCWriteHandler(func(lsccWrite *LSCCWrite) error {
				lsccWrites = append(lsccWrites, lsccWrite)
				return nil
			}),
			WithBlockHandler(func(block *cb.Block) error {
				blocks = append(blocks, block)
				return nil
			}),
		)
		require.NotNil(t, p)
		require.Equal(t, channelID, p.ChannelID())

		require.NoError(t, p.Visit(block, pvtData))
		assert.EqualValues(t, 1101, p.LedgerHeight())

		require.Len(t, blocks, 1)

		require.Len(t, ccEvents, 2)
		assert.Equal(t, uint64(0), ccEvents[0].TxNum)
		assert.Equal(t, txID1, ccEvents[0].TxID)
		assert.Equal(t, uint64(4), ccEvents[1].TxNum)
		assert.Equal(t, txID5, ccEvents[1].TxID)

		require.Len(t, reads, 3) // 2 public state and 1 private collection read
		assert.Equal(t, uint64(0), reads[0].TxNum)
		assert.Equal(t, txID1, reads[0].TxID)
		assert.Equal(t, uint64(0), reads[1].TxNum)
		assert.Equal(t, txID1, reads[1].TxID)
		assert.Equal(t, uint64(0), reads[2].TxNum)
		assert.Equal(t, txID1, reads[2].TxID)

		require.Len(t, writes, 5) // 3 public state and 2 private collection writes
		assert.Equal(t, uint64(0), writes[0].TxNum)
		assert.Equal(t, txID1, writes[0].TxID)
		assert.Equal(t, uint64(0), writes[1].TxNum)
		assert.Equal(t, txID1, writes[1].TxID)
		assert.Equal(t, uint64(0), writes[2].TxNum)
		assert.Equal(t, txID1, writes[2].TxID)
		assert.Equal(t, uint64(0), writes[3].TxNum)
		assert.Equal(t, txID1, writes[3].TxID)
		assert.Equal(t, uint64(1), writes[4].TxNum)
		assert.Equal(t, txID2, writes[4].TxID)

		require.Len(t, collHashReads, 1)
		assert.Equal(t, uint64(0), reads[0].TxNum)
		assert.Equal(t, txID1, reads[0].TxID)

		require.Len(t, collHashWrites, 2)
		assert.Equal(t, uint64(0), writes[0].TxNum)
		assert.Equal(t, txID1, writes[0].TxID)
		assert.Equal(t, uint64(0), writes[1].TxNum)
		assert.Equal(t, txID1, writes[1].TxID)

		require.Len(t, lsccWrites, 2)
		assert.Equal(t, uint64(3), lsccWrites[0].TxNum)
		assert.Equal(t, txID4, lsccWrites[0].TxID)
		assert.Equal(t, uint64(4), lsccWrites[1].TxNum)
		assert.Equal(t, txID5, lsccWrites[1].TxID)
	})
}

func TestVisitor_ErrorHandler(t *testing.T) {
	block, pvtData := mockBlockWithTransactions(t)

	t.Run("With CCEventHandler error", func(t *testing.T) {
		errExpected := fmt.Errorf("injected event handler error")
		handler := func(*CCEvent) error { return errExpected }

		t.Run("Ignore", func(t *testing.T) {
			p := New(channelID,
				WithCCEventHandler(handler),
			)
			require.NotNil(t, p)
			require.Equal(t, channelID, p.ChannelID())
			require.NoError(t, p.Visit(block, pvtData))
		})

		t.Run("Halt", func(t *testing.T) {
			p := New(channelID,
				WithErrorHandler(func(err error, ctx *Context) error {
					require.Equal(t, CCEventHandlerErr, ctx.Category)
					require.Equal(t, channelID, ctx.ChannelID)
					require.NotNil(t, ctx.CCEvent)

					return err
				}),
				WithCCEventHandler(handler),
			)
			require.NotNil(t, p)
			require.Equal(t, channelID, p.ChannelID())

			err := p.Visit(block, pvtData)
			require.Error(t, err)
			require.Contains(t, err.Error(), errExpected.Error())
		})
	})

	t.Run("With ReadHandler error", func(t *testing.T) {
		errExpected := fmt.Errorf("injected read handler error")
		handler := func(*Read) error { return errExpected }

		t.Run("Ignore", func(t *testing.T) {
			p := New(channelID,
				WithReadHandler(handler),
			)
			require.NotNil(t, p)
			require.Equal(t, channelID, p.ChannelID())
			require.NoError(t, p.Visit(block, pvtData))
		})

		t.Run("Halt", func(t *testing.T) {
			p := New(channelID,
				WithErrorHandler(func(err error, ctx *Context) error {
					require.Equal(t, ReadHandlerErr, ctx.Category)
					require.Equal(t, channelID, ctx.ChannelID)
					require.NotNil(t, ctx.Read)

					return err
				}),
				WithReadHandler(handler),
			)
			require.NotNil(t, p)
			require.Equal(t, channelID, p.ChannelID())

			err := p.Visit(block, pvtData)
			require.Error(t, err)
			require.Contains(t, err.Error(), errExpected.Error())
		})
	})

	t.Run("With WriteHandler error", func(t *testing.T) {
		errExpected := fmt.Errorf("injected write handler error")
		handler := func(*Write) error { return errExpected }

		t.Run("Ignore", func(t *testing.T) {
			p := New(channelID,
				WithWriteHandler(handler),
			)
			require.NotNil(t, p)
			require.Equal(t, channelID, p.ChannelID())
			require.NoError(t, p.Visit(block, pvtData))
		})

		t.Run("Halt", func(t *testing.T) {
			p := New(channelID,
				WithErrorHandler(func(err error, ctx *Context) error {
					require.Equal(t, WriteHandlerErr, ctx.Category)
					require.Equal(t, channelID, ctx.ChannelID)
					require.NotNil(t, ctx.Write)

					return err
				}),
				WithWriteHandler(handler),
			)
			require.NotNil(t, p)
			require.Equal(t, channelID, p.ChannelID())

			err := p.Visit(block, pvtData)
			require.Error(t, err)
			require.Contains(t, err.Error(), errExpected.Error())
		})
	})

	t.Run("With CollHashReadHandler error", func(t *testing.T) {
		errExpected := fmt.Errorf("injected coll hash read handler error")
		handler := func(*CollHashRead) error { return errExpected }

		t.Run("Ignore", func(t *testing.T) {
			p := New(channelID,
				WithCollHashReadHandler(handler),
			)
			require.NotNil(t, p)
			require.Equal(t, channelID, p.ChannelID())
			require.NoError(t, p.Visit(block, pvtData))
		})

		t.Run("Halt", func(t *testing.T) {
			p := New(channelID,
				WithErrorHandler(func(err error, ctx *Context) error {
					require.Equal(t, CollHashReadHandlerErr, ctx.Category)
					require.Equal(t, channelID, ctx.ChannelID)
					require.NotNil(t, ctx.CollHashRead)

					return err
				}),
				WithCollHashReadHandler(handler),
			)
			require.NotNil(t, p)
			require.Equal(t, channelID, p.ChannelID())

			err := p.Visit(block, pvtData)
			require.Error(t, err)
			require.Contains(t, err.Error(), errExpected.Error())
		})
	})

	t.Run("With CollHashWriteHandler error", func(t *testing.T) {
		errExpected := fmt.Errorf("injected coll hash write handler error")
		handler := func(*CollHashWrite) error { return errExpected }

		t.Run("Ignore", func(t *testing.T) {
			p := New(channelID,
				WithCollHashWriteHandler(handler),
			)
			require.NotNil(t, p)
			require.Equal(t, channelID, p.ChannelID())
			require.NoError(t, p.Visit(block, pvtData))
		})

		t.Run("Halt", func(t *testing.T) {
			p := New(channelID,
				WithErrorHandler(func(err error, ctx *Context) error {
					require.Equal(t, CollHashWriteHandlerErr, ctx.Category)
					require.Equal(t, channelID, ctx.ChannelID)
					require.NotNil(t, ctx.CollHashWrite)

					return err
				}),
				WithCollHashWriteHandler(handler),
			)
			require.NotNil(t, p)
			require.Equal(t, channelID, p.ChannelID())

			err := p.Visit(block, pvtData)
			require.Error(t, err)
			require.Contains(t, err.Error(), errExpected.Error())
		})
	})

	t.Run("With LSCCWriteHandler error", func(t *testing.T) {
		errExpected := fmt.Errorf("injected LSCC write handler error")
		handler := func(*LSCCWrite) error { return errExpected }

		t.Run("Ignore", func(t *testing.T) {
			p := New(channelID,
				WithLSCCWriteHandler(handler),
			)
			require.NotNil(t, p)
			require.Equal(t, channelID, p.ChannelID())
			require.NoError(t, p.Visit(block, pvtData))
		})

		t.Run("Halt", func(t *testing.T) {
			p := New(channelID,
				WithErrorHandler(func(err error, ctx *Context) error {
					require.Equal(t, LSCCWriteHandlerErr, ctx.Category)
					require.Equal(t, channelID, ctx.ChannelID)
					require.NotNil(t, ctx.LSCCWrite)

					return err
				}),
				WithLSCCWriteHandler(handler),
			)
			require.NotNil(t, p)
			require.Equal(t, channelID, p.ChannelID())

			err := p.Visit(block, pvtData)
			require.Error(t, err)
			require.Contains(t, err.Error(), errExpected.Error())
		})
	})

	t.Run("With ConfigUpdateHandler error", func(t *testing.T) {
		bb := mocks.NewBlockBuilder(channelID, 1100)
		bb.ConfigUpdate()
		block := bb.Build()

		errExpected := fmt.Errorf("injected config update handler error")
		handler := func(*ConfigUpdate) error { return errExpected }

		t.Run("Ignore", func(t *testing.T) {
			p := New(channelID,
				WithConfigUpdateHandler(handler),
			)
			require.NotNil(t, p)
			require.Equal(t, channelID, p.ChannelID())
			require.NoError(t, p.Visit(block, pvtData))
		})

		t.Run("Halt", func(t *testing.T) {
			p := New(channelID,
				WithErrorHandler(func(err error, ctx *Context) error {
					require.Equal(t, ConfigUpdateHandlerErr, ctx.Category)
					require.Equal(t, channelID, ctx.ChannelID)
					require.NotNil(t, ctx.ConfigUpdate)

					return err
				}),
				WithConfigUpdateHandler(handler),
			)
			require.NotNil(t, p)
			require.Equal(t, channelID, p.ChannelID())

			err := p.Visit(block, pvtData)
			require.Error(t, err)
			require.Contains(t, err.Error(), errExpected.Error())
		})
	})

	t.Run("With PublishedBlockHandler error", func(t *testing.T) {
		errExpected := fmt.Errorf("injected block handler error")
		handler := func(block2 *cb.Block) error { return errExpected }

		t.Run("Ignore", func(t *testing.T) {
			p := New(channelID,
				WithBlockHandler(handler),
			)
			require.NotNil(t, p)
			require.Equal(t, channelID, p.ChannelID())
			require.NoError(t, p.Visit(block, pvtData))
		})

		t.Run("Halt", func(t *testing.T) {
			p := New(channelID,
				WithErrorHandler(func(err error, ctx *Context) error {
					require.Equal(t, BlockHandlerErr, ctx.Category)
					require.Equal(t, channelID, ctx.ChannelID)

					return err
				}),
				WithBlockHandler(handler),
			)
			require.NotNil(t, p)
			require.Equal(t, channelID, p.ChannelID())

			err := p.Visit(block, pvtData)
			require.Error(t, err)
			require.Contains(t, err.Error(), errExpected.Error())
		})
	})
}

func TestVisitor_PublishConfigUpdateEvents(t *testing.T) {
	b := mocks.NewBlockBuilder(channelID, 1100)
	b.ConfigUpdate()

	t.Run("No handlers", func(t *testing.T) {
		p := New(channelID)
		require.NoError(t, p.Visit(b.Build(), nil))
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

		require.NoError(t, p.Visit(b.Build(), nil))
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

	require.NoError(t, p.Visit(b.Build(), nil))
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

	handler := func(lsccWrite *LSCCWrite) error {
		numLSCCWrites++
		info.set(lsccWrite.CCID, lsccWrite.CCData, lsccWrite.CCP)
		return nil
	}

	v := New(channelID,
		WithLSCCWriteHandler(handler),
	)
	require.NotNil(t, v)

	vHalt := New(channelID,
		WithErrorHandler(func(err error, ctx *Context) error { return err }),
		WithLSCCWriteHandler(handler),
	)
	require.NotNil(t, vHalt)

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

		require.NoError(t, v.Visit(b.Build(), nil))
		require.Empty(t, info.ccName)
		require.Nil(t, info.ccData)
		require.Nil(t, info.ccp)

		require.Error(t, vHalt.Visit(b.Build(), nil))
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

		require.NoError(t, v.Visit(b.Build(), nil))
		require.Empty(t, info.ccName)
		require.Nil(t, info.ccData)
		require.Nil(t, info.ccp)

		require.Error(t, vHalt.Visit(b.Build(), nil))
		require.Empty(t, info.ccName)
		require.Nil(t, info.ccData)
		require.Nil(t, info.ccp)
	})
}

func TestVisitor_Error(t *testing.T) {
	v := New(channelID)

	vHalt := New(channelID, WithErrorHandler(func(err error, ctx *Context) error {
		return err
	}))

	block, pvtData := mockBlockWithTransactions(t)

	t.Run("Unmarshal error", func(t *testing.T) {
		errExpected := errors.New("injected Unmarshal error")
		restore := unmarshal
		unmarshal = func(buf []byte, pb proto.Message) error { return errExpected }
		defer func() { unmarshal = restore }()

		err := vHalt.Visit(block, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.NoError(t, v.Visit(block, pvtData))
	})

	t.Run("ExtractEnvelope error", func(t *testing.T) {
		errExpected := errors.New("injected ExtractEnvelope error")
		restore := extractEnvelope
		extractEnvelope = func(block *cb.Block, index int) (envelope *cb.Envelope, e error) { return nil, errExpected }
		defer func() { extractEnvelope = restore }()

		err := vHalt.Visit(block, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.NoError(t, v.Visit(block, pvtData))
	})

	t.Run("ExtractPayload error", func(t *testing.T) {
		errExpected := errors.New("injected ExtractPayload error")
		restore := extractPayload
		extractPayload = func(envelope *cb.Envelope) (payload *cb.Payload, e error) { return nil, errExpected }
		defer func() { extractPayload = restore }()

		err := vHalt.Visit(block, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.NoError(t, v.Visit(block, pvtData))
	})

	t.Run("UnmarshalChannelHeader error", func(t *testing.T) {
		errExpected := errors.New("injected UnmarshalChannelHeader error")
		restore := unmarshalChannelHeader
		unmarshalChannelHeader = func(bytes []byte) (header *cb.ChannelHeader, e error) { return nil, errExpected }
		defer func() { unmarshalChannelHeader = restore }()

		err := vHalt.Visit(block, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.NoError(t, v.Visit(block, pvtData))
	})

	t.Run("GetTransaction error", func(t *testing.T) {
		errExpected := errors.New("injected GetTransaction error")
		restore := getTransaction
		getTransaction = func(txBytes []byte) (transaction *pb.Transaction, e error) { return nil, errExpected }
		defer func() { getTransaction = restore }()

		err := vHalt.Visit(block, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.NoError(t, v.Visit(block, pvtData))
	})

	t.Run("GetChaincodeActionPayload error", func(t *testing.T) {
		errExpected := errors.New("injected GetChaincodeActionPayload error")
		restore := getChaincodeActionPayload
		getChaincodeActionPayload = func(capBytes []byte) (payload *pb.ChaincodeActionPayload, e error) { return nil, errExpected }
		defer func() { getChaincodeActionPayload = restore }()

		err := vHalt.Visit(block, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.NoError(t, v.Visit(block, pvtData))
	})

	t.Run("Unmarshal pvt data RW set error", func(t *testing.T) {
		errExpected := errors.New("injected Unmarshal error")
		restore := unmarshal
		unmarshal = func(buf []byte, pb proto.Message) error { return errExpected }
		defer func() { unmarshal = restore }()

		b := mocks.NewBlockBuilder(channelID, 1100)
		b.Transaction(txID1, pb.TxValidationCode_VALID)
		block := b.Build()

		err := vHalt.Visit(b.Build(), mockPvtData(block.Header.Number, 0))
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.NoError(t, v.Visit(block, pvtData))

		vNoHalt := New(channelID)
		require.NoError(t, vNoHalt.Visit(b.Build(), mockPvtData(block.Header.Number, 0)))
	})
}

func TestVisitor_NoStopOnError(t *testing.T) {
	expectedErr := fmt.Errorf("injected error")
	numReads := 0
	numWrites := 0
	numLSCCWrites := 0
	numCCEvents := 0

	p := New(channelID,
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

	block, pvtData := mockBlockWithTransactions(t)

	require.NoError(t, p.Visit(block, pvtData))
	assert.Equal(t, 3, numReads)  // 2 state and 1 private collection reads
	assert.Equal(t, 5, numWrites) // 3 state and 2 private collection writes
	assert.Equal(t, 2, numCCEvents)
	assert.EqualValues(t, 1101, p.LedgerHeight())
}

func TestVisitor_PvtDataNoStopOnError(t *testing.T) {
	expectedErr := fmt.Errorf("injected error")
	numReads := 0
	numWrites := 0

	p := New(channelID,
		WithReadHandler(func(read *Read) error {
			numReads++
			return expectedErr
		}),
		WithWriteHandler(func(write *Write) error {
			numWrites++
			return expectedErr
		}),
	)
	require.NotNil(t, p)

	b := mocks.NewBlockBuilder(channelID, 1100)
	b.Transaction(txID1, pb.TxValidationCode_VALID).ChaincodeAction(ccID1)
	block := b.Build()
	pvtData := mockPvtData(block.Header.Number, 0)

	require.NoError(t, p.Visit(block, pvtData))
	assert.Equal(t, 1, numReads)  // 1 private collection reads
	assert.Equal(t, 2, numWrites) // 2 private collection writes
	assert.EqualValues(t, 1101, p.LedgerHeight())
}

func TestVisitor_PvtDataStopOnHandlerError(t *testing.T) {
	expectedErr := fmt.Errorf("injected error")

	t.Run("Read handler error", func(t *testing.T) {
		numReads := 0
		numWrites := 0
		var category Category

		p := New(channelID,
			WithReadHandler(func(read *Read) error {
				numReads++
				return expectedErr
			}),
			WithWriteHandler(func(write *Write) error {
				numWrites++
				return nil
			}),
			WithErrorHandler(func(err error, ctx *Context) error {
				category = ctx.Category
				return err
			}),
		)
		require.NotNil(t, p)

		b := mocks.NewBlockBuilder(channelID, 1100)
		b.Transaction(txID1, pb.TxValidationCode_VALID).ChaincodeAction(ccID1)
		block := b.Build()
		pvtData := mockPvtData(block.Header.Number, 0)

		err := p.Visit(block, pvtData)
		require.Error(t, err)
		require.Contains(t, err.Error(), expectedErr.Error())
		require.Equal(t, 1, numReads)
		require.Equal(t, 0, numWrites)
		require.Equal(t, ReadHandlerErr, category)
	})

	t.Run("Write handler error", func(t *testing.T) {
		numReads := 0
		numWrites := 0
		var category Category

		p := New(channelID,
			WithReadHandler(func(read *Read) error {
				numReads++
				return nil
			}),
			WithWriteHandler(func(write *Write) error {
				numWrites++
				return expectedErr
			}),
			WithErrorHandler(func(err error, ctx *Context) error {
				category = ctx.Category
				return err
			}),
		)
		require.NotNil(t, p)

		b := mocks.NewBlockBuilder(channelID, 1100)
		b.Transaction(txID1, pb.TxValidationCode_VALID).ChaincodeAction(ccID1)
		block := b.Build()
		pvtData := mockPvtData(block.Header.Number, 0)

		err := p.Visit(block, pvtData)
		require.Error(t, err)
		require.Contains(t, err.Error(), expectedErr.Error())
		require.Equal(t, 1, numReads)
		require.Equal(t, 1, numWrites)
		require.Equal(t, WriteHandlerErr, category)
	})
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

func mockBlockWithTransactions(t *testing.T) (*cb.Block, ledger.TxPvtDataMap) {
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
		HashedRead(hashedKey1, v1).
		HashedWrite(hashedKey1, hashedValue1)
	cc2_1.Collection(coll2).
		Delete(hashedKey1)

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

	b.Transaction(txID4, pb.TxValidationCode_VALID).
		ChaincodeAction(LsccID).
		Write(ccID1, ccDataBytes)

	tb4 := b.Transaction(txID5, pb.TxValidationCode_VALID)
	tb4.ChaincodeAction(LsccID).
		Write(ccID1, ccDataBytes).
		ChaincodeEvent(ccEvent1, nil)

	block := b.Build()

	return block, mockPvtData(block.Header.Number, 0)
}

func TestVisitorError_Cause(t *testing.T) {
	cause := errors.New("cause of the error")
	err := newVisitorError(cause)
	require.NotNil(t, err)
	require.Equal(t, cause, errors.Cause(err))
}

func mockPvtData(blockNum, txIdx uint64) ledger.TxPvtDataMap {
	pvtBuilder := mocks.NewPvtReadWriteSetBuilder()
	pvtNsBuilder := pvtBuilder.Namespace(ccID1)
	pvtNsBuilder.Collection(coll1).Write(key1, []byte("value1")).Read(key2, blockNum, txIdx)
	pvtNsBuilder.Collection(coll2).Write(key2, []byte("value2"))

	pvtData := &ledger.TxPvtData{
		SeqInBlock: txIdx,
		WriteSet:   pvtBuilder.BuildReadWriteSet(),
	}

	return ledger.TxPvtDataMap{0: pvtData}
}
