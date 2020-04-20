/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package state

import (
	"testing"

	"github.com/hyperledger/fabric-protos-go/common"
	proto "github.com/hyperledger/fabric-protos-go/gossip"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/protoext"
	"github.com/hyperledger/fabric/gossip/util"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric/extensions/gossip/api"

	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

func TestProviderExtension(t *testing.T) {
	//all roles
	rolesValue := make(map[roles.Role]struct{})
	roles.SetRoles(rolesValue)
	defer func() { roles.SetRoles(nil) }()

	sampleError := errors.New("not implemented")

	handleAddPayload := func(payload *proto.Payload, blockingMode bool) error {
		return sampleError
	}

	handleStoreBlock := func(block *common.Block, pvtData util.PvtDataCollections) error {
		return sampleError
	}

	extension := NewGossipStateProviderExtension("test", nil, &api.Support{}, false)

	payload := &proto.Payload{
		SeqNum: 1000,
	}

	//test extension.AddPayload
	require.Error(t, sampleError, extension.AddPayload(handleAddPayload)(payload, false))

	//test extension.StoreBlock
	require.Error(t, sampleError, extension.StoreBlock(handleStoreBlock)(nil, util.PvtDataCollections{}))

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

	addPayload := func(payload *proto.Payload, blockingMode bool) error {
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

	handleAddPayload := func(payload *proto.Payload, blockingMode bool) error {
		return sampleError
	}

	handleStoreBlock := func(block *common.Block, pvtData util.PvtDataCollections) error {
		return sampleError
	}

	extension := NewGossipStateProviderExtension("test", nil, &api.Support{
		Ledger: &mockPeerLedger{&mocks.Ledger{}},
	}, false)

	payload := &proto.Payload{
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
	require.Nil(t, extension.StoreBlock(handleStoreBlock)(block, util.PvtDataCollections{}))

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

	addPayload := func(payload *proto.Payload, blockingMode bool) error {
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
	require.True(t, extension.Predicate(predicate)(discovery.NetworkMember{Properties: &proto.Properties{Roles: []string{"endorser"}}}))
	require.False(t, extension.Predicate(predicate)(discovery.NetworkMember{Properties: &proto.Properties{Roles: []string{"committer"}}}))
	require.True(t, extension.Predicate(predicate)(discovery.NetworkMember{Properties: &proto.Properties{Roles: []string{}}}))
	require.False(t, extension.Predicate(predicate)(discovery.NetworkMember{Properties: &proto.Properties{Roles: []string{""}}}))
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

	handleAddPayload := func(payload *proto.Payload, blockingMode bool) error {
		return sampleError
	}

	handleStoreBlock := func(block *common.Block, pvtData util.PvtDataCollections) error {
		return sampleError
	}

	extension := NewGossipStateProviderExtension("test", nil, &api.Support{}, false)

	payload := &proto.Payload{
		SeqNum: 1000,
	}

	//test extension.AddPayload
	require.Error(t, sampleError, extension.AddPayload(handleAddPayload)(payload, false))

	//test extension.StoreBlock
	require.Error(t, sampleError, extension.StoreBlock(handleStoreBlock)(nil, util.PvtDataCollections{}))

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

	addPayload := func(payload *proto.Payload, blockingMode bool) error {
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
