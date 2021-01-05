/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dispatcher

import (
	"time"

	gproto "github.com/hyperledger/fabric-protos-go/gossip"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/extensions/collections/api/store"
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	ledgerconfig "github.com/hyperledger/fabric/extensions/roles"
	"github.com/hyperledger/fabric/gossip/protoext"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"

	collcommon "github.com/trustbloc/fabric-peer-ext/pkg/collections/common"
	"github.com/trustbloc/fabric-peer-ext/pkg/common"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/discovery"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/requestmgr"
	"github.com/trustbloc/fabric-peer-ext/pkg/gossip/appdata"
)

var logger = flogging.MustGetLogger("ext_dispatcher")

// isEndorser should only be overridden for unit testing
var isEndorser = func() bool {
	return ledgerconfig.IsEndorser()
}

// isCommitter should only be overridden for unit testing
var isCommitter = func() bool {
	return ledgerconfig.IsCommitter()
}

type appDataHandlerProvider interface {
	HandlerForType(dataType string) (appdata.Handler, bool)
}

type responder struct {
	channelID string
	request   protoext.ReceivedMessage
}

func newResponder(channelID string, req protoext.ReceivedMessage) *responder {
	return &responder{
		channelID: channelID,
		request:   req,
	}
}

func (r *responder) Respond(resp []byte) {
	msg := r.request.GetGossipMessage()
	reqID := msg.GetAppDataReq().Nonce

	logger.Debugf("[%s] Responding with application response for request %d", r.channelID, reqID)

	r.request.Respond(&gproto.GossipMessage{
		// Copy nonce field from the request, so it will be possible to match response
		Nonce:   msg.Nonce,
		Tag:     gproto.GossipMessage_CHAN_ONLY,
		Channel: []byte(r.channelID),
		Content: &gproto.GossipMessage_AppDataRes{
			AppDataRes: &gproto.AppDataResponse{
				Nonce:    reqID,
				Response: resp,
			},
		},
	})
}

// Dispatcher is a Gossip message dispatcher
type Dispatcher struct {
	appDataHandlerProvider
	ccProvider collcommon.CollectionConfigProvider
	channelID  string
	reqMgr     requestmgr.RequestMgr
	dataStore  storeapi.Store
	discovery  *discovery.Discovery
}

// Dispatch handles the message and returns true if the message was handled; false if the message is unrecognized
func (s *Dispatcher) Dispatch(msg protoext.ReceivedMessage) bool {
	switch {
	case msg.GetGossipMessage().GetCollDataReq() != nil:
		logger.Debug("Handling collection data request message")
		s.handleDataRequest(msg)
		return true
	case msg.GetGossipMessage().GetCollDataRes() != nil:
		logger.Debug("Handling collection data response message")
		s.handleDataResponse(msg)
		return true
	case msg.GetGossipMessage().GetAppDataReq() != nil:
		logger.Debug("Handling application data request message")
		s.handleAppDataRequest(msg)
		return true
	case msg.GetGossipMessage().GetAppDataRes() != nil:
		logger.Debug("Handling application data response message")
		s.handleAppDataResponse(msg)
		return true
	default:
		logger.Debug("Not handling request")
		return false
	}
}

func (s *Dispatcher) handleDataRequest(msg protoext.ReceivedMessage) {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("[ENTER] -> handleDataRequest")
		defer logger.Debug("[EXIT] ->  handleDataRequest")
	}

	if !isEndorser() && !isCommitter() {
		logger.Warningf("Only endorsers and committers should be receiving collection data request messages")
		return
	}

	req := msg.GetGossipMessage().GetCollDataReq()
	if len(req.Digests) == 0 {
		logger.Warning("Got nil digests in CollDataRequestMsg")
		return
	}

	reqMSPID := s.mspIDFromPKIID(msg.GetConnectionInfo())
	if reqMSPID == "" {
		return
	}

	responses, err := s.getRequestData(reqMSPID, req)
	if err != nil {
		logger.Warningf("[%s] Error processing request for data: %s", s.channelID, err.Error())
		return
	}

	logger.Debugf("[%s] Responding with collection data for request %d", s.channelID, req.Nonce)

	msg.Respond(&gproto.GossipMessage{
		// Copy nonce field from the request, so it will be possible to match response
		Nonce:   msg.GetGossipMessage().Nonce,
		Tag:     gproto.GossipMessage_CHAN_ONLY,
		Channel: []byte(s.channelID),
		Content: &gproto.GossipMessage_CollDataRes{
			CollDataRes: &gproto.RemoteCollDataResponse{
				Nonce:    req.Nonce,
				Elements: responses,
			},
		},
	})
}

func (s *Dispatcher) handleDataResponse(msg protoext.ReceivedMessage) {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debug("[ENTER] -> handleDataResponse")
		defer logger.Debug("[EXIT] ->  handleDataResponse")
	}

	mspID := s.mspIDFromPKIID(msg.GetConnectionInfo())
	if mspID == "" {
		return
	}

	res := msg.GetGossipMessage().GetCollDataRes()

	s.reqMgr.Respond(
		res.Nonce,
		&requestmgr.Response{
			Endpoint: msg.GetConnectionInfo().Endpoint,
			MSPID:    mspID,
			// FIXME: Should the message be signed?
			//Signature:   element.Signature,
			//Identity:    element.Identity,
			Data: s.getResponseData(res),
		},
	)
}

