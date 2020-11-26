/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package appdata

import (
	"context"
	"sync"
	"testing"
	"time"

	gcommon "github.com/hyperledger/fabric/gossip/common"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/common"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/discovery"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/requestmgr"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

const (
	channel1 = "channel1"
)

var (
	org1MSPID      = "Org1MSP"
	p1Org1Endpoint = "p1.org1.com"
	p2Org1Endpoint = "p2.org1.com"
	p3Org1Endpoint = "p3.org1.com"

	org2MSPID      = "Org2MSP"
	p1Org2Endpoint = "p1.org2.com"
	p2Org2Endpoint = "p2.org2.com"
	p3Org2Endpoint = "p3.org2.com"

	org3MSPID      = "Org3MSP"
	p1Org3Endpoint = "p1.org3.com"
	p2Org3Endpoint = "p2.org3.com"
	p3Org3Endpoint = "p3.org3.com"
)

var (
	p1Org1PKIID = gcommon.PKIidType("pkiid_P1O1")
	p2Org1PKIID = gcommon.PKIidType("pkiid_P2O1")
	p3Org1PKIID = gcommon.PKIidType("pkiid_P3O1")

	p1Org2PKIID = gcommon.PKIidType("pkiid_P1O2")
	p2Org2PKIID = gcommon.PKIidType("pkiid_P2O2")
	p3Org2PKIID = gcommon.PKIidType("pkiid_P3O2")

	p1Org3PKIID = gcommon.PKIidType("pkiid_P1O3")
	p2Org3PKIID = gcommon.PKIidType("pkiid_P2O3")
	p3Org3PKIID = gcommon.PKIidType("pkiid_P3O3")
)

func TestNewRetriever(t *testing.T) {
	require.NotNil(t, NewRetriever(channel1, mocks.NewMockGossipAdapter(), 2, 2))
}

func TestRetriever(t *testing.T) {
	gossip := mocks.NewMockGossipAdapter()
	gossip.Self(org1MSPID, mocks.NewMember(p1Org1Endpoint, p1Org1PKIID)).
		Member(org1MSPID, mocks.NewMember(p2Org1Endpoint, p2Org1PKIID)).
		Member(org1MSPID, mocks.NewMember(p3Org1Endpoint, p3Org1PKIID)).
		Member(org2MSPID, mocks.NewMember(p1Org2Endpoint, p1Org2PKIID)).
		Member(org2MSPID, mocks.NewMember(p2Org2Endpoint, p2Org2PKIID)).
		Member(org2MSPID, mocks.NewMember(p3Org2Endpoint, p3Org2PKIID)).
		Member(org3MSPID, mocks.NewMember(p1Org3Endpoint, p1Org3PKIID)).
		Member(org3MSPID, mocks.NewMember(p2Org3Endpoint, p2Org3PKIID)).
		Member(org3MSPID, mocks.NewMember(p3Org3Endpoint, p3Org3PKIID))

	var req requestmgr.Request
	var mutex sync.RWMutex

	reqMgr := requestmgr.Get(channel1)

	r := NewRetrieverForTest(channel1, gossip, 2, 2,
		func() requestmgr.Request {
			mutex.Lock()
			defer mutex.Unlock()

			req = reqMgr.NewRequest()
			return req
		},
	)
	require.NotNil(t, r)

	v := make(common.Values, 3)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response := &requestmgr.Response{
		Endpoint: "peer1",
		MSPID:    "msp1",
		Data:     []byte("response"),
	}

	go func() {
		time.Sleep(100 * time.Millisecond)

		mutex.RLock()
		defer mutex.RUnlock()

		requestmgr.Get(channel1).Respond(req.ID(), response)
	}()

	request := NewRequest("dataType1", []byte("request payload"))

	values, err := r.Retrieve(ctx, request, func(response []byte) (common.Values, error) {
		return v, nil
	}, func(values common.Values) bool {
		return true
	}, WithPeerFilter(func(member *discovery.Member) bool {
		return member.MSPID == org2MSPID
	}),
	)

	require.NoError(t, err)
	require.Len(t, values, 3)
}
