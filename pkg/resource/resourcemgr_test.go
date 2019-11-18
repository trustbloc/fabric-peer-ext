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
	Register(creator4)
	Register(creator3)
	Register(creator2)
	Register(creator1)

	err := Mgr.Initialize(
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
	require.Len(t, Mgr.resources, 4)

	var r1 *testResource1
	var r2 *testResource2
	var r3 *testResource3
	var r4 *testResource4

	for _, r := range Mgr.resources {
		switch r := r.(type) {
		case *testResource1:
			require.Equal(t, ccName1, r.name)
			require.Equal(t, ccPath1, r.path)
			r1 = r
		case *testResource2:
			require.Equal(t, ccName1, r.name)
			require.Empty(t, r.ChannelIDs())
			r2 = r
		case *testResource3:
			require.Equal(t, ccName1+" - "+ccPath1, r.nameAndPath)
			r3 = r
		case *testResource4:
			r4 = r
		}
	}

	require.NotNil(t, r1)
	require.NotNil(t, r2)
	require.NotNil(t, r3)
	require.NotNil(t, r4)

	Mgr.ChannelJoined(channel1)
	Mgr.ChannelJoined(channel2)

	time.Sleep(100 * time.Millisecond)
	require.Len(t, r2.ChannelIDs(), 2)

	Mgr.Close()
	require.True(t, r2.closed)
}

func TestManager_InitializeError(t *testing.T) {
	t.Run("Invalid number of return args -> error", func(t *testing.T) {
		mgr := NewManager()
		require.NotNil(t, mgr)
		mgr.Register(NoRetArgs)
		err := mgr.Initialize(&nameAndPathProviderImpl{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid number of return values")
	})
	t.Run("Dependency not found -> error", func(t *testing.T) {
		mgr := NewManager()
		require.NotNil(t, mgr)
		mgr.Register(UnknownProvider)
		err := mgr.Initialize(&nameAndPathProviderImpl{})
		require.Error(t, err)
		require.Equal(t, injectinvoker.ErrProviderNotFound, errors.Cause(err))
	})
}

func TestManager_NoProvider(t *testing.T) {
	Register(creator5)

	err := Mgr.Initialize(
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
	require.Error(t, errors.Cause(err), injectinvoker.ErrProviderNotFound)
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

func creator1(np nameProvider, pp pathProvider, argsp argsProvider) *testResource1 {
	return &testResource1{
		name:     np.GetName(),
		path:     pp.GetPath(),
		initArgs: argsp.GetArgs(),
	}
}

func creator2(np nameProvider, pp pathProvider) *testResource2 {
	r := &testResource2{}
	r.name = np.GetName()
	r.path = pp.GetPath()
	return r
}

type nameAndPathProvider interface {
	GetNameAndPath() string
}

type someFunctionProvider interface {
	SomeFunction(arg string) error
}

func creator3(npp nameAndPathProvider, sfp someFunctionProvider) *testResource3 {
	err := sfp.SomeFunction("arg1")
	if err != nil {
		panic(err.Error())
	}
	return &testResource3{
		nameAndPath: npp.GetNameAndPath(),
	}
}

func creator4() *testResource4 {
	return &testResource4{}
}

type unknownProvider interface {
	SomeUnknownFunc()
}

func creator5(p unknownProvider) *testResource5 {
	return &testResource5{}
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

type testResource5 struct {
}
