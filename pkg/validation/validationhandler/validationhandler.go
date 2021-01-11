/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validationhandler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/bluele/gcache"
	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric-protos-go/common"
	gproto "github.com/hyperledger/fabric-protos-go/gossip"
	"github.com/hyperledger/fabric/common/flogging"
	xgossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/comm"
	gcommon "github.com/hyperledger/fabric/gossip/common"
	discovery2 "github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/msp"
	"github.com/pkg/errors"

	extcommon "github.com/trustbloc/fabric-peer-ext/pkg/common"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/discovery"
	"github.com/trustbloc/fabric-peer-ext/pkg/config"
	"github.com/trustbloc/fabric-peer-ext/pkg/gossip/appdata"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
	vcommon "github.com/trustbloc/fabric-peer-ext/pkg/validation/common"
	"github.com/trustbloc/fabric-peer-ext/pkg/validation/validationpolicy"
	"github.com/trustbloc/fabric-peer-ext/pkg/validation/validationresults"
)

var logger = flogging.MustGetLogger("ext_validation")

const validateBlockDataType = "validate-block"

type distributedValidatorProvider interface {
	GetValidatorForChannel(channelID string) vcommon.DistributedValidator
}

type appDataHandlerRegistry interface {
	Register(dataType string, handler appdata.Handler) error
}

type dataRetriever interface {
	Retrieve(ctxt context.Context, request *appdata.Request, responseHandler appdata.ResponseHandler, allSet appdata.AllSet, opts ...appdata.Option) (extcommon.Values, error)
}

type gossipProvider interface {
	GetGossipService() xgossipapi.GossipService
}

type identityProvider interface {
	GetDefaultSigningIdentity() (msp.SigningIdentity, error)
}

type blockPublisherProvider interface {
	ForChannel(channelID string) xgossipapi.BlockPublisher
}

type contextProvider interface {
	ValidationContextForBlock(channelID string, blockNum uint64) (context.Context, error)
}

// Providers contains the dependencies for the validation handler
type Providers struct {
	AppDataHandlerRegistry        appDataHandlerRegistry
	DistributedValidationProvider distributedValidatorProvider
	GossipProvider                gossipProvider
	IdentityProvider              identityProvider
	BlockPublisherProvider        blockPublisherProvider
	ContextProvider               contextProvider
}

// Provider implements a distributed validation handler provider
type Provider struct {
	handlers gcache.Cache
}

// NewProvider returns a new distributed validation handler provider
func NewProvider(providers *Providers) *Provider {
	logger.Info("Creating distributed validation handler provider")

	p := &Provider{
		handlers: gcache.New(0).LoaderFunc(func(cID interface{}) (interface{}, error) {
			channelID := cID.(string)

			return newHandler(
					channelID,
					providers.GossipProvider.GetGossipService(),
					providers.DistributedValidationProvider.GetValidatorForChannel(channelID),
					providers.IdentityProvider,
					providers.BlockPublisherProvider.ForChannel(channelID),
					providers.ContextProvider),
				nil
		}).Build(),
	}

	if config.IsDistributedValidationEnabled() && roles.IsValidator() && !roles.IsCommitter() {
		logger.Info("Registering block validation request handler")

		if err := providers.AppDataHandlerRegistry.Register(validateBlockDataType, p.handleValidateRequest); err != nil {
			// Should never happen
			panic(err)
		}
	}

	return p
}

// Close cleans up all of the resources in the provider
func (p *Provider) Close() {
	for _, h := range p.handlers.GetALL() {
		h.(*handler).Close()
	}
}

// SendValidationRequest sends a validation request to remote peers
func (p *Provider) SendValidationRequest(channelID string, req *vcommon.ValidationRequest) {
	p.getHandler(channelID).submitValidationRequest(req)
}

// ValidatePending validates any pending blocks
func (p *Provider) ValidatePending(channelID string, blockNum uint64) {
	p.getHandler(channelID).validatePending(blockNum)
}

func (p *Provider) handleValidateRequest(channelID string, req *gproto.AppDataRequest, responder appdata.Responder) {
	p.getHandler(channelID).handleValidateRequest(req, responder)
}

func (p *Provider) getHandler(channelID string) *handler {
	h, err := p.handlers.Get(channelID)
	if err != nil {
		// Should never happen
		panic(err)
	}

	return h.(*handler)
}

type handler struct {
	*discovery.Discovery
	xgossipapi.BlockPublisher
	validator vcommon.DistributedValidator
	dataRetriever
	channelID      string
	requestChan    chan *vcommon.ValidationRequest
	ip             identityProvider
	requestCache   *requestCache
	cp             contextProvider
	requestTimeout time.Duration
}

