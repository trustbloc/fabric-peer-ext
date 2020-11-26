/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package appdata

import (
	"context"
	"math/rand"

	gproto "github.com/hyperledger/fabric-protos-go/gossip"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/comm"
	gcommon "github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"

	extcommon "github.com/trustbloc/fabric-peer-ext/pkg/common"
	extdiscovery "github.com/trustbloc/fabric-peer-ext/pkg/common/discovery"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/multirequest"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/requestmgr"
)

// PeerFilter returns true to include the peer in the Gossip request
type PeerFilter = func(*extdiscovery.Member) bool

// ResponseHandler handles the response to retrieved data
type ResponseHandler func(response []byte) (extcommon.Values, error)

// AllSet returns true if all of the values have been set
type AllSet func(values extcommon.Values) bool

type gossipService interface {
	PeersOfChannel(id gcommon.ChannelID) []discovery.NetworkMember
	SelfMembershipInfo() discovery.NetworkMember
	IdentityInfo() api.PeerIdentitySet
	Send(msg *gproto.GossipMessage, peers ...*comm.RemotePeer)
}

// Request contains an application data request
type Request struct {
	DataType string
	Payload  []byte
}

// NewRequest returns a new application data request
func NewRequest(dataType string, payload []byte) *Request {
	return &Request{
		DataType: dataType,
		Payload:  payload,
	}
}

type requestCreator func() requestmgr.Request

// Retriever retrieves data from one or more peers on a channel
type Retriever struct {
	*extdiscovery.Discovery
	gossipService
	createRequest     requestCreator
	gossipMaxAttempts int
	gossipMaxPeers    int
}

// NewRetriever returns a new application data retriever
func NewRetriever(channelID string, gossip gossipService, gossipMaxAttempts, gossipMaxPeers int) *Retriever {
	reqMgr := requestmgr.Get(channelID)
	return newRetriever(channelID, gossip, gossipMaxAttempts, gossipMaxPeers, reqMgr.NewRequest)
}

func newRetriever(channelID string, gossip gossipService, gossipMaxAttempts, gossipMaxPeers int, reqCreator requestCreator) *Retriever {
	return &Retriever{
		Discovery:         extdiscovery.New(channelID, gossip),
		gossipService:     gossip,
		createRequest:     reqCreator,
		gossipMaxAttempts: gossipMaxAttempts,
		gossipMaxPeers:    gossipMaxPeers,
	}
}

type options struct {
	peerFilter PeerFilter
}

// Option is a retriever option
type Option func(options *options)

// WithPeerFilter sets a peer filter
func WithPeerFilter(filter PeerFilter) Option {
	return func(options *options) {
		options.peerFilter = filter
	}
}

// Retrieve retrieves application data from one or more peers
func (r *Retriever) Retrieve(ctxt context.Context, request *Request, responseHandler ResponseHandler, allSet AllSet, opts ...Option) (extcommon.Values, error) {
	logger.Debugf("[%s] Retrieving data for request: %+v", r.ChannelID(), request)

	var retrieverOpts options
	for _, opt := range opts {
		opt(&retrieverOpts)
	}

	var values extcommon.Values
	var attemptedPeers extdiscovery.PeerGroup
	for attempt := 1; attempt <= r.gossipMaxAttempts; attempt++ {
		peers := r.getPeersForRetrieval(func(peer *extdiscovery.Member) bool {
			if peer.Local || attemptedPeers.Contains(peer) {
				return false
			}

			if retrieverOpts.peerFilter != nil {
				return retrieverOpts.peerFilter(peer)
			}

			return true
		})

		if len(peers) == 0 {
			logger.Infof("[%s] Exhausted all peers for retrieval on attempt %d", r.ChannelID(), attempt)
			break
		}

		attemptedPeers = append(attemptedPeers, peers...)

		cReq := multirequest.New()

		for _, peer := range peers {
			logger.Debugf("[%s] Adding request to get data from [%s] ...", r.ChannelID(), peer)

			cReq.Add(peer.String(), r.getDataFromPeer(request, peer, responseHandler))
		}

		remoteValues := cReq.Execute(ctxt).Values

		logger.Debugf("[%s] Merging existing values for %s: %s, with values received from remote peers: %s", r.ChannelID(), request.Payload, values, remoteValues)

		values = values.Merge(remoteValues)

		if allSet(values) {
			logger.Debugf("[%s] Got all values on attempt %d for %s: %s", r.ChannelID(), attempt, request.Payload, values)
			break
		}

		logger.Infof("[%s] Could not get all values on attempt %d for %s. Got: %s", r.ChannelID(), attempt, request.Payload, values)
	}

	return values, nil
}

func (r *Retriever) getDataFromPeer(request *Request, endorser *extdiscovery.Member, responseHandler ResponseHandler) multirequest.Request {
	return func(ctxt context.Context) (extcommon.Values, error) {
		logger.Debugf("[%s] Getting data from [%s] ...", r.ChannelID(), endorser)

		values, err := r.getData(ctxt, request, endorser, responseHandler)
		if err != nil {
			if err == context.Canceled {
				logger.Debugf("[%s] Request to get data from [%s] was cancelled", r.ChannelID(), endorser)
			} else {
				logger.Debugf("[%s] Error getting data from [%s]: %s", r.ChannelID(), endorser, err)
			}
			return nil, err
		}

		return values, nil
	}
}

func (r *Retriever) getData(ctxt context.Context, request *Request, peer *extdiscovery.Member, handleResponse ResponseHandler) (extcommon.Values, error) {
	req := r.createRequest()

	logger.Debugf("[%s] Sending Gossip request %d to peer [%s]", r.ChannelID(), req.ID(), peer)

	r.Send(r.createAppDataRequestMsg(request, req.ID()), asRemotePeer(peer))

	logger.Debugf("[%s] Waiting for response from peer [%s] to request %d", r.ChannelID(), peer, req.ID())
	res, err := req.GetResponse(ctxt)
	if err != nil {
		return nil, err
	}

	data := res.Data.([]byte)

	logger.Debugf("[%s] Got response from peer [%s] to request %d: %s", r.ChannelID(), peer, req.ID(), data)

	return handleResponse(data)
}

func (r *Retriever) createAppDataRequestMsg(request *Request, reqID uint64) *gproto.GossipMessage {
	return &gproto.GossipMessage{
		Tag:     gproto.GossipMessage_CHAN_ONLY,
		Channel: []byte(r.ChannelID()),
		Content: &gproto.GossipMessage_AppDataReq{
			AppDataReq: &gproto.AppDataRequest{
				Nonce:    reqID,
				DataType: request.DataType,
				Request:  request.Payload,
			},
		},
	}
}

func (r *Retriever) getPeersForRetrieval(filter PeerFilter) extdiscovery.PeerGroup {
	members := r.GetMembers(filter)

	logger.Debugf("[%s] Peers to choose from: %v", r.ChannelID(), members)

	var peers extdiscovery.PeerGroup

	for _, i := range rand.Perm(len(members)) {
		peers = append(peers, members[i])

		if len(peers) == r.gossipMaxPeers {
			break
		}
	}

	logger.Debugf("[%s] Chose peers: %s", r.ChannelID(), peers)

	return peers
}

func asRemotePeer(m *extdiscovery.Member) *comm.RemotePeer {
	return &comm.RemotePeer{
		Endpoint: m.Endpoint,
		PKIID:    m.PKIid,
	}
}
