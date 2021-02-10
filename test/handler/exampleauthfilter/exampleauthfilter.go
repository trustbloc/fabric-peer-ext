/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package exampleauthfilter

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	cb "github.com/hyperledger/fabric-protos-go/common"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric/common/flogging"
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	gcommon "github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/statedb"

	dcasclient "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/dcas/client"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/api"
)

var logger = flogging.MustGetLogger("ext_example_authfilter")

const (
	endorseFunc                   = "endorse"
	endorseAndCommitFunc          = "endorseandcommit"
	commitFunc                    = "commit"
	verifyProposalSignatureFunc   = "verifyProposalSignature"
	validateProposalResponsesFunc = "validateProposalResponses"
	putCASFunc                    = "putcas"
	getCASFunc                    = "getcas"
	getCASNodeFunc                = "getcasnode"
	getPrivateDataNoLockFunc      = "getprivatenolock"
)

type gossipProvider interface {
	GetGossipService() gossipapi.GossipService
}

type function func(channelID, ccName string, args [][]byte) pb.Response

type txnServiceProvider interface {
	ForChannel(channelID string) (api.Service, error)
}

type dcasClientProvider interface {
	GetDCASClient(channelID string, namespace string, coll string) (dcasclient.DCAS, error)
}

type qeChannelProvider interface {
	QueryExecutorProviderForChannel(channelID string) statedb.QueryExecutorProvider
}

// AuthFilter is a sample Auth filter used in the BDD test. It demonstrates the handler registry and
// dependency injection for Auth filters.
type AuthFilter struct {
	next               pb.EndorserServer
	gossipProvider     gossipProvider
	functionRegistry   map[string]function
	txnProvider        txnServiceProvider
	dcasClientProvider dcasClientProvider
	qeChannelProvider  qeChannelProvider
}

// New returns a new example auth AuthFilter. The Gossip provider is supplied via dependency injection.
func New(gossip gossipProvider, txnProvider txnServiceProvider, dcasClientProvider dcasClientProvider, qep qeChannelProvider) *AuthFilter {
	f := &AuthFilter{
		gossipProvider:     gossip,
		txnProvider:        txnProvider,
		dcasClientProvider: dcasClientProvider,
		qeChannelProvider:  qep,
	}

	f.initFunctionRegistry()

	return f
}

// Name returns the unique name of the Auth filter
func (f *AuthFilter) Name() string {
	return "ExampleAuthFilter"
}

// Init initializes the Auth filter
func (f *AuthFilter) Init(next pb.EndorserServer) {
	logger.Info("Initialized example auth AuthFilter")

	f.next = next
}

// ProcessProposal is invoked during an endorsement (before the chaincode). This implementation
// looks for a specific chaincode name and function and, if satisfied, returns an error response
// with a message that contains the channel peer endpoints.
func (f *AuthFilter) ProcessProposal(ctx context.Context, signedProp *pb.SignedProposal) (*pb.ProposalResponse, error) {
	logger.Debugf("ProcessProposal in example auth AuthFilter was invoked. Signed proposal: %s", signedProp)

	channelID, ccSpec, err := getChannelAndSpec(signedProp)
	if err != nil {
		return nil, err
	}

	var funcName string
	if len(ccSpec.Input.Args) > 0 {
		funcName = string(ccSpec.Input.Args[0])
	}

	if ccSpec.ChaincodeId.Name != "e2e_cc" && ccSpec.ChaincodeId.Name != "tdata_examplecc" {
		return f.next.ProcessProposal(ctx, signedProp)
	}

	if funcName == "authFilterError" {
		logger.Debugf("Returning peers in channel as an error message")

		gossip := f.gossipProvider.GetGossipService()

		var peersString string
		for _, p := range gossip.PeersOfChannel(gcommon.ChannelID(channelID)) {
			peersString += p.Endpoint + " "
		}

		return &pb.ProposalResponse{
			Response: &pb.Response{
				Status:  shim.ERROR,
				Message: fmt.Sprintf("Peers in channel [%s]: %s", channelID, peersString),
			},
		}, nil
	}

	fctn, ok := f.functionRegistry[funcName]
	if !ok {
		return f.next.ProcessProposal(ctx, signedProp)
	}

	resp := fctn(channelID, ccSpec.ChaincodeId.Name, ccSpec.Input.Args[1:])

	return &pb.ProposalResponse{
		Version:     0,
		Timestamp:   nil,
		Response:    &resp,
		Payload:     nil,
		Endorsement: nil,
	}, nil
}

