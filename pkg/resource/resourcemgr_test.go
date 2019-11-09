/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package resource

import (
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/injectinvoker"
)

const (
	ccName1  = "cc1"
	ccPath1  = "/scc/cc1"
	arg1     = "arg1"
	arg2     = "arg2"
	channel1 = "channel1"
	channel2 = "channel2"
)

func TestManager_Initialize(t *testing.T) {
	mgr := NewManager()

	creators := &creators{}
	mgr.Register(creators.creator4, PriorityLow)
	mgr.Register(creators.creator3, PriorityLow)
	mgr.Register(creators.creator2, PriorityNormal)
	mgr.Register(creators.creator1, PriorityHighest)

	err := mgr.Initialize(
		&nameAndPathProviderImpl{
			name: ccName1,
			path: ccPath1,
		},
		&argsProviderImpl{
			args: [][]byte{
				[]byte(arg1),
				[]byte(arg2),
			},
		},
	)
	require.NoError(t, err)
	require.Len(t, mgr.resources, 4)

	require.Equal(t, 1, creators.creator1InitOrder)
	require.Equal(t, 2, creators.creator2InitOrder)
	require.Equal(t, 3, creators.creator4InitOrder)
	require.Equal(t, 4, creators.creator3InitOrder)

	r1, ok := mgr.resources[0].(*testResource1)
	require.True(t, ok)
	require.NotNil(t, r1)
	require.Equal(t, ccName1, r1.name)
	require.Equal(t, ccPath1, r1.path)

	r2, ok := mgr.resources[1].(*testResource2)
	require.True(t, ok)
	require.NotNil(t, r2)
	require.Equal(t, ccName1, r2.name)
	require.Empty(t, r2.ChannelIDs())

	r4, ok := mgr.resources[2].(*testResource4)
	require.True(t, ok)
	require.NotNil(t, r4)

	// testResource3 depends on the previous resources as providers
	r3, ok := mgr.resources[3].(*testResource3)
	require.True(t, ok)
	require.NotNil(t, r3)
	require.Equal(t, ccName1+" - "+ccPath1, r3.nameAndPath)

	mgr.ChannelJoined(channel1)
	mgr.ChannelJoined(channel2)

	time.Sleep(100 * time.Millisecond)
	require.Len(t, r2.ChannelIDs(), 2)

	mgr.Close()
	require.True(t, r2.closed)
}

func TestManager_InitializeError(t *testing.T) {
	t.Run("Invalid number of return args -> error", func(t *testing.T) {
		mgr := NewManager()
		require.NotNil(t, mgr)
		mgr.Register(NoRetArgs, PriorityNormal)
		err := mgr.Initialize(&nameAndPathProviderImpl{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid number of return values")
	})
	t.Run("Dependency not found -> error", func(t *testing.T) {
		mgr := NewManager()
		require.NotNil(t, mgr)
		mgr.Register(UnknownProvider, PriorityNormal)
		err := mgr.Initialize(&nameAndPathProviderImpl{})
		require.Error(t, err)
		require.Equal(t, injectinvoker.ErrProviderNotFound, errors.Cause(err))
	})
}

type nameAndPathProviderImpl struct {
	name string
	path string
}

func (s *nameAndPathProviderImpl) GetName() string {
	return s.name
}

func (s *nameAndPathProviderImpl) GetPath() string {
	return s.path
}

type argsProviderImpl struct {
	args [][]byte
}

func (s *argsProviderImpl) GetArgs() [][]byte {
	return s.args
}

type nameProvider interface {
	GetName() string
}

type pathProvider interface {
	GetPath() string
}

type argsProvider interface {
	GetArgs() [][]byte
}

type creators struct {
	order             int
	creator1InitOrder int
	creator2InitOrder int
	creator3InitOrder int
	creator4InitOrder int
}

func (c *creators) creator1(np nameProvider, pp pathProvider, argsp argsProvider) *testResource1 {
	c.order++
	c.creator1InitOrder = c.order
	return &testResource1{
		name:     np.GetName(),
		path:     pp.GetPath(),
		initArgs: argsp.GetArgs(),
	}
}

func (c *creators) creator2(np nameProvider) *testResource2 {
	c.order++
	c.creator2InitOrder = c.order
	r := &testResource2{}
	r.name = np.GetName()
	return r
}

type nameAddPathProvider interface {
	GetNameAndPath() string
}

type someFunctionProvider interface {
	SomeFunction(arg string) error
}

func (c *creators) creator3(npp nameAddPathProvider, sfp someFunctionProvider) *testResource3 {
	c.order++
	c.creator3InitOrder = c.order
	err := sfp.SomeFunction("arg1")
	if err != nil {
		panic(err.Error())
	}
	return &testResource3{
		nameAndPath: npp.GetNameAndPath(),
	}
}

func (c *creators) creator4() *testResource4 {
	c.order++
	c.creator4InitOrder = c.order
	return &testResource4{}
}

func NoRetArgs() {
}

type customProvider interface {
	GetData(arg int) string
}

func UnknownProvider(customProvider) *testResource1 {
	return &testResource1{}
}

type testResource1 struct {
	name     string
	path     string
	initArgs [][]byte
}

func (s *testResource1) Name() string {
	return s.name
}

func (s *testResource1) Path() string {
	return s.path
}

func (s *testResource1) InitArgs() [][]byte {
	return s.initArgs
}

func (s *testResource1) GetNameAndPath() string {
	return s.name + " - " + s.path
}

type testResource2 struct {
	testResource1
	closed     bool
	channelIDs []string
	mutex      sync.RWMutex
}

func (s *testResource2) Close() {
	s.closed = true
}

func (s *testResource2) ChannelJoined(channelID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.channelIDs = append(s.channelIDs, channelID)
}

func (s *testResource2) SomeFunction(arg string) error {
	return nil
}

func (s *testResource2) ChannelIDs() []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.channelIDs
}

type testResource3 struct {
	nameAndPath string
}

type testResource4 struct {
}
