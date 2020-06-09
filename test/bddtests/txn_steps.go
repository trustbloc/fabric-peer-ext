/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"crypto"
	"encoding/base64"
	"encoding/hex"
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
	return d.doQueryTxnService(channelID, ccID, args, peerIDs)
}

func (d *TxnSteps) queryTxnServiceWithExpectedError(channelID, ccID, args, peerIDs, expectedError string) error {
	err := d.doQueryTxnService(channelID, ccID, args, peerIDs)
	if err == nil {
		return errors.Errorf("expecting error [%s] but got none", expectedError)
	}

	if strings.Contains(err.Error(), expectedError) {
		logger.Infof("Got error [%s] which was expected to contain [%s]", err, expectedError)

		return nil
	}

	return errors.Errorf("expecting error to contain [%s] but got [%s]", expectedError, err.Error())
}

func (d *TxnSteps) assignBase64URLEncodedValue(varName, value string) error {
	bddtests.SetVar(varName, base64.URLEncoding.EncodeToString([]byte(value)))

	return nil
}

func (d *TxnSteps) doQueryTxnService(channelID, ccID, args, peerIDs string) error {
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
		logger.Infof("Got error response: %s", resp.Payload)
		return fmt.Errorf("QueryChaincode return error: %s", err)
	}

	bddtests.SetResponse(string(resp.Payload))

	logger.Infof("Got response: %s", resp.Payload)

	return nil
}

func (d *TxnSteps) saveComputedTxnIDToVar(varName, identity, nonce string) error {
	resolved, err := bddtests.ResolveVars(identity)
	if err != nil {
		return err
	}

	identity = resolved.(string)

	resolved, err = bddtests.ResolveVars(nonce)
	if err != nil {
		return err
	}

	nonce = resolved.(string)

	c, err := base64.URLEncoding.DecodeString(identity)
	if err != nil {
		return err
	}

	n, err := base64.URLEncoding.DecodeString(nonce)
	if err != nil {
		return err
	}

	txnID, err := computeTxnID(c, n)
	if err != nil {
		return err
	}

	bddtests.SetVar(varName, txnID)

	return nil
}

func computeTxnID(creator, nonce []byte) (string, error) {
	hash := crypto.SHA256.New()

	b := append(nonce, creator...)

	_, err := hash.Write(b)
	if err != nil {
		return "", errors.WithMessagef(err, "hashing of nonce and creator failed")
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// RegisterSteps registers off-ledger steps
func (d *TxnSteps) RegisterSteps(s *godog.Suite) {
	s.BeforeScenario(d.BDDContext.BeforeScenario)
	s.AfterScenario(d.BDDContext.AfterScenario)

	s.Step(`^txn service is invoked on channel "([^"]*)" with chaincode "([^"]*)" with args "([^"]*)" on peers "([^"]*)"$`, d.queryTxnService)
	s.Step(`^txn service is invoked on channel "([^"]*)" with chaincode "([^"]*)" with args "([^"]*)" on peers "([^"]*)" then the error response should contain "([^"]*)"$`, d.queryTxnServiceWithExpectedError)
	s.Step(`^variable "([^"]*)" is assigned the base64 URL-encoded value "([^"]*)"$`, d.assignBase64URLEncodedValue)
	s.Step(`^variable "([^"]*)" is computed from the identity "([^"]*)" and nonce "([^"]*)"$`, d.saveComputedTxnIDToVar)
}