func getChannelAndSpec(signedProp *pb.SignedProposal) (string, *pb.ChaincodeSpec, error) {
	proposal, err := protoutil.UnmarshalProposal(signedProp.ProposalBytes)
	if err != nil {
		return "", nil, err
	}

	channelID, cis, err := getChaincodeInvocationSpec(proposal)
	if err != nil {
		return "", nil, err
	}

	if cis == nil || cis.ChaincodeSpec == nil || cis.ChaincodeSpec.ChaincodeId == nil || cis.ChaincodeSpec.Input == nil {
		return "", nil, fmt.Errorf("invalid cc invocation spec: %s", err)
	}

	return channelID, cis.ChaincodeSpec, nil
}

func getChaincodeInvocationSpec(prop *pb.Proposal) (string, *pb.ChaincodeInvocationSpec, error) {
	if prop == nil {
		return "", nil, errors.New("proposal is nil")
	}

	header, err := getHeader(prop.Header)
	if err != nil {
		return "", nil, err
	}

	channelHeader, err := protoutil.UnmarshalChannelHeader(header.ChannelHeader)
	if err != nil {
		return "", nil, err
	}

	ccPropPayload, err := getChaincodeProposalPayload(prop.Payload)
	if err != nil {
		return "", nil, err
	}

	cis := &pb.ChaincodeInvocationSpec{}
	err = proto.Unmarshal(ccPropPayload.Input, cis)
	if err != nil {
		return "", nil, err
	}

	return channelHeader.ChannelId, cis, nil
}

func getHeader(bytes []byte) (*cb.Header, error) {
	hdr := &cb.Header{}
	err := proto.Unmarshal(bytes, hdr)
	return hdr, errors.Wrap(err, "error unmarshaling Header")
}

func getChaincodeProposalPayload(bytes []byte) (*pb.ChaincodeProposalPayload, error) {
	cpp := &pb.ChaincodeProposalPayload{}
	err := proto.Unmarshal(bytes, cpp)
	return cpp, errors.Wrap(err, "error unmarshaling ChaincodeProposalPayload")
}

func (f *AuthFilter) endorse(channelID, _ string, args [][]byte) pb.Response {
	req, err := getEndorsementRequest(args)
	if err != nil {
		logger.Errorf("Error getting endorsement request for channel [%s]: %s", channelID, err)
		return shim.Error(err.Error())
	}

	txnSvc, err := f.txnProvider.ForChannel(channelID)
	if err != nil {
		logger.Errorf("Error getting transaction service for channel [%s]: %s", channelID, err)
		return shim.Error(fmt.Sprintf("Error getting transaction service for channel [%s]: %s", channelID, err))
	}

	logger.Infof("Executing Endorse on channel [%s]...", channelID)

	resp, err := txnSvc.Endorse(req)
	if err != nil {
		logger.Infof("[%s] Error returned from EndorseAndCommit: %s", channelID, err)

		if errors.Cause(err) == api.ErrInvalidTxnID {
			return f.newInvalidTxnResponse(txnSvc)
		}

		return shim.Error(fmt.Sprintf("Error returned from Endorse: %s", err))
	}

	logger.Infof("... Endorse succeeded on channel [%s]", channelID)

	responseBytes, err := json.Marshal(resp)
	if err != nil {
		logger.Errorf("[%s] Error marshalling endorsement response: %s", channelID, err)

		return shim.Error(fmt.Sprintf("error marshalling endorsement response: %s", err))
	}

	response := &response{
		Status:              http.StatusOK,
		Payload:             string(resp.Payload),
		EndorsementResponse: responseBytes,
	}

	respBytes, err := json.Marshal(response)
	if err != nil {
		return shim.Error(fmt.Sprintf("error marshalling response: %s", err))
	}

	return shim.Success(respBytes)
}

