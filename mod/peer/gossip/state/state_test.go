/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package state

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	gproto "github.com/hyperledger/fabric-protos-go/gossip"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/peer/mock"
	"github.com/hyperledger/fabric/extensions/gossip/api"
	gmocks "github.com/hyperledger/fabric/extensions/gossip/mocks"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/protoext"
	"github.com/hyperledger/fabric/gossip/util"
	"github.com/pkg/errors"
	viper "github.com/spf13/viper2015"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/gossip/blockpublisher"
	"github.com/trustbloc/fabric-peer-ext/pkg/gossip/state"
	statemocks "github.com/trustbloc/fabric-peer-ext/pkg/gossip/state/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

//go:generate counterfeiter -o ../mocks/gossipmediator.gen.go --fake-name GossipServiceMediator . GossipServiceMediator

// Ensure roles are initialized
var _ = roles.GetRoles()

func TestGossipStateProviderExtension_AddPayload(t *testing.T) {
	resetViper := setViper(config.ConfDistributedValidationEnabled, true)
	defer resetViper()

	vp := &gmocks.ValidationProvider{}
	cp := &gmocks.ValidationContextProvider{}

	mgr := InitValidationMgr(vp, cp)
	require.NotNil(t, mgr)

	extension := NewGossipStateProviderExtension("test", &gmocks.GossipServiceMediator{}, &api.Support{}, false)

	block := mocks.NewBlockBuilder(channelID, 1000).Build()

	data, err := proto.Marshal(block)
	require.NoError(t, err)

	payload := &gproto.Payload{
		SeqNum: 1000,
		Data:   data,
	}

	t.Run("Committer", func(t *testing.T) {
		reset := roles.SetRole(roles.CommitterRole)
		defer reset()

		handleAddPayload := func(payload *gproto.Payload, blockingMode bool) error {
			return nil
		}

		require.NoError(t, extension.AddPayload(handleAddPayload)(payload, false))
	})

	t.Run("Non-committer", func(t *testing.T) {
		reset := roles.SetRole(roles.EndorserRole)
		defer reset()

		handleAddPayload := func(payload *gproto.Payload, blockingMode bool) error {
			return nil
		}

		t.Run("Block validated", func(t *testing.T) {
			require.NoError(t, extension.AddPayload(handleAddPayload)(payload, false))
		})

		t.Run("Block not validated", func(t *testing.T) {
			bb := mocks.NewBlockBuilder(channelID, 1000)
			bb.Transaction("tx1", peer.TxValidationCode_NOT_VALIDATED)
			block := bb.Build()

			data, err := proto.Marshal(block)
			require.NoError(t, err)

			payload := &gproto.Payload{
				SeqNum: 1000,
				Data:   data,
			}

			require.NoError(t, extension.AddPayload(handleAddPayload)(payload, false))
		})
	})

	t.Run("Error", func(t *testing.T) {
		reset := roles.SetRole(roles.CommitterRole)
		defer reset()

		handleAddPayload := func(payload *gproto.Payload, blockingMode bool) error {
			return nil
		}

		t.Run("Handler error", func(t *testing.T) {
			sampleError := errors.New("not implemented")

			handleAddPayload := func(payload *gproto.Payload, blockingMode bool) error {
				return sampleError
			}

			require.EqualError(t, extension.AddPayload(handleAddPayload)(payload, false), sampleError.Error())
		})

		t.Run("Nil payload", func(t *testing.T) {
			require.EqualError(t, extension.AddPayload(handleAddPayload)(nil, false), "payload is nil")
		})

		t.Run("Unmarshal error", func(t *testing.T) {
			payload := &gproto.Payload{
				SeqNum: 1000,
				Data:   []byte("invalid"),
			}

			require.EqualError(t, extension.AddPayload(handleAddPayload)(payload, false), "error unmarshalling block with seqNum 1000: unexpected EOF")
		})

		t.Run("Invalid block header", func(t *testing.T) {
			payload := &gproto.Payload{
				SeqNum: 1000,
			}

			require.EqualError(t, extension.AddPayload(handleAddPayload)(payload, false), "block with sequence 1000 has no header or data")
		})
	})
}