type gossipAdapter interface {
	PeersOfChannel(id gcommon.ChannelID) []discovery2.NetworkMember
	SelfMembershipInfo() discovery2.NetworkMember
	IdentityInfo() api.PeerIdentitySet
	Send(msg *gproto.GossipMessage, peers ...*comm.RemotePeer)
}

func newHandler(channelID string, gossip gossipAdapter, validator vcommon.DistributedValidator, ip identityProvider, bp xgossipapi.BlockPublisher, cp contextProvider) *handler {
	h := &handler{
		Discovery:      discovery.New(channelID, gossip),
		BlockPublisher: bp,
		validator:      validator,
		channelID:      channelID,
		requestChan:    make(chan *vcommon.ValidationRequest, 10),
		dataRetriever:  appdata.NewRetriever(channelID, gossip, 1, 0),
		ip:             ip,
		requestCache:   newRequestCache(channelID),
		cp:             cp,
		requestTimeout: config.GetValidationWaitTime(),
	}

	go h.dispatchValidationRequests()

	return h
}

// Close cleans up resources in the handler
func (h *handler) Close() {
	logger.Infof("[%s] Closing handler", h.channelID)

	close(h.requestChan)
}

// handleValidateRequest handles a validation request and responds to the given responder with the validation results
func (h *handler) handleValidateRequest(req *gproto.AppDataRequest, responder appdata.Responder) {
	block := &cb.Block{}
	err := proto.Unmarshal(req.Request, block)
	if err != nil {
		logger.Errorf("[%s] Error unmarshalling block: %s", h.channelID, err)

		return
	}

	logger.Debugf("[%s] Handling validation request for block %d", h.channelID, block.Header.Number)

	currentHeight := h.LedgerHeight()

	if block.Header.Number == currentHeight {
		logger.Infof("[%s] Validating block [%d] with %d transaction(s)", h.channelID, block.Header.Number, len(block.Data.Data))

		ctx, err := h.cp.ValidationContextForBlock(h.channelID, block.Header.Number)
		if err != nil {
			logger.Errorf("[%s] Unable to validate block %d: %s", h.channelID, block.Header.Number, err)

			return
		}

		h.validate(ctx, block, responder)
	} else if block.Header.Number > currentHeight {
		logger.Infof("[%s] Block [%d] with %d transaction(s) cannot be validated yet since our ledger height is %d. Adding to cache.", h.channelID, block.Header.Number, len(block.Data.Data), currentHeight)

		h.requestCache.Add(block, responder)
	} else {
		logger.Infof("[%s] Block [%d] will not be validated since the block has already been committed. Our ledger height is %d.", h.channelID, block.Header.Number, currentHeight)
	}
}

func (h *handler) validate(ctx context.Context, block *cb.Block, responder appdata.Responder) {
	results, txIDs, err := h.validator.ValidatePartial(ctx, block)

	var signature, identity []byte

	var errStr string
	if err != nil {
		if errors.Is(err, context.Canceled) {
			logger.Debugf("[%s] Validation of block %d was cancelled", h.channelID, block.Header.Number)

			// Don't send a response
			return
		}

		logger.Warningf("[%s] Error validating partial block %d: %s", h.channelID, block.Header.Number, err)
		errStr = err.Error()
	} else {
		logger.Debugf("[%s] Done validating partial block %d", h.channelID, block.Header.Number)

		signature, identity, err = h.signValidationResults(block.Header.Number, results)
		if err != nil {
			logger.Errorf("Error signing validation results: %s", err)
		}
	}

	valResults := &validationresults.Results{
		BlockNumber: block.Header.Number,
		TxFlags:     results,
		TxIDs:       txIDs,
		Err:         errStr,
		Endpoint:    h.Self().Endpoint,
		MSPID:       h.Self().MSPID,
		Signature:   signature,
		Identity:    identity,
	}

	resBytes, err := json.Marshal(valResults)
	if err != nil {
		logger.Errorf("[%s] Error marshalling results: %s", h.channelID, err)

		return
	}

	responder.Respond(resBytes)
}

func (h *handler) validatePending(blockNum uint64) {
	logger.Debugf("[%s] Checking for pending request for block %d", h.channelID, blockNum)

	req := h.requestCache.Remove(blockNum)
	if req != nil {
		logger.Infof("[%s] Validating pending request for block %d", h.channelID, blockNum)

		ctx, err := h.cp.ValidationContextForBlock(h.channelID, blockNum)
		if err != nil {
			logger.Errorf("[%s] Unable to validate pending block %d: %s", h.channelID, blockNum, err)

			return
		}

		h.validate(ctx, req.block, req.responder)
	} else {
		logger.Debugf("[%s] Pending request not found for block %d", h.channelID, blockNum)
	}
}

