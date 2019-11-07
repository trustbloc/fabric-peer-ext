/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package injectinvoker

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

const (
	ccName1 = "cc1"
	ccPath1 = "/scc/cc1"
	arg1    = "arg1"
	arg2    = "arg2"
)

func TestInvoker_Invoke(t *testing.T) {
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

	t.Run("Success", func(t *testing.T) {
		retVals, err := inj.Invoke(ValidFunc)
		require.NoError(t, err)
		require.Equal(t, 1, len(retVals))
		retVal := retVals[0]
		r, ok := retVal.Interface().(SomeIface)
		require.True(t, ok)
		require.NotNil(t, r)
		require.Equal(t, ccName1, r.GetValue1())
		require.Equal(t, ccPath1, r.GetValue2())
		require.Equal(t, 2, len(r.GetValue3()))
		require.Equal(t, arg1, string(r.GetValue3()[0]))
		require.Equal(t, arg2, string(r.GetValue3()[1]))
	})

	t.Run("Success with struct", func(t *testing.T) {
		retVals, err := inj.Invoke(ValidFuncWithStruct)
		require.NoError(t, err)
		require.Equal(t, 2, len(retVals))

		retVal1 := retVals[0]
		r1, ok := retVal1.Interface().(SomeIface)
		require.True(t, ok)
		require.NotNil(t, r1)
		require.Equal(t, ccName1, r1.GetValue1())
		require.Equal(t, ccPath1, r1.GetValue2())
		require.Equal(t, 2, len(r1.GetValue3()))
		require.Equal(t, arg1, string(r1.GetValue3()[0]))
		require.Equal(t, arg2, string(r1.GetValue3()[1]))

		retVal2 := retVals[1]
		r2, ok := retVal2.Interface().(SomeOtherIface)
		require.True(t, ok)
		require.NotNil(t, r2)
		require.Equal(t, ccName1, r2.GetValue4())
	})

	t.Run("Success with nested struct pointer", func(t *testing.T) {
		retVals, err := inj.Invoke(ValidFuncWithNestedStructPtr)
		require.NoError(t, err)
		require.Equal(t, 2, len(retVals))

		retVal1 := retVals[0]
		r1, ok := retVal1.Interface().(SomeIface)
		require.True(t, ok)
		require.NotNil(t, r1)
		require.Equal(t, ccName1, r1.GetValue1())
		require.Equal(t, ccPath1, r1.GetValue2())
		require.Equal(t, 2, len(r1.GetValue3()))
		require.Equal(t, arg1, string(r1.GetValue3()[0]))
		require.Equal(t, arg2, string(r1.GetValue3()[1]))

		retVal2 := retVals[1]
		r2, ok := retVal2.Interface().(SomeOtherIface)
		require.True(t, ok)
		require.NotNil(t, r2)
		require.Equal(t, ccName1, r2.GetValue4())
	})

	t.Run("Invalid arg -> error", func(t *testing.T) {
		inj := New(&nameAndPathProviderImpl{})
		ret, err := inj.Invoke(InvalidArg)
		require.Error(t, err)
		require.Equal(t, ErrUnsupportedType, errors.Cause(err))
		require.Nil(t, ret)
	})

	t.Run("Dependency not found -> error", func(t *testing.T) {
		inj := New(&nameAndPathProviderImpl{})
		ret, err := inj.Invoke(InvalidProvider)
		require.Error(t, err)
		require.Equal(t, ErrProviderNotFound, errors.Cause(err))
		require.Nil(t, ret)
	})

	t.Run("AddProvider -> error", func(t *testing.T) {
		inj := New()
		ret, err := inj.Invoke(ValidFuncWithStruct)
		require.Error(t, err)
		require.Equal(t, ErrProviderNotFound, errors.Cause(err))
		require.Nil(t, ret)

		inj.AddProvider(&nameAndPathProviderImpl{})
		inj.AddProvider(&argsProviderImpl{})
		_, err = inj.Invoke(ValidFuncWithStruct)
		require.NoError(t, err)
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

func ValidFunc(np nameProvider, pp pathProvider, argsp argsProvider) SomeIface {
	return &someImpl{
		name:     np.GetValue1(),
		path:     pp.GetValue2(),
		initArgs: argsp.GetValue3(),
	}
}

func ValidFuncWithStruct(ph ProviderHolder) (SomeIface, SomeOtherIface) {
	return &someImpl{
			name:     ph.NP.GetValue1(),
			path:     ph.PP.GetValue2(),
			initArgs: ph.ArgsP.GetValue3(),
		},
		&someOtherImpl{value4: ph.NP.GetValue1()}
}

func ValidFuncWithNestedStructPtr(nph *NestedProviderHolder) (SomeIface, SomeOtherIface) {
	return &someImpl{
			name:     nph.PH.NP.GetValue1(),
			path:     nph.PH.PP.GetValue2(),
			initArgs: nph.PH.ArgsP.GetValue3(),
		},
		&someOtherImpl{value4: nph.AnotherNP.GetValue1()}
}

type customProvider interface {
	GetData(arg int) string
}

type SomeIface interface {
	GetValue1() string
	GetValue2() string
	GetValue3() [][]byte
}

type SomeOtherIface interface {
	GetValue4() string
}

func InvalidProvider(customProvider) SomeIface {
	return &someImpl{}
}

func InvalidArg(string) SomeIface {
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

type someOtherImpl struct {
	value4 string
}

func (s *someOtherImpl) GetValue4() string {
	return s.value4
}

type ProviderHolder struct {
	NP    nameProvider
	PP    pathProvider
	ArgsP argsProvider
	pvt   string // Private field should be ignored
	f     func() // Function field should be ignored
}

type NestedProviderHolder struct {
	PH        *ProviderHolder
	AnotherNP nameProvider
}