func (f *AuthFilter) endorseAndCommit(channelID, _ string, args [][]byte) pb.Response {
	req, err := getEndorsementRequest(args)
	if err != nil {
		logger.Errorf("Error getting endorsement request for channel [%s]: %s", channelID, err)
		return shim.Error(err.Error())
	}

	txnSvc, err := f.txnProvider.ForChannel(channelID)
	if err != nil {
		return shim.Error(fmt.Sprintf("error getting transaction service for channel [%s]: %s", channelID, err))
	}

	logger.Infof("Executing EndorseAndCommit on channel [%s]...", channelID)

	resp, committed, err := txnSvc.EndorseAndCommit(req)
	if err != nil {
		logger.Infof("[%s] Error returned from EndorseAndCommit: %s", channelID, err)

		if errors.Cause(err) == api.ErrInvalidTxnID {
			return f.newInvalidTxnResponse(txnSvc)
		}

		return shim.Error(fmt.Sprintf("error returned from EndorseAndCommit: %s", err))
	}

	logger.Infof("... EndorseAndCommit succeeded on channel [%s] - Transaction committed: %t", channelID, committed)

	response := &response{
		Status:    http.StatusOK,
		Payload:   string(resp.Payload),
		Committed: committed,
	}

	respBytes, err := json.Marshal(response)
	if err != nil {
		return shim.Error(fmt.Sprintf("error marshalling response: %s", err))
	}

	return shim.Success(respBytes)
}

func (f *AuthFilter) commit(channelID, _ string, args [][]byte) pb.Response {
	req, err := getCommitRequest(args)
	if err != nil {
		logger.Errorf("Error getting endorsement request for channel [%s]: %s", channelID, err)
		return shim.Error(err.Error())
	}

	txnSvc, err := f.txnProvider.ForChannel(channelID)
	if err != nil {
		return shim.Error(fmt.Sprintf("error getting transaction service for channel [%s]: %s", channelID, err))
	}

	logger.Infof("Executing CommitEndorsements on channel [%s]...", channelID)

	resp, committed, err := txnSvc.CommitEndorsements(req)
	if err != nil {
		logger.Infof("[%s] Error returned from CommitEndorsements: %s", channelID, err)

		return shim.Error(fmt.Sprintf("error returned from CommitEndorsements: %s", err))
	}

	logger.Infof("... CommitEndorsements succeeded on channel [%s] - Transaction committed: %t", channelID, committed)

	response := &response{
		Status:    http.StatusOK,
		Payload:   string(resp.Payload),
		Committed: committed,
	}

	respBytes, err := json.Marshal(response)
	if err != nil {
		return shim.Error(fmt.Sprintf("error marshalling response: %s", err))
	}

	return shim.Success(respBytes)
}

func (f *AuthFilter) verifyProposalSignature(channelID, _ string, args [][]byte) pb.Response {
	logger.Infof("[%s] Verifying proposal signature. Signed proposal: %s", channelID, args[0])

	signedProposal := &pb.SignedProposal{}
	if err := json.Unmarshal(args[0], signedProposal); err != nil {
		return shim.Error(fmt.Sprintf("Failed Unmarshal signedProposal: %s", err))
	}

	svc, err := f.txnProvider.ForChannel(channelID)
	if err != nil {
		logger.Errorf("Error getting transaction service for channel [%s]: %s", channelID, err)
		return shim.Error(fmt.Sprintf("Error getting transaction service for channel [%s]: %s", channelID, err))
	}

	err = svc.VerifyProposalSignature(signedProposal)
	if err != nil {
		logger.Infof("[%s] Failed to verify proposal signature", channelID)
		return shim.Error(fmt.Sprintf("VerifyProposalSignature returned error: %s", err))
	}

	logger.Infof("[%s] Successfully verified proposal signature", channelID)

	return shim.Success(nil)
}

