/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package scc

import (
	"sync"
	"testing"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

const (
	channel1 = "channel1"
)

func TestCreateSCC(t *testing.T) {
	t.Run("No dependencies -> Success", func(t *testing.T) {
		instance = newRegistry()
		Register(newSCCWithNoDependencies)

		descs := Create()
		require.NotNil(t, descs)
	})

	t.Run("Dependencies not satisfied -> panic", func(t *testing.T) {
		instance = newRegistry()
		Register(newSCCWithDependencies)

		require.Panics(t, func() {
			Create()
		})
	})

	t.Run("Dependencies satisfied -> Success", func(t *testing.T) {
		instance = newRegistry()
		Register(newSCCWithDependencies)

		descs := Create(mocks.NewQueryExecutorProvider())
		require.NotNil(t, descs)
	})
}

func TestRegistry(t *testing.T) {
	t.Run("No dependencies -> Success", func(t *testing.T) {
		r := newRegistry()

		cc := newSCCWithNoDependencies()
		require.NoError(t, r.register(cc))
		require.EqualError(t, r.register(cc), "system chaincode already registered: [testscc]")

		require.True(t, r == r.Initialize())
		r.ChannelJoined(channel1)

		time.Sleep(100 * time.Millisecond)

		channels := cc.joinedChannels()
		require.Len(t, channels, 1)
		require.Equal(t, channel1, channels[0])
	})
}

type testSCC struct {
	mutex    sync.Mutex
	channels map[string]struct{}
}

func newSCCWithNoDependencies() *testSCC {
	return &testSCC{
		channels: make(map[string]struct{}),
	}
}

type queryExecutorProvider interface {
	GetQueryExecutorForLedger(cid string) (ledger.QueryExecutor, error)
}

func newSCCWithDependencies(qeProvider queryExecutorProvider) *testSCC {
	return &testSCC{}
}

func (scc *testSCC) Name() string              { return "testscc" }
func (scc *testSCC) Chaincode() shim.Chaincode { return scc }

func (scc *testSCC) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

func (scc *testSCC) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

func (scc *testSCC) ChannelJoined(channelID string) {
	scc.mutex.Lock()
	scc.channels[channelID] = struct{}{}
	scc.mutex.Unlock()
}

func (scc *testSCC) joinedChannels() []string {
	scc.mutex.Lock()
	defer scc.mutex.Unlock()

	var channels []string

	for ch := range scc.channels {
		channels = append(channels, ch)
	}

	return channels
}
