/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cscc

import (
	"fmt"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/common"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/util/retry"
	"github.com/hyperledger/fabric/core/aclmgmt"
	"github.com/hyperledger/fabric/core/committer/txvalidator/v20/plugindispatcher"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/ledgermgmt"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/core/policy"
	"github.com/hyperledger/fabric/core/scc/cscc"
	"github.com/pkg/errors"

	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
)

var logger = flogging.MustGetLogger("ext_cscc")

const maxAttempts = 10

type channelInitializer func(cid string,
	deployedCCInfoProvider ledger.DeployedChaincodeInfoProvider,
	lr plugindispatcher.LifecycleResources,
	nr plugindispatcher.CollectionAndLifecycleResources,
) error

// New creates a new instance of the PeerConfiger.
func New(
	aclProvider aclmgmt.ACLProvider,
	deployedCCInfoProvider ledger.DeployedChaincodeInfoProvider,
	lr plugindispatcher.LifecycleResources,
	nr plugindispatcher.CollectionAndLifecycleResources,
	policyChecker policy.PolicyChecker,
	p *peer.Peer,
	bccsp bccsp.BCCSP,
) *PeerConfiger {
	pc := &PeerConfiger{p: p}

	var joinChannelHandler cscc.HandleJoinChannel
	if !roles.IsCommitter() {
		joinChannelHandler = pc.joinChain
	}

	pc.PeerConfiger = cscc.New(aclProvider, deployedCCInfoProvider, lr, nr, policyChecker, p, bccsp, joinChannelHandler)
	pc.handleInitializeChannel = pc.initializeChannel
	return pc
}

// PeerConfiger implements the configuration handler for the peer.
type PeerConfiger struct {
	*cscc.PeerConfiger
	p                       *peer.Peer
	handleJoinChannel       cscc.HandleJoinChannel
	handleInitializeChannel channelInitializer
}

func (c *PeerConfiger) joinChain(
	channelID string,
	_ *common.Block,
	deployedCCInfoProvider ledger.DeployedChaincodeInfoProvider,
	lr plugindispatcher.LifecycleResources,
	nr plugindispatcher.CollectionAndLifecycleResources,
) pb.Response {
	logger.Debugf("Initializing channel [%s]...", channelID)

	if err := c.initializeChannelWithRetry(channelID, deployedCCInfoProvider, lr, nr); err != nil {
		logger.Errorf("Error initializing channel [%s]: %s", channelID, err)
		return shim.Error(fmt.Sprintf("Error initializing channel [%s] : [%s]", channelID, err))
	}

	logger.Debugf("... successfully initialized channel [%s].", channelID)
	return shim.Success(nil)
}

//initializeChannelWithRetry perform channel initialization with retry
func (c *PeerConfiger) initializeChannelWithRetry(cid string, deployedCCInfoProvider ledger.DeployedChaincodeInfoProvider, lr plugindispatcher.LifecycleResources, nr plugindispatcher.CollectionAndLifecycleResources) error {
	_, err := retry.Invoke(
		func() (interface{}, error) {
			return nil, c.handleInitializeChannel(cid, deployedCCInfoProvider, lr, nr)
		},
		retry.WithMaxAttempts(maxAttempts),
		retry.WithBeforeRetry(func(err error, attempt int, backoff time.Duration) bool {
			logger.Infof("Got error initializing channel [%s] on attempt %d. Will retry in %s: %s", cid, attempt, backoff, err)
			return true
		}),
	)
	return err
}

func (c *PeerConfiger) initializeChannel(cid string,
	deployedCCInfoProvider ledger.DeployedChaincodeInfoProvider,
	lr plugindispatcher.LifecycleResources,
	nr plugindispatcher.CollectionAndLifecycleResources,
) error {
	err := c.p.InitializeChannel(cid, deployedCCInfoProvider, lr, nr)
	if err == nil {
		return nil
	}

	if errors.Cause(err) == ledgermgmt.ErrLedgerAlreadyOpened {
		logger.Infof("Ledger for channel [%s] is already open. Nothing else to do.", cid)
		return nil
	}

	return err
}
