/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package builder

import (
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/scc"
	"github.com/stretchr/testify/require"
)

const (
	ccName1 = "cc1"
	ccPath1 = "/scc/cc1"
	arg1    = "arg1"
	arg2    = "arg2"
)

func TestSCCBuilder_Build(t *testing.T) {
	b := New().Add(creator1).Add(creator2)

	sccDescs, err := b.Build(
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
	require.Equal(t, 2, len(sccDescs))

	sccDesc := sccDescs[0]
	require.Equal(t, ccName1, sccDesc.Name())
	require.Equal(t, ccPath1, sccDesc.Path())
	require.Equal(t, 2, len(sccDesc.InitArgs()))
	require.Equal(t, arg1, string(sccDesc.InitArgs()[0]))
	require.Equal(t, arg2, string(sccDesc.InitArgs()[1]))

	sccDesc2 := sccDescs[1]
	require.Equal(t, ccName1, sccDesc2.Name())
	require.Empty(t, sccDesc2.Path())
	require.Empty(t, sccDesc2.InitArgs())
}

func TestBuilder_BuildError(t *testing.T) {
	t.Run("Invalid number of return args -> error", func(t *testing.T) {
		_, err := New().Add(NoRetArgs).Build(&nameAndPathProviderImpl{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid number of return values")
	})

	t.Run("Invalid return arg type -> error", func(t *testing.T) {
		_, err := New().Add(InvalidRetArgs).Build(&nameAndPathProviderImpl{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid return value")
	})

	t.Run("Dependency not found -> error", func(t *testing.T) {
		_, err := New().Add(UnknownProvider).Build(&nameAndPathProviderImpl{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "No provider found that satisfies interface")
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

type st2 interface {
	Func4(arg int64) string
}

func creator1(np nameProvider, pp pathProvider, argsp argsProvider) scc.SelfDescribingSysCC {
	return &sysCC{
		name:     np.GetName(),
		path:     pp.GetPath(),
		initArgs: argsp.GetArgs(),
	}
}

func creator2(np nameProvider) scc.SelfDescribingSysCC {
	return &sysCC{
		name: np.GetName(),
	}
}

func NoRetArgs() {
}

func InvalidRetArgs() error {
	return nil
}

type customProvider interface {
	GetData(arg int) string
}

func UnknownProvider(customProvider) scc.SelfDescribingSysCC {
	return &sysCC{}
}

type sysCC struct {
	name              string
	path              string
	initArgs          [][]byte
	chaincode         shim.Chaincode
	invokableExternal bool
	invokableCC2CC    bool
	enabled           bool
}

//Unique name of the system chaincode
func (s *sysCC) Name() string {
	return s.name
}

//Path to the system chaincode; currently not used
func (s *sysCC) Path() string {
	return s.path
}

//InitArgs initialization arguments to startup the system chaincode
func (s *sysCC) InitArgs() [][]byte {
	return s.initArgs
}

// Chaincode returns the underlying chaincode
func (s *sysCC) Chaincode() shim.Chaincode {
	return s.chaincode
}

// InvokableExternal keeps track of whether
// this system chaincode can be invoked
// through a proposal sent to this peer
func (s *sysCC) InvokableExternal() bool {
	return s.invokableExternal
}

// InvokableCC2CC keeps track of whether
// this system chaincode can be invoked
// by way of a chaincode-to-chaincode
// invocation
func (s *sysCC) InvokableCC2CC() bool {
	return s.invokableCC2CC
}

// Enabled a convenient switch to enable/disable system chaincode without
// having to remove entry from importsysccs.go
func (s *sysCC) Enabled() bool {
	return s.enabled
}