func (f *AuthFilter) validateProposalResponses(channelID, _ string, args [][]byte) pb.Response {
	if len(args) != 2 {
		return shim.Error("Expecting args: signed proposal bytes and proposal responses bytes")
	}

	signedProposal := &pb.SignedProposal{}
	if err := json.Unmarshal(args[0], signedProposal); err != nil {
		return shim.Error(fmt.Sprintf("Failed to unmarshal signed proposal: %s", err))
	}

	var proposalResponses []*pb.ProposalResponse
	if err := json.Unmarshal(args[1], &proposalResponses); err != nil {
		return shim.Error(fmt.Sprintf("Failed to unmarshal proposal responses: %s", err))
	}

	svc, err := f.txnProvider.ForChannel(channelID)
	if err != nil {
		logger.Errorf("Error getting transaction service for channel [%s]: %s", channelID, err)
		return shim.Error(fmt.Sprintf("Error getting transaction service for channel [%s]: %s", channelID, err))
	}

	code, err := svc.ValidateProposalResponses(signedProposal, proposalResponses)
	if err != nil {
		return shim.Error(fmt.Sprintf("ValidateProposalResponses returned error: %s", err))
	}

	codeBytes, err := json.Marshal(code)
	if err != nil {
		panic(err)
	}

	logger.Infof("[%s] Successfully validated proposal responses", channelID)

	return shim.Success(codeBytes)
}

func (f *AuthFilter) putCAS(channelID, _ string, args [][]byte) pb.Response {
	if len(args) < 3 {
		return shim.Error("Expecting args: namespace, collection, value and options")
	}

	ns := args[0]
	coll := args[1]
	value := args[2]

	var opts []dcasclient.Option

	if len(args) > 3 {
		var err error
		opts, err = parseDCASOptions(string(args[3]))
		if err != nil {
			return shim.Error(fmt.Sprintf("One or more options is invalid: %s", err))
		}
	}

	casClient, err := f.dcasClientProvider.GetDCASClient(channelID, string(ns), string(coll))
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to create DCAS client: %s", err))
	}

	cid, err := casClient.Put(bytes.NewReader(value), opts...)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to store data to CAS: %s", err))
	}

	return shim.Success([]byte(cid))
}

func (f *AuthFilter) getCAS(channelID, _ string, args [][]byte) pb.Response {
	if len(args) != 3 {
		return shim.Error("Expecting args: namespace, collection, and CID")
	}

	ns := args[0]
	coll := args[1]
	cid := args[2]

	casClient, err := f.dcasClientProvider.GetDCASClient(channelID, string(ns), string(coll))
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to create DCAS client: %s", err))
	}

	value := bytes.NewBuffer(nil)
	err = casClient.Get(string(cid), value)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to retrieve data from CAS: %s", err))
	}

	return shim.Success(value.Bytes())
}

func (f *AuthFilter) getCASNode(channelID, _ string, args [][]byte) pb.Response {
	if len(args) != 3 {
		return shim.Error("Expecting args: namespace, collection, and CID")
	}

	ns := args[0]
	coll := args[1]
	cid := args[2]

	casClient, err := f.dcasClientProvider.GetDCASClient(channelID, string(ns), string(coll))
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to create DCAS client: %s", err))
	}

	nd, err := casClient.GetNode(string(cid))
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to retrieve data from CAS: %s", err))
	}

	bytes, err := json.Marshal(nd)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to marshal CAS node: %s", err))
	}

	return shim.Success(bytes)
}

