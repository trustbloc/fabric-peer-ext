/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"crypto"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/golang/protobuf/proto"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	contextApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	reqClientCtx "github.com/hyperledger/fabric-sdk-go/pkg/context"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/txn"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-test-common/bddtests"
)

const (
	userName = "User1"
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

	targetPeers, err := d.Peers(peerIDs)
	if err != nil {
		return err
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

		targetPeer, err := d.BDDContext.OrgUserContext(orgID, bddtests.USER).InfraProvider().CreatePeerFromConfig(&fabApi.NetworkPeer{PeerConfig: target.Config})
		if err != nil {
			return errors.WithMessage(err, "NewPeer failed")
		}

		peers = append(peers, targetPeer)
	}

	chClient, err := d.BDDContext.OrgChannelClient(orgID, bddtests.USER, channelID)
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

func (d *TxnSteps) saveSignedProposalToVar(ccID, strArgs, orgID, channelID, varName string) error {
	ctx, err := d.BDDContext.Sdk().ChannelContext(channelID, fabsdk.WithUser(userName), fabsdk.WithOrg(orgID))()
	if err != nil {
		return errors.Wrap(err, "Failed to create channel context")
	}

	args := bddtests.GetByteArgs(strings.Split(strArgs, ","))

	prop, err := createSignedProposal(ctx, ccID, string(args[0]), args[1:])
	if err != nil {
		return err
	}

	propBytes, err := json.Marshal(prop)
	if err != nil {
		return err
	}

	logger.Infof("Saving signed proposal to var [%s]: %s", varName, propBytes)

	bddtests.SetVar(varName, string(propBytes))

	return nil
}

func (d *TxnSteps) saveSignedProposalWithInvalidSignatureToVar(ccID, strArgs, orgID, channelID, varName string) error {
	ctx, err := d.BDDContext.Sdk().ChannelContext(channelID, fabsdk.WithUser(userName), fabsdk.WithOrg(orgID))()
	if err != nil {
		return errors.Wrap(err, "Failed to create channel context")
	}

	args := bddtests.GetByteArgs(strings.Split(strArgs, ","))

	prop, err := createSignedProposal(ctx, ccID, string(args[0]), args[1:])
	if err != nil {
		return err
	}

	prop.Signature = []byte("invalid")

	propBytes, err := json.Marshal(prop)
	if err != nil {
		return err
	}

	logger.Infof("Saving signed proposal to var [%s]: %s", varName, propBytes)

	bddtests.SetVar(varName, string(propBytes))

	return nil
}

func (d *TxnSteps) sendProposalAndSaveResponseToVar(proposal, peerIDs, varName string) error {
	return d.doSendProposalAndSaveResponseToVar(proposal, peerIDs, varName, false)
}

func (d *TxnSteps) sendProposalAndSaveInvalidResponseToVar(proposal, peerIDs, varName string) error {
	return d.doSendProposalAndSaveResponseToVar(proposal, peerIDs, varName, true)
}

func (d *TxnSteps) doSendProposalAndSaveResponseToVar(proposal, peerIDs, varName string, injectResponseErr bool) error {
	if peerIDs == "" {
		return errors.New("no target peers specified")
	}

	resolved, err := bddtests.ResolveVars(proposal)
	if err != nil {
		return err
	}

	proposal = resolved.(string)

	logger.Infof("Got signed proposal: %s", proposal)

	signedProposal := &pb.SignedProposal{}
	err = json.Unmarshal([]byte(proposal), signedProposal)
	if err != nil {
		return err
	}

	ctx := d.BDDContext.OrgUserContext("peerorg1", bddtests.USER)

	var peers []fabApi.Peer

	targets, err := d.Peers(peerIDs)
	if err != nil {
		return err
	}

	for _, target := range targets {
		targetPeer, err := ctx.InfraProvider().CreatePeerFromConfig(&fabApi.NetworkPeer{PeerConfig: target.Config})
		if err != nil {
			return err
		}

		peers = append(peers, targetPeer)
	}

	reqCtx, cancel := reqClientCtx.NewRequest(ctx, reqClientCtx.WithTimeout(30*time.Second))
	defer cancel()

	request := fabApi.ProcessProposalRequest{SignedProposal: signedProposal}

	var responses []*pb.ProposalResponse
	for _, p := range peers {
		logger.Infof("Sending proposal to target: [%s]", p.URL())

		response, err := p.ProcessTransactionProposal(reqCtx, request)
		if err != nil {
			return err
		}

		if injectResponseErr {
			response.ProposalResponse.Endorsement.Signature = []byte("invalid signature")
		}

		responses = append(responses, response.ProposalResponse)
	}

	respBytes, err := json.Marshal(responses)
	if err != nil {
		return err
	}

	logger.Infof("Saving proposal responses to var [%s]: %s", varName, respBytes)

	bddtests.SetVar(varName, string(respBytes))

	return nil
}

func (d *TxnSteps) Peers(peerIDs string) (bddtests.Peers, error) {
	var peers []*bddtests.PeerConfig
	for _, id := range strings.Split(peerIDs, ",") {
		peer := d.BDDContext.PeerConfigForID(id)
		if peer == nil {
			return nil, errors.Errorf("peer [%s] not found", id)
		}

		peers = append(peers, peer)
	}

	return peers, nil
}

func createSignedProposal(ctx contextApi.Channel, ccID string, fctn string, args [][]byte) (*pb.SignedProposal, error) {
	request := fabApi.ChaincodeInvokeRequest{
		ChaincodeID: ccID,
		Fcn:         fctn,
		Args:        args,
	}

	reqCtx, cancel := reqClientCtx.NewRequest(ctx)
	defer cancel()

	transactor, err := ctx.ChannelService().Transactor(reqCtx)
	if err != nil {
		return nil, err
	}

	txh, err := transactor.CreateTransactionHeader()
	if err != nil {
		return nil, errors.WithMessage(err, "creating transaction header failed")
	}

	prop, err := txn.CreateChaincodeInvokeProposal(txh, request)
	if err != nil {
		return nil, err
	}

	return signProposal(ctx, prop.Proposal)
}

func signProposal(ctx contextApi.Client, proposal *pb.Proposal) (*pb.SignedProposal, error) {
	proposalBytes, err := proto.Marshal(proposal)
	if err != nil {
		return nil, errors.Wrap(err, "marshal proposal failed")
	}

	signingMgr := ctx.SigningManager()
	if signingMgr == nil {
		return nil, errors.New("signing manager is nil")
	}

	signature, err := signingMgr.Sign(proposalBytes, ctx.PrivateKey())
	if err != nil {
		return nil, errors.WithMessage(err, "sign failed")
	}

	return &pb.SignedProposal{ProposalBytes: proposalBytes, Signature: signature}, nil
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
	s.Step(`^a signed proposal is created for chaincode "([^"]*)" with args "([^"]*)" with org "([^"]*)" on channel "([^"]*)" and is saved to variable "([^"]*)"$`, d.saveSignedProposalToVar)
	s.Step(`^a signed proposal with an invalid signature is created for chaincode "([^"]*)" with args "([^"]*)" with org "([^"]*)" on channel "([^"]*)" and is saved to variable "([^"]*)"$`, d.saveSignedProposalWithInvalidSignatureToVar)
	s.Step(`^the signed proposal "([^"]*)" is sent to peers "([^"]*)" and the responses are saved to variable "([^"]*)"$`, d.sendProposalAndSaveResponseToVar)
	s.Step(`^the signed proposal "([^"]*)" is sent to peers "([^"]*)" and the invalid responses are saved to variable "([^"]*)"$`, d.sendProposalAndSaveInvalidResponseToVar)
}
