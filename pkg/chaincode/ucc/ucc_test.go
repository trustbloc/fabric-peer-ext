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

	v1     = "v1"
	v1_0_1 = "v1.0.1"
	v2     = "v2"
	v2_1_1 = "v2.1.1"

	channel1 = "channel1"
)

func TestNew(t *testing.T) {
	require.NotNil(t, newRegistry())
}

func TestRegister(t *testing.T) {
	Register(newTestCC1V1)
	Register(newTestCC1V2)
	Register(newTestCC2V1)

	require.NoError(t, resource.Mgr.Initialize())

	WaitForReady()

	require.Len(t, Chaincodes(), 3)

	cc, ok := Get(cc1, v1)
	require.True(t, ok)
	require.NotNil(t, cc)
	require.Equal(t, cc1, cc.Name())
	require.Equal(t, v1, cc.Version())

	cc, ok = Get(cc1, v1_0_1)
	require.True(t, ok)
	require.NotNil(t, cc)
	require.Equal(t, cc1, cc.Name())
	require.Equal(t, v1, cc.Version())

	cc, ok = Get(cc1, v2)
	require.True(t, ok)
	require.NotNil(t, cc)
	require.Equal(t, cc1, cc.Name())
	require.Equal(t, v2, cc.Version())

	cc, ok = Get(cc2, v1)
	require.True(t, ok)
	require.NotNil(t, cc)
	require.Equal(t, cc2, cc.Name())
	require.Equal(t, v1, cc.Version())

	noCC, ok := Get(cc1, v2_1_1)
	require.False(t, ok)
	require.Nil(t, noCC)

	resource.Mgr.ChannelJoined(channel1)

	time.Sleep(50 * time.Millisecond)

	_, exists := cc.(*testCC).Channels()[channel1]
	require.True(t, exists)
}

func TestRegister_InvalidVersion(t *testing.T) {
	inst := newRegistry()
	errExpected := "validation error for in-process user chaincode [testcc2:v2.1.1]: version must only have a major and optional minor part (e.g. v1 or v1.1)"
	require.EqualError(t, inst.register(newTestCC2V211()), errExpected)
}

func TestRegister_Duplicate(t *testing.T) {
	inst := newRegistry()
	require.NoError(t, inst.register(newTestCC1V1()))

	err := inst.register(newTestCC1V1())
	require.Error(t, err)
	require.Contains(t, err.Error(), "chaincode already registered")
}

type testCC struct {
	name     string
	version  string
	channels map[string]struct{}
	mutex    sync.RWMutex
}

func newTestCC1V1() *testCC {
	return &testCC{
		name:     cc1,
		version:  v1,
		channels: make(map[string]struct{}),
	}
}

func newTestCC1V2() *testCC {
	return &testCC{
		name:     cc1,
		version:  v2,
		channels: make(map[string]struct{}),
	}
}

func newTestCC2V1() *testCC {
	return &testCC{
		name:     cc2,
		version:  v1,
		channels: make(map[string]struct{}),
	}
}

func newTestCC2V211() *testCC {
	return &testCC{
		name:     cc2,
		version:  v2_1_1,
		channels: make(map[string]struct{}),
	}
}

func (cc *testCC) Name() string                                { return cc.name }
func (cc *testCC) Version() string                             { return cc.version }
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