func (f *AuthFilter) getPrivateDataNoLock(channelID, ccName string, args [][]byte) pb.Response {
	if len(args) != 2 {
		return shim.Error("Expecting args: collection, and key")
	}

	coll := string(args[0])
	key := string(args[1])

	qe, err := f.qeChannelProvider.QueryExecutorProviderForChannel(channelID).NewQueryExecutorNoLock()
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to create DCAS client: %s", err))
	}

	data, err := qe.GetPrivateData(ccName, coll, key)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to retrieve private data: %s", err))
	}

	logger.Infof("[%s] Returning private data for [%s:%s:%s]: %s", channelID, ccName, coll, key, data)

	return shim.Success(data)
}

func (f *AuthFilter) initFunctionRegistry() {
	f.functionRegistry = make(map[string]function)
	f.functionRegistry[endorseFunc] = f.endorse
	f.functionRegistry[endorseAndCommitFunc] = f.endorseAndCommit
	f.functionRegistry[commitFunc] = f.commit
	f.functionRegistry[verifyProposalSignatureFunc] = f.verifyProposalSignature
	f.functionRegistry[validateProposalResponsesFunc] = f.validateProposalResponses
	f.functionRegistry[putCASFunc] = f.putCAS
	f.functionRegistry[getCASFunc] = f.getCAS
	f.functionRegistry[getCASNodeFunc] = f.getCASNode
	f.functionRegistry[getPrivateDataNoLockFunc] = f.getPrivateDataNoLock
}

func (f *AuthFilter) newInvalidTxnResponse(txnSvc api.Service) pb.Response {
	// Send back the signing identity so that the client can formulate a proper TxnID
	identity, err := txnSvc.SigningIdentity()
	if err != nil {
		logger.Errorf("Error getting signing identity: %s", err)

		return shim.Error("could not get signing identity")
	}

	r := &response{
		Status:  http.StatusBadRequest,
		Payload: base64.URLEncoding.EncodeToString(identity),
	}

	respBytes, err := json.Marshal(r)
	if err != nil {
		logger.Errorf("Error marshalling response: %s", err)

		return shim.Error("error marshalling response")
	}

	return shim.Success(respBytes)
}

type response struct {
	Status              int
	Payload             string
	Committed           bool
	SigningIdentity     string
	EndorsementResponse []byte
}

type endorsementRequest struct {
	ChaincodeID      string          `json:"cc_id"`
	Args             []string        `json:"args"`
	CommitType       string          `json:"commit_type"`
	IgnoreNameSpaces []api.Namespace `json:"ignore_namespaces"`
	PeerFilter       string          `json:"peer_filter"`
	PeerFilterArgs   []string        `json:"peer_filter_args"`
	TransactionID    string          `json:"tx_id"`
	Nonce            string          `json:"nonce"`
	AsyncCommit      bool            `json:"async_commit"`
}

type commitRequest struct {
	CommitType          string          `json:"commit_type"`
	IgnoreNameSpaces    []api.Namespace `json:"ignore_namespaces"`
	EndorsementResponse []byte          `json:"endorsement_response"`
	AsyncCommit         bool            `json:"async_commit"`
}