func TestGossipStateProviderExtension_StoreBlock(t *testing.T) {
	resetViper := setViper(config.ConfDistributedValidationEnabled, true)
	defer resetViper()

	ccEvtMgr := &statemocks.CCEventMgr{}
	ccEvtMgrProvider := &statemocks.CCEventMgrProvider{}
	ccEvtMgrProvider.GetMgrReturns(ccEvtMgr)
	handlerRegistry := &statemocks.AppDataHandlerRegistry{}
	bpp := blockpublisher.NewProvider()

	state.NewUpdateHandler(&state.Providers{BPProvider: bpp, MgrProvider: ccEvtMgrProvider, HandlerRegistry: handlerRegistry})

	vp := &gmocks.ValidationProvider{}
	cp := &gmocks.ValidationContextProvider{}

	mgr := InitValidationMgr(vp, cp)
	require.NotNil(t, mgr)

	extension := NewGossipStateProviderExtension("test", &gmocks.GossipServiceMediator{}, &api.Support{
		Ledger: &mock.PeerLedger{},
	}, false)

	t.Run("Committer", func(t *testing.T) {
		reset := roles.SetRole(roles.CommitterRole)
		defer reset()

		t.Run("Nil block", func(t *testing.T) {
			handleStoreBlock := func(block *common.Block, pvtData util.PvtDataCollections) (*ledger.BlockAndPvtData, error) {
				return nil, nil
			}

			_, err := extension.StoreBlock(handleStoreBlock)(mocks.NewBlockBuilder(channelID, 1000).Build(), nil)
			require.NoError(t, err)
		})

		t.Run("Non-nil block", func(t *testing.T) {
			handleStoreBlock := func(block *common.Block, pvtData util.PvtDataCollections) (*ledger.BlockAndPvtData, error) {
				return &ledger.BlockAndPvtData{
					Block: block,
				}, nil
			}

			_, err := extension.StoreBlock(handleStoreBlock)(mocks.NewBlockBuilder(channelID, 1000).Build(), nil)
			require.NoError(t, err)
		})
	})

	t.Run("Non-committer", func(t *testing.T) {
		reset := roles.SetRole(roles.EndorserRole, roles.ValidatorRole)
		defer reset()

		t.Run("Block validated", func(t *testing.T) {
			handleStoreBlock := func(block *common.Block, pvtData util.PvtDataCollections) (*ledger.BlockAndPvtData, error) {
				return &ledger.BlockAndPvtData{
					Block: block,
				}, nil
			}

			_, err := extension.StoreBlock(handleStoreBlock)(mocks.NewBlockBuilder(channelID, 1000).Build(), nil)
			require.NoError(t, err)
		})

		t.Run("Block not validated", func(t *testing.T) {
			bb := mocks.NewBlockBuilder(channelID, 1000)
			bb.Transaction("tx1", peer.TxValidationCode_NOT_VALIDATED)
			block := bb.Build()

			handleStoreBlock := func(block *common.Block, pvtData util.PvtDataCollections) (*ledger.BlockAndPvtData, error) {
				return nil, nil
			}

			_, err := extension.StoreBlock(handleStoreBlock)(block, nil)
			require.NoError(t, err)
		})
	})

	t.Run("Error", func(t *testing.T) {
		reset := roles.SetRole(roles.CommitterRole)
		defer reset()

		t.Run("Handler error", func(t *testing.T) {
			sampleError := errors.New("not implemented")

			handleStoreBlock := func(block *common.Block, pvtData util.PvtDataCollections) (*ledger.BlockAndPvtData, error) {
				return nil, sampleError
			}

			_, err := extension.StoreBlock(handleStoreBlock)(mocks.NewBlockBuilder(channelID, 1000).Build(), nil)
			require.EqualError(t, err, sampleError.Error())
		})
	})
}

