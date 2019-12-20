/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ucc

import (
	"sync"
	"testing"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/extensions/chaincode/api"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/resource"
)

const (
	cc1 = "testcc1"
	cc2 = "testcc2"

	channel1 = "channel1"
)

func TestNew(t *testing.T) {
	require.NotNil(t, newRegistry())
}

func TestRegister(t *testing.T) {
	Register(newTestCC1)
	Register(newTestCC2)

	require.NoError(t, resource.Mgr.Initialize())

	WaitForReady()

	require.Len(t, Chaincodes(), 2)

	cc, ok := Get(cc1)
	require.True(t, ok)
	require.NotNil(t, cc)
	require.Equal(t, cc1, cc.Name())

	cc, ok = Get(cc2)
	require.True(t, ok)
	require.NotNil(t, cc)
	require.Equal(t, cc2, cc.Name())

	noCC, ok := Get("non-existent")
	require.False(t, ok)
	require.Nil(t, noCC)

	resource.Mgr.ChannelJoined(channel1)

	time.Sleep(50 * time.Millisecond)

	_, exists := cc.(*testCC).Channels()[channel1]
	require.True(t, exists)
}

type testCC struct {
	name     string
	channels map[string]struct{}
	mutex    sync.RWMutex
}

func newTestCC1() *testCC {
	return &testCC{
		name:     cc1,
		channels: make(map[string]struct{}),
	}
}

func newTestCC2() *testCC {
	return &testCC{
		name:     cc2,
		channels: make(map[string]struct{}),
	}
}

func (cc *testCC) Name() string                                { return cc.name }
func (cc *testCC) Chaincode() shim.Chaincode                   { return cc }
func (cc *testCC) GetDBArtifacts() map[string]*api.DBArtifacts { return nil }

func (cc *testCC) ChannelJoined(channelID string) {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	cc.channels[channelID] = struct{}{}
}

func (cc *testCC) Channels() map[string]struct{} {
	cc.mutex.RLock()
	defer cc.mutex.RUnlock()

	channels := make(map[string]struct{})
	for cid, val := range cc.channels {
		channels[cid] = val
	}
	return channels
}

func (cc *testCC) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

func (cc *testCC) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}
