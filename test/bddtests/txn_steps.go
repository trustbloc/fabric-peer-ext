/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"fmt"
	"strings"

	"github.com/cucumber/godog"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-test-common/bddtests"
)

// TxnSteps defines the transaction service steps
type TxnSteps struct {
	BDDContext *bddtests.BDDContext
	content    string
	address    string
}

// NewTxnSteps returns a new TxnSteps
func NewTxnSteps(context *bddtests.BDDContext) *TxnSteps {
	return &TxnSteps{BDDContext: context}
}

func (d *TxnSteps) queryTxnService(channelID, ccID, args, peerIDs string) error {
	bddtests.ClearResponse()

	if peerIDs == "" {
		return errors.New("no target peers specified")
	}

	var targetPeers []*bddtests.PeerConfig
	for _, id := range strings.Split(peerIDs, ",") {
		peer := d.BDDContext.PeerConfigForID(id)
		if peer == nil {
			return errors.Errorf("peer [%s] not found", id)
		}
		targetPeers = append(targetPeers, peer)
	}

	logger.Debugf("Querying peers [%s]...", targetPeers)

	argArr, err := bddtests.ResolveAllVars(args)
	if err != nil {
		return err
	}

	logger.Infof("Got args:")
	for i, arg := range argArr {
		logger.Infof("- arg[%d]: [%s]", i, arg)
	}

	var peers []fabApi.Peer
	var orgID string

	for _, target := range targetPeers {
		orgID = target.OrgID

		targetPeer, err := d.BDDContext.OrgUserContext(orgID, bddtests.ADMIN).InfraProvider().CreatePeerFromConfig(&fabApi.NetworkPeer{PeerConfig: target.Config})
		if err != nil {
			return errors.WithMessage(err, "NewPeer failed")
		}

		peers = append(peers, targetPeer)
	}

	chClient, err := d.BDDContext.OrgChannelClient(orgID, bddtests.ADMIN, channelID)
	if err != nil {
		logger.Errorf("Failed to create new channel client: %s", err)
		return errors.Wrap(err, "Failed to create new channel client")
	}

	retryOpts := retry.DefaultOpts
	retryOpts.RetryableCodes = retry.ChannelClientRetryableCodes

	systemHandlerChain := invoke.NewProposalProcessorHandler(
		invoke.NewEndorsementHandler(),
	)

	resp, err := chClient.InvokeHandler(
		systemHandlerChain, channel.Request{
			ChaincodeID: ccID,
			Fcn:         argArr[0],
			Args:        bddtests.GetByteArgs(argArr[1:]),
		},
		channel.WithTargets(peers...),
		channel.WithRetry(retryOpts),
	)
	if err != nil {
		return fmt.Errorf("QueryChaincode return error: %s", err)
	}

	bddtests.SetResponse(string(resp.Payload))

	logger.Infof("Got response: %s", resp.Payload)

	return nil
}

// RegisterSteps registers off-ledger steps
func (d *TxnSteps) RegisterSteps(s *godog.Suite) {
	s.BeforeScenario(d.BDDContext.BeforeScenario)
	s.AfterScenario(d.BDDContext.AfterScenario)
	s.Step(`^txn service is invoked on channel "([^"]*)" with chaincode "([^"]*)" with args "([^"]*)" on peers "([^"]*)"$`, d.queryTxnService)
}
