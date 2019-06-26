/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package channel

import (
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"

	"github.com/hyperledger/fabric/common/configtx/test"

	"github.com/hyperledger/fabric/core/committer/txvalidator/plugin"
	"github.com/hyperledger/fabric/core/ledger/ledgermgmt"

	"github.com/trustbloc/fabric-peer-ext/pkg/roles"

	"github.com/hyperledger/fabric/core/committer/txvalidator/v20/plugindispatcher"
	"github.com/hyperledger/fabric/core/common/sysccprovider"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/protos/common"

	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/stretchr/testify/require"
)

func TestJoinChainHandlerAsCommitter(t *testing.T) {

	sampelResponse := pb.Response{Message: "sample-test-msg"}
	handle := func(string, *common.Block, sysccprovider.SystemChaincodeProvider,
		ledger.DeployedChaincodeInfoProvider, plugindispatcher.LifecycleResources, plugindispatcher.CollectionAndLifecycleResources) pb.Response {
		return sampelResponse
	}

	//make sure roles is committer not endorser
	if roles.IsEndorser() {
		rolesValue := make(map[roles.Role]struct{})
		rolesValue[roles.CommitterRole] = struct{}{}
		roles.SetRoles(rolesValue)
		defer func() { roles.SetRoles(nil) }()
	}
	require.True(t, roles.IsCommitter())
	require.False(t, roles.IsEndorser())

	response := JoinChainHandler(handle)("", nil, nil, nil, nil, nil)
	require.Equal(t, sampelResponse, response)

}

func TestJoinChainHandlerAsEndorser(t *testing.T) {

	const cid = "sample-cid"
	handle := func(string, *common.Block, sysccprovider.SystemChaincodeProvider,
		ledger.DeployedChaincodeInfoProvider, plugindispatcher.LifecycleResources, plugindispatcher.CollectionAndLifecycleResources) pb.Response {
		panic("regular handle not supposed to be called in case of endorser")
	}

	cChain := func(string, ledger.PeerLedger, *common.Block, sysccprovider.SystemChaincodeProvider, plugin.Mapper,
		ledger.DeployedChaincodeInfoProvider, plugindispatcher.LifecycleResources, plugindispatcher.CollectionAndLifecycleResources) error {
		return nil
	}

	iChain := func(string) {}

	cfgBlk := func(ledger ledger.PeerLedger) (*common.Block, error) {
		return nil, nil
	}
	//register handlers for create chain and init chain
	RegisterChannelInitializer(nil, cChain, iChain, cfgBlk)

	cleanup := ledgermgmt.InitializeTestEnv(t)
	defer cleanup()

	gb, _ := test.MakeGenesisBlock(cid)
	ledger, _ := ledgermgmt.CreateLedger(gb)
	ledger.Close()

	//make sure roles is committer not endorser
	if roles.IsCommitter() {
		rolesValue := make(map[roles.Role]struct{})
		rolesValue[roles.EndorserRole] = struct{}{}
		roles.SetRoles(rolesValue)
		defer func() { roles.SetRoles(nil) }()
	}
	require.True(t, roles.IsEndorser())
	require.False(t, roles.IsCommitter())

	response := JoinChainHandler(handle)(cid, nil, nil, nil, nil, nil)
	require.Equal(t, shim.Success(nil), response)

}
