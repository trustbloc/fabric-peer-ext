/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package channel

import (
	"fmt"
	"sync"

	"github.com/hyperledger/fabric/common/util/retry"
	"github.com/hyperledger/fabric/core/ledger/ledgermgmt"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/committer/txvalidator/plugin"
	"github.com/hyperledger/fabric/core/committer/txvalidator/v20/plugindispatcher"
	"github.com/hyperledger/fabric/core/common/sysccprovider"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/extensions/roles"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
)

var logger = flogging.MustGetLogger("extension-channel")

const maxAttempts = 10

var once sync.Once

var createChain CreateChain

var pluginMapper plugin.Mapper

var initChain func(string)

//JoinChain is a handle function for cscc join chain
type JoinChain func(string, *common.Block, sysccprovider.SystemChaincodeProvider, ledger.DeployedChaincodeInfoProvider, plugindispatcher.LifecycleResources, plugindispatcher.CollectionAndLifecycleResources) pb.Response

//CreateChain a handle function used by initialize channel feature
type CreateChain func(string, ledger.PeerLedger, *common.Block, sysccprovider.SystemChaincodeProvider, plugin.Mapper,
	ledger.DeployedChaincodeInfoProvider, plugindispatcher.LifecycleResources, plugindispatcher.CollectionAndLifecycleResources) error

//JoinChainHandler can be used to provide extended features to CSCC join chain
func JoinChainHandler(handle JoinChain) JoinChain {

	if roles.IsCommitter() {
		return handle
	}

	return func(chainID string, block *common.Block, sccp sysccprovider.SystemChaincodeProvider, deployedCCInfoProvider ledger.DeployedChaincodeInfoProvider, lr plugindispatcher.LifecycleResources, nr plugindispatcher.CollectionAndLifecycleResources) pb.Response {
		logger.Debugf("Not a committer - initializing channel [%s]...", chainID)

		if err := initializeChannelWithRetry(chainID, block, sccp, deployedCCInfoProvider, lr, nr); err != nil {
			logger.Errorf("Error initializing channel [%s]: %s", chainID, err)
			return shim.Error(fmt.Sprintf("Error initializing channel [%s] : [%s]", chainID, err))
		}

		logger.Debugf("... successfully initializing channel [%s].", chainID)
		return shim.Success(nil)
	}
}

//RegisterChannelInitializer registers channel initializer using get block handle and create chain handle
func RegisterChannelInitializer(pm plugin.Mapper, cChain CreateChain, iChain func(string)) {
	once.Do(func() {
		createChain = cChain
		initChain = iChain
		pluginMapper = pm
		return
	})
	logger.Warn("Plugin mappers and createChain handler cannot be replaced")
}

//initializeChannelWithRetry perform channel initialization with retry
func initializeChannelWithRetry(cid string, block *common.Block, sccp sysccprovider.SystemChaincodeProvider,
	deployedCCInfoProvider ledger.DeployedChaincodeInfoProvider, lr plugindispatcher.LifecycleResources, nr plugindispatcher.CollectionAndLifecycleResources) error {

	initializeChannel := func() error {
		logger.Infof("Loading channel [%s]", cid)

		var err error
		var ledger ledger.PeerLedger
		if ledger, err = ledgermgmt.OpenLedger(cid); err != nil {
			logger.Warningf("Failed to load ledger %s(%s)", cid, err)
			logger.Debugf("Error while loading ledger %s with message %s. We continue to the next ledger rather than abort.", cid, err)
			return err
		}

		// Create a chain if we get a valid ledger with config block
		if err = createChain(cid, ledger, block, sccp, pluginMapper, deployedCCInfoProvider, lr, nr); err != nil {
			logger.Warningf("Failed to load chain %s(%s)", cid, err)
			logger.Debugf("Error reloading chain %s with message %s. We continue to the next chain rather than abort.", cid, err)
			return err
		}

		initChain(cid)

		return nil
	}

	_, err := retry.Invoke(
		func() (interface{}, error) {
			err := initializeChannel()
			return nil, err
		},
		retry.WithMaxAttempts(maxAttempts),
	)
	return err
}