func TestProviderExtension(t *testing.T) {
	ccEvtMgr := &statemocks.CCEventMgr{}
	ccEvtMgrProvider := &statemocks.CCEventMgrProvider{}
	ccEvtMgrProvider.GetMgrReturns(ccEvtMgr)
	handlerRegistry := &statemocks.AppDataHandlerRegistry{}
	bpp := blockpublisher.NewProvider()

	state.NewUpdateHandler(&state.Providers{BPProvider: bpp, MgrProvider: ccEvtMgrProvider, HandlerRegistry: handlerRegistry})

	vp := &gmocks.ValidationProvider{}
	cp := &gmocks.ValidationContextProvider{}

	mgr := InitValidationMgr(vp, cp)
	require.NotNil(t, mgr)

	reset := roles.SetRole(roles.CommitterRole, roles.EndorserRole)
	defer reset()

	sampleError := errors.New("not implemented")

	handleAddPayload := func(payload *gproto.Payload, blockingMode bool) error {
		return sampleError
	}

	handleStoreBlock := func(block *common.Block, pvtData util.PvtDataCollections) (*ledger.BlockAndPvtData, error) {
		return nil, sampleError
	}

	extension := NewGossipStateProviderExtension("test", nil, &api.Support{
		Ledger: &mock.PeerLedger{},
	}, false)

	payload := &gproto.Payload{
		SeqNum: 1000,
	}

	//test extension.AddPayload
	require.Error(t, sampleError, extension.AddPayload(handleAddPayload)(payload, false))

	//test extension.StoreBlock
	_, err := extension.StoreBlock(handleStoreBlock)(mocks.NewBlockBuilder(channelID, 1000).Build(), nil)
	require.EqualError(t, err, sampleError.Error())

	//test extension.AntiEntropy
	handled := make(chan bool, 1)

	//test extension.HandleStateRequest
	handleStateRequest := func(msg protoext.ReceivedMessage) {
		handled <- true
	}

	extension.HandleStateRequest(handleStateRequest)(nil)

	select {
	case ok := <-handled:
		require.True(t, ok)
		break
	default:
		require.Fail(t, "state request handle should get called in case of all roles")
	}

	//test extension.RequestBlocksInRange
	handleRequestBlocksInRange := func(start uint64, end uint64) {
		handled <- true
	}

	addPayload := func(payload *gproto.Payload, blockingMode bool) error {
		handled <- false
		return nil
	}

	extension.RequestBlocksInRange(handleRequestBlocksInRange, addPayload)(1, 2)

	select {
	case ok := <-handled:
		require.True(t, ok)
		break
	default:
		require.Fail(t, "RequestBlocksInRange should get called in case of all roles")
	}
}

func TestProviderByEndorser(t *testing.T) {

	//make sure roles is endorser not committer
	if roles.IsCommitter() {
		rolesValue := make(map[roles.Role]struct{})
		rolesValue[roles.EndorserRole] = struct{}{}
		roles.SetRoles(rolesValue)
		defer func() { roles.SetRoles(nil) }()
	}
	require.False(t, roles.IsCommitter())
	require.True(t, roles.IsEndorser())

	sampleError := errors.New("not implemented")

	handleAddPayload := func(payload *gproto.Payload, blockingMode bool) error {
		return sampleError
	}

	handleStoreBlock := func(block *common.Block, pvtData util.PvtDataCollections) (*ledger.BlockAndPvtData, error) {
		return nil, sampleError
	}

	extension := NewGossipStateProviderExtension("test", nil, &api.Support{
		Ledger: &mockPeerLedger{&mocks.Ledger{}},
	}, false)

	payload := &gproto.Payload{
		SeqNum: 1000,
	}

	block := &common.Block{
		Header: &common.BlockHeader{
			Number: 1000,
		},
	}

	//test extension.AddPayload
	require.Error(t, sampleError, extension.AddPayload(handleAddPayload)(payload, false))

	//test extension.StoreBlock
	_, err := extension.StoreBlock(handleStoreBlock)(block, util.PvtDataCollections{})
	require.NoError(t, err)

	//test extension.AntiEntropy
	handled := make(chan bool, 1)

	//test extension.HandleStateRequest
	handleStateRequest := func(msg protoext.ReceivedMessage) {
		handled <- true
	}

	extension.HandleStateRequest(handleStateRequest)(nil)

	select {
	case ok := <-handled:
		require.True(t, ok)
		break
	default:
		require.Fail(t, "state request handle should get called in case of endorsers")
	}

	//test extension.RequestBlocksInRange
	handleRequestBlocksInRange := func(start uint64, end uint64) {
		handled <- false
	}

	addPayload := func(payload *gproto.Payload, blockingMode bool) error {
		handled <- true
		return nil
	}

	extension.RequestBlocksInRange(handleRequestBlocksInRange, addPayload)(1, 1)

	select {
	case ok := <-handled:
		require.True(t, ok)
		break
	default:
		require.Fail(t, "RequestBlocksInRange should get called in case of all roles")
	}
}