func getEndorsementRequest(args [][]byte) (*api.Request, error) {
	if len(args) < 1 {
		return nil, errors.New("expecting endorsement request")
	}

	logger.Infof("Got endorsement request: %s", args[0])

	request := endorsementRequest{}
	err := json.Unmarshal(args[0], &request)
	if err != nil {
		return nil, errors.Errorf("error unmarshalling endorsement request: %s", err)
	}

	commitType, err := asCommitType(request.CommitType)
	if err != nil {
		return nil, err
	}

	var peerFilter api.PeerFilter
	switch request.PeerFilter {
	case "msp":
		peerFilter = newMSPPeerFilter(request.PeerFilterArgs)
	}

	var nonce []byte

	if request.Nonce != "" {
		var err error
		nonce, err = base64.URLEncoding.DecodeString(request.Nonce)
		if err != nil {
			return nil, errors.WithMessage(err, "invalid base64 URL encoded nonce")
		}
	}

	return &api.Request{
		ChaincodeID:      request.ChaincodeID,
		Args:             asByteArrays(request.Args),
		CommitType:       commitType,
		IgnoreNameSpaces: request.IgnoreNameSpaces,
		PeerFilter:       peerFilter,
		TransactionID:    request.TransactionID,
		Nonce:            nonce,
		AsyncCommit:      request.AsyncCommit,
	}, nil
}

func getCommitRequest(args [][]byte) (*api.CommitRequest, error) {
	if len(args) < 1 {
		return nil, errors.New("expecting commit request")
	}

	logger.Infof("Got commit request: %s", args[0])

	request := commitRequest{}
	err := json.Unmarshal(args[0], &request)
	if err != nil {
		return nil, errors.Errorf("error unmarshalling commit request: %s", err)
	}

	commitType, err := asCommitType(request.CommitType)
	if err != nil {
		return nil, err
	}

	endorsementResponse := &channel.Response{}
	err = json.Unmarshal(request.EndorsementResponse, endorsementResponse)
	if err != nil {
		return nil, errors.Errorf("error unmarshalling endorsement response: %s", err)
	}

	return &api.CommitRequest{
		EndorsementResponse: endorsementResponse,
		CommitType:          commitType,
		IgnoreNameSpaces:    request.IgnoreNameSpaces,
		AsyncCommit:         request.AsyncCommit,
	}, nil
}

func asByteArrays(args []string) [][]byte {
	arr := make([][]byte, len(args))

	for i, arg := range args {
		arr[i] = []byte(arg)
	}

	return arr
}

func asCommitType(t string) (api.CommitType, error) {
	switch t {
	case "":
		return api.CommitOnWrite, nil
	case "commit-on-write":
		return api.CommitOnWrite, nil
	case "commit":
		return api.Commit, nil
	case "no-commit":
		return api.NoCommit, nil
	default:
		return 0, errors.Errorf("invalid commit_type: [%s]", t)
	}
}

type mspPeerFilter struct {
	mspIDs []string
}

func newMSPPeerFilter(mspIDs []string) api.PeerFilter {
	logger.Infof("[%s] Creating MSP filter that chooses peers in MSPs %s", mspIDs)

	return &mspPeerFilter{
		mspIDs: mspIDs,
	}
}

func (f *mspPeerFilter) Accept(p api.Peer) bool {
	for _, mspID := range f.mspIDs {
		if p.MSPID() == mspID {
			logger.Infof("Accepting peer [%s] since it is a member of %s", p.Endpoint(), f.mspIDs)

			return true
		}
	}

	logger.Infof("Not accepting peer [%s] since it is NOT a member of %s", p.Endpoint(), f.mspIDs)

	return false
}

func parseDCASOptions(options string) ([]dcasclient.Option, error) {
	var opts []dcasclient.Option

	for _, pair := range strings.Split(options, ";") {
		kv := strings.Split(pair, "=")
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid option: %s - option must be in the format name=value", pair)
		}

		key := kv[0]
		value := kv[1]

		switch key {
		case "node-type":
			opts = append(opts, dcasclient.WithNodeType(dcasclient.NodeType(value)))
		case "input-encoding":
			opts = append(opts, dcasclient.WithInputEncoding(dcasclient.InputEncoding(value)))
		case "format":
			opts = append(opts, dcasclient.WithFormat(dcasclient.Format(value)))
		default:
			return nil, fmt.Errorf("unsupported option: %s", key)
		}
	}

	return opts, nil
}