func (s *Dispatcher) getRequestData(reqMSPID string, req *gproto.RemoteCollDataRequest) ([]*gproto.CollDataElement, error) {
	var responses []*gproto.CollDataElement
	for _, digest := range req.Digests {
		if digest == nil {
			return nil, errors.New("got nil digest in CollDataRequestMsg")
		}
		e, err := s.getRequestDataElement(reqMSPID, digest)
		if err != nil {
			return nil, err
		}
		responses = append(responses, e)
	}
	return responses, nil
}

func (s *Dispatcher) getRequestDataElement(reqMSPID string, digest *gproto.CollDataDigest) (*gproto.CollDataElement, error) {
	key := store.NewKey(digest.EndorsedAtTxID, digest.Namespace, digest.Collection, digest.Key)

	logger.Debugf("[%s] Getting data for key [%s]", s.channelID, key)
	value, err := s.getDataForKey(key)
	if err != nil {
		return nil, errors.WithMessagef(err, "error getting data for [%s]", key)
	}

	e := &gproto.CollDataElement{
		Digest: digest,
	}

	authorized, err := s.isAuthorized(reqMSPID, digest.Namespace, digest.Collection)
	if err != nil {
		return nil, err
	}

	if !authorized {
		logger.Infof("[%s] Requesting MSP [%s] is not authorized to read data for [%s]", s.channelID, reqMSPID, key)
	} else if value != nil {
		e.Value = value.Value
		e.ExpiryTime = common.ToTimestamp(value.Expiry)
	}

	return e, nil
}

func (s *Dispatcher) getResponseData(res *gproto.RemoteCollDataResponse) []*requestmgr.Element {
	var elements []*requestmgr.Element
	for _, e := range res.Elements {
		d := e.Digest
		logger.Debugf("[%s] Coll data response for request %d - [%s:%s:%s] received", s.channelID, res.Nonce, d.Namespace, d.Collection, d.Key)

		element := &requestmgr.Element{
			Namespace:  d.Namespace,
			Collection: d.Collection,
			Key:        d.Key,
			Value:      e.Value,
		}

		if e.ExpiryTime != nil {
			element.Expiry = time.Unix(e.ExpiryTime.Seconds, 0)
		}
		elements = append(elements, element)
	}
	return elements
}

func (s *Dispatcher) getDataForKey(key *storeapi.Key) (*storeapi.ExpiringValue, error) {
	logger.Debugf("[%s] Getting config for [%s:%s]", s.channelID, key.Namespace, key.Collection)
	config, err := s.ccProvider.ForChannel(s.channelID).Config(key.Namespace, key.Collection)
	if err != nil {
		return nil, err
	}

	switch config.Type {
	case pb.CollectionType_COL_TRANSIENT:
		logger.Debugf("[%s] Getting transient data for key [%s]", s.channelID, key)
		return s.dataStore.GetTransientData(key)
	case pb.CollectionType_COL_DCAS:
		fallthrough
	case pb.CollectionType_COL_OFFLEDGER:
		logger.Debugf("[%s] Getting off-ledger data for key [%s]", s.channelID, key)
		return s.dataStore.GetData(key)
	default:
		return nil, errors.Errorf("unsupported collection type: [%s]", config.Type)
	}
}

// isAuthorized determines whether the given MSP ID is authorized to read data from the given collection
func (s *Dispatcher) isAuthorized(mspID string, ns, coll string) (bool, error) {
	policy, err := s.ccProvider.ForChannel(s.channelID).Policy(ns, coll)
	if err != nil {
		return false, errors.WithMessagef(err, "unable to get policy for collection [%s:%s]", ns, coll)
	}

	for memberMSPID := range policy.MemberOrgs() {
		if memberMSPID == mspID {
			return true, nil
		}
	}

	return false, nil
}

func (s *Dispatcher) handleAppDataRequest(msg protoext.ReceivedMessage) {
	req := msg.GetGossipMessage().GetAppDataReq()

	resp := newResponder(s.channelID, msg)

	handleRequest, ok := s.HandlerForType(req.DataType)
	if ok {
		logger.Debugf("[%s] Handling application data request from [%s] of type [%s]", s.channelID, msg.GetConnectionInfo().Endpoint, req.DataType)

		handleRequest(s.channelID, req, resp)
	} else {
		logger.Warnf("[%s] No handler registered for data type [%s]", s.channelID, req.DataType)

		// Respond with nil so that the requestor does not wait until he times out
		resp.Respond(nil)
	}
}

func (s *Dispatcher) handleAppDataResponse(msg protoext.ReceivedMessage) {
	mspID := s.mspIDFromPKIID(msg.GetConnectionInfo())
	if mspID == "" {
		return
	}

	res := msg.GetGossipMessage().GetAppDataRes()

	s.reqMgr.Respond(
		res.Nonce,
		&requestmgr.Response{
			Endpoint: msg.GetConnectionInfo().Endpoint,
			MSPID:    mspID,
			Data:     res.Response,
		},
	)
}

func (s *Dispatcher) mspIDFromPKIID(conn *protoext.ConnectionInfo) string {
	mspid, ok := s.discovery.GetMSPID(conn.ID)
	if !ok {
		logger.Warningf("Unable to get MSP ID from PKI ID of remote endpoint [%s]", conn.Endpoint)
		return ""
	}

	return mspid
}
