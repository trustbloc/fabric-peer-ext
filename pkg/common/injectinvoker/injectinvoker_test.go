/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package injectinvoker

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	ccName1 = "cc1"
	ccPath1 = "/scc/cc1"
	arg1    = "arg1"
	arg2    = "arg2"
)

func TestInvoker_Invoke(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		inj := New(
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
		retVals, err := inj.Invoke(ValidFunc)
		require.NoError(t, err)
		require.Equal(t, 1, len(retVals))
		retVal := retVals[0]
		r, ok := retVal.Interface().(someIface)
		require.True(t, ok)
		require.NotNil(t, r)
		require.Equal(t, ccName1, r.GetValue1())
		require.Equal(t, ccPath1, r.GetValue2())
		require.Equal(t, 2, len(r.GetValue3()))
		require.Equal(t, arg1, string(r.GetValue3()[0]))
		require.Equal(t, arg2, string(r.GetValue3()[1]))
	})

	t.Run("Invalid arg -> error", func(t *testing.T) {
		inj := New(&nameAndPathProviderImpl{})
		sccDesc, err := inj.Invoke(InvalidArg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "not an interface")
		require.Nil(t, sccDesc)
	})

	t.Run("Dependency not found -> error", func(t *testing.T) {
		inj := New(&nameAndPathProviderImpl{})
		sccDesc, err := inj.Invoke(InvalidProvider)
		require.Error(t, err)
		require.Contains(t, err.Error(), "No provider found that satisfies interface")
		require.Nil(t, sccDesc)
	})
}

type nameAndPathProviderImpl struct {
	name string
	path string
}

func (s *nameAndPathProviderImpl) GetValue1() string {
	return s.name
}

func (s *nameAndPathProviderImpl) GetValue2() string {
	return s.path
}

type argsProviderImpl struct {
	args [][]byte
}

func (s *argsProviderImpl) GetValue3() [][]byte {
	return s.args
}

type nameProvider interface {
	GetValue1() string
}

type pathProvider interface {
	GetValue2() string
}

type argsProvider interface {
	GetValue3() [][]byte
}

func ValidFunc(np nameProvider, pp pathProvider, argsp argsProvider) someIface {
	return &someImpl{
		name:     np.GetValue1(),
		path:     pp.GetValue2(),
		initArgs: argsp.GetValue3(),
	}
}

type customProvider interface {
	GetData(arg int) string
}

type someIface interface {
	GetValue1() string
	GetValue2() string
	GetValue3() [][]byte
}

func InvalidProvider(customProvider) someIface {
	return &someImpl{}
}

func InvalidArg(string) someIface {
	return &someImpl{}
}

type someImpl struct {
	name     string
	path     string
	initArgs [][]byte
}

func (s *someImpl) GetValue1() string {
	return s.name
}

func (s *someImpl) GetValue2() string {
	return s.path
}

func (s *someImpl) GetValue3() [][]byte {
	return s.initArgs
}