func TestPredicate(t *testing.T) {
	predicate := func(peer discovery.NetworkMember) bool {
		return true
	}
	extension := NewGossipStateProviderExtension("test", nil, &api.Support{}, false)
	require.True(t, extension.Predicate(predicate)(discovery.NetworkMember{Properties: &gproto.Properties{Roles: []string{"endorser"}}}))
	require.False(t, extension.Predicate(predicate)(discovery.NetworkMember{Properties: &gproto.Properties{Roles: []string{"committer"}}}))
	require.True(t, extension.Predicate(predicate)(discovery.NetworkMember{Properties: &gproto.Properties{Roles: []string{}}}))
	require.False(t, extension.Predicate(predicate)(discovery.NetworkMember{Properties: &gproto.Properties{Roles: []string{""}}}))
}

func TestProviderByCommitter(t *testing.T) {

	//make sure roles is committer not endorser
	if roles.IsEndorser() {
		rolesValue := make(map[roles.Role]struct{})
		rolesValue[roles.CommitterRole] = struct{}{}
		roles.SetRoles(rolesValue)
		defer func() { roles.SetRoles(nil) }()
	}
	require.True(t, roles.IsCommitter())
	require.False(t, roles.IsEndorser())

	sampleError := errors.New("not implemented")

	handleAddPayload := func(payload *gproto.Payload, blockingMode bool) error {
		return sampleError
	}

	handleStoreBlock := func(block *common.Block, pvtData util.PvtDataCollections) (*ledger.BlockAndPvtData, error) {
		return nil, sampleError
	}

	extension := NewGossipStateProviderExtension("test", nil, &api.Support{}, false)

	payload := &gproto.Payload{
		SeqNum: 1000,
	}

	//test extension.AddPayload
	require.Error(t, sampleError, extension.AddPayload(handleAddPayload)(payload, false))

	//test extension.StoreBlock
	_, err := extension.StoreBlock(handleStoreBlock)(nil, nil)
	require.Error(t, err, sampleError)

	//test extension.AntiEntropy
	handled := make(chan bool, 1)

	//test extension.HandleStateRequest
	handleStateRequest := func(msg protoext.ReceivedMessage) {
		handled <- true
	}

	extension.HandleStateRequest(handleStateRequest)(nil)

	select {
	case <-handled:
		require.Fail(t, "state request handle shouldn't get called in case of committer")
	default:
		//do nothing
	}

	//test extension.RequestBlocksInRange
	handleRequestBlocksInRange := func(start uint64, end uint64) {
		handled <- true
	}

	addPayload := func(payload *gproto.Payload, blockingMode bool) error {
		handled <- false
		return nil
	}

	extension.RequestBlocksInRange(handleRequestBlocksInRange, addPayload)(1, 2)

	select {
	case ok := <-handled:
		require.True(t, ok)
		break
	default:
		require.Fail(t, "RequestBlocksInRange should get called in case of committer")
	}
}

//mockPeerLedger mocks peerledger and support GetBlockByNumber feature
type mockPeerLedger struct {
	*mocks.Ledger
}

// GetBlockByNumber returns the block by number
func (m *mockPeerLedger) GetBlockByNumber(blockNumber uint64) (*common.Block, error) {
	b := mocks.NewBlockBuilder("test", blockNumber)
	return b.Build(), nil
}

func setViper(key string, value interface{}) (reset func()) {
	oldVal := viper.Get(key)
	viper.Set(key, value)

	return func() { viper.Set(key, oldVal) }
}
