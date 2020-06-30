/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cscc

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric/core/committer/txvalidator/v20/plugindispatcher"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/ledgermgmt"
	"github.com/hyperledger/fabric/core/ledger/ledgermgmt/ledgermgmttest"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/extensions/testutil"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

const (
	channelID = "testchannel"
)

func TestJoinChainHandler(t *testing.T) {
	_, _, destroy := testutil.SetupExtTestEnv()
	defer destroy()

	testDir, err := ioutil.TempDir("", "cscc_test")
	require.NoError(t, err, "error in creating test dir")
	defer os.Remove(testDir)

	ledgerInitializer := ledgermgmttest.NewInitializer(testDir)
	ledgerInitializer.CustomTxProcessors = map[common.HeaderType]ledger.CustomTxProcessor{
		common.HeaderType_CONFIG: &peer.ConfigTxProcessor{},
	}
	ledgerMgr := ledgermgmt.NewLedgerMgr(ledgerInitializer)
	defer ledgerMgr.Close()

	//make sure roles is committer not endorser
	if roles.IsCommitter() {
		rolesValue := make(map[roles.Role]struct{})
		rolesValue[roles.EndorserRole] = struct{}{}
		roles.SetRoles(rolesValue)
		defer func() { roles.SetRoles(nil) }()
	}
	require.True(t, roles.IsEndorser())
	require.False(t, roles.IsCommitter())

	p := New(nil, nil, nil, nil, nil, nil, nil)
	require.NotNil(t, p)

	t.Run("Success", func(t *testing.T) {
		initChannelInvoked := false
		p.handleInitializeChannel = func(cid string,
			deployedCCInfoProvider ledger.DeployedChaincodeInfoProvider,
			lr plugindispatcher.LifecycleResources,
			nr plugindispatcher.CollectionAndLifecycleResources,
		) error {
			initChannelInvoked = true
			return nil
		}

		response := p.joinChain(channelID, nil, nil, nil, nil)
		require.Equal(t, shim.Success(nil), response)
		require.True(t, initChannelInvoked)
	})

	t.Run("Error", func(t *testing.T) {
		errExpected := errors.New("injected error")
		p.handleInitializeChannel = func(cid string,
			deployedCCInfoProvider ledger.DeployedChaincodeInfoProvider,
			lr plugindispatcher.LifecycleResources,
			nr plugindispatcher.CollectionAndLifecycleResources,
		) error {
			return errExpected
		}

		response := p.joinChain(channelID, nil, nil, nil, nil)
		require.Equal(t, shim.Error(fmt.Sprintf("Error initializing channel [%s] : [%s]", channelID, errExpected.Error())), response)
	})
}