func (h *handler) sendValidationRequest(req *vcommon.ValidationRequest) error {
	blockNum := req.Block.Header.Number

	logger.Debugf("[%s] Sending validation request for block %d", h.channelID, blockNum)

	parentCtx, err := h.cp.ValidationContextForBlock(h.channelID, blockNum)
	if err != nil {
		return errors.WithMessagef(err, "unable to send validation request for block %d", blockNum)
	}

	validatingPeers, err := h.validator.GetValidatingPeers(req.Block)
	if err != nil {
		return errors.WithMessagef(err, "unable to send validation request for block %d", blockNum)
	}

	if !validatingPeers.ContainsRemote() {
		logger.Infof("[%s] There are no remote validating peers for block %d", h.channelID, blockNum)

		return nil
	}

	mapPeerToIdx := make(map[string]int)

	for i, m := range validatingPeers {
		mapPeerToIdx[m.Endpoint] = i
	}

	ctx, cancel := context.WithTimeout(parentCtx, h.requestTimeout)
	defer cancel()

	// The resulting value doesn't matter since we send the partial results immediately to the committer as they are received
	_, err = h.Retrieve(
		ctx,
		&appdata.Request{
			DataType: validateBlockDataType,
			Payload:  req.BlockBytes,
		},
		h.getResponseHandler(mapPeerToIdx),
		func(values extcommon.Values) bool {
			return values.AllSet()
		},
		appdata.WithPeerFilter(h.peerFilter(validatingPeers, blockNum)),
	)

	return err
}

func (h *handler) submitValidationRequest(req *vcommon.ValidationRequest) {
	logger.Debugf("[%s] Submitting validation request for block %d", h.channelID, req.Block.Header.Number)

	h.requestChan <- req
}

func (h *handler) dispatchValidationRequests() {
	logger.Infof("[%s] Starting validation dispatcher", h.channelID)

	for req := range h.requestChan {
		if err := h.sendValidationRequest(req); err != nil {
			logger.Errorf("[%s] Error sending validation request: %s", h.channelID, err)
		}
	}

	logger.Infof("[%s] Validation dispatcher shutting down", h.channelID)
}

func (h *handler) signValidationResults(blockNum uint64, txFlags []byte) ([]byte, []byte, error) {
	logger.Infof("[%s] Signing validation results for block %d", h.channelID, blockNum)

	signer, err := h.ip.GetDefaultSigningIdentity()
	if err != nil {
		return nil, nil, err
	}

	// serialize the signing identity
	identityBytes, err := signer.Serialize()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "could not serialize the signing identity")
	}

	// sign the concatenation of the block number, results, and the serialized signer identity with this peer's key
	signature, err := signer.Sign(validationpolicy.GetDataToSign(blockNum, txFlags, identityBytes))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "could not sign the proposal response payload")
	}

	return signature, identityBytes, nil
}

func (h *handler) getResponseHandler(mapPeerToIdx map[string]int) func(response []byte) (extcommon.Values, error) {
	return func(response []byte) (extcommon.Values, error) {
		vr := &validationresults.Results{}
		if err := json.Unmarshal(response, vr); err != nil {
			return nil, err
		}

		logger.Debugf("[%s] Got validation response from %s for block %d", h.channelID, vr.Endpoint, vr.BlockNumber)

		// Immediately submit the results so that the committer can merge the results
		h.validator.SubmitValidationResults(vr)

		values := make(extcommon.Values, len(mapPeerToIdx))

		idx, ok := mapPeerToIdx[vr.Endpoint]
		if !ok {
			logger.Warningf("[%s] Peer index for %s not found for block %d", h.channelID, vr.Endpoint, vr.BlockNumber)
		}

		values[idx] = vr

		return values, nil
	}
}

func (h *handler) peerFilter(validatingPeers discovery.PeerGroup, blockNum uint64) appdata.PeerFilter {
	return func(member *discovery.Member) bool {
		if validatingPeers.Contains(member) {
			logger.Debugf("[%s] Sending validation request for block %d to %s", h.channelID, blockNum, member.Endpoint)

			return true
		}

		logger.Debugf("[%s] Not sending validation request for block %d to %s", h.channelID, blockNum, member.Endpoint)

		return false
	}
}
