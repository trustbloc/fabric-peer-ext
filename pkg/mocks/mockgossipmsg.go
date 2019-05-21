/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"github.com/hyperledger/fabric/extensions/collections/api/store"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/protoext"
	gproto "github.com/hyperledger/fabric/protos/gossip"
	"github.com/trustbloc/fabric-peer-ext/pkg/common"
)

// KeyVal stores a key and a value
type KeyVal struct {
	*store.Key
	*store.ExpiringValue
}

// NewKeyValue creates a new KeyVal
func NewKeyValue(key *store.Key, value *store.ExpiringValue) *KeyVal {
	return &KeyVal{
		Key:           key,
		ExpiringValue: value,
	}
}

// NewCollDataReqMsg returns a mock collection data request message
func NewCollDataReqMsg(channelID string, reqID uint64, keys ...*store.Key) *protoext.SignedGossipMessage {
	var digests []*gproto.CollDataDigest
	for _, key := range keys {
		digests = append(digests, &gproto.CollDataDigest{
			Namespace:      key.Namespace,
			Collection:     key.Collection,
			Key:            key.Key,
			EndorsedAtTxID: key.EndorsedAtTxID,
		})
	}

	msg, _ := protoext.NoopSign(&gproto.GossipMessage{
		Tag:     gproto.GossipMessage_CHAN_ONLY,
		Channel: []byte(channelID),
		Content: &gproto.GossipMessage_CollDataReq{
			CollDataReq: &gproto.RemoteCollDataRequest{
				Nonce:   reqID,
				Digests: digests,
			},
		},
	})
	return msg
}

// NewCollDataResMsg returns a mock collection data response message
func NewCollDataResMsg(channelID string, reqID uint64, keyVals ...*KeyVal) *protoext.SignedGossipMessage {
	var elements []*gproto.CollDataElement
	for _, kv := range keyVals {
		elements = append(elements, &gproto.CollDataElement{
			Digest: &gproto.CollDataDigest{
				Namespace:      kv.Namespace,
				Collection:     kv.Collection,
				Key:            kv.Key.Key,
				EndorsedAtTxID: kv.EndorsedAtTxID,
			},
			Value:      kv.Value,
			ExpiryTime: common.ToTimestamp(kv.Expiry),
		})
	}

	msg, _ := protoext.NoopSign(&gproto.GossipMessage{
		Tag:     gproto.GossipMessage_CHAN_ONLY,
		Channel: []byte(channelID),
		Content: &gproto.GossipMessage_CollDataRes{
			CollDataRes: &gproto.RemoteCollDataResponse{
				Nonce:    reqID,
				Elements: elements,
			},
		},
	})
	return msg
}

// NewDataMsg returns a mock data message
func NewDataMsg(channelID string) *protoext.SignedGossipMessage {
	msg, _ := protoext.NoopSign(&gproto.GossipMessage{
		Tag:     gproto.GossipMessage_CHAN_ONLY,
		Channel: []byte(channelID),
		Content: &gproto.GossipMessage_DataMsg{},
	})
	return msg
}

// MockReceivedMessage mocks the Gossip received message
type MockReceivedMessage struct {
	Message   *protoext.SignedGossipMessage
	RespondTo func(msg *gproto.GossipMessage)
	Member    discovery.NetworkMember
}

// Respond responds to the given request
func (m *MockReceivedMessage) Respond(msg *gproto.GossipMessage) {
	if m.RespondTo != nil {
		m.RespondTo(msg)
	}
}

// GetGossipMessage returns the mock signed gossip message
func (m *MockReceivedMessage) GetGossipMessage() *protoext.SignedGossipMessage {
	return m.Message
}

// GetSourceEnvelope is not implemented
func (m *MockReceivedMessage) GetSourceEnvelope() *gproto.Envelope {
	panic("not implemented")
}

// GetConnectionInfo returns the connection information of the source of the message
func (m *MockReceivedMessage) GetConnectionInfo() *protoext.ConnectionInfo {
	return &protoext.ConnectionInfo{
		ID:       m.Member.PKIid,
		Endpoint: m.Member.Endpoint,
	}
}

// Ack is a noop
func (m *MockReceivedMessage) Ack(err error) {
	// Nothing to do
}
