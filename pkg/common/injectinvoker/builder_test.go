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

func TestBuilder_Build(t *testing.T) {
	b := NewBuilder().Add(creator1)

	resources, err := b.Build(
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
	require.Equal(t, 1, len(resources))

	r1, ok := resources[0].(*sysCC)
	require.True(t, ok)
	require.Equal(t, ccName1, r1.Name())
	require.Equal(t, ccPath1, r1.Path())
	require.Len(t, r1.InitArgs(), 2)
}

func TestBuilder_BuildError(t *testing.T) {
	t.Run("Invalid number of return args -> error", func(t *testing.T) {
		_, err := NewBuilder().Add(NoRetArgs).Build(&nameAndPathProviderImpl{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "expecting exactly one return value")
	})

	t.Run("Dependency not found -> error", func(t *testing.T) {
		_, err := NewBuilder().Add(UnknownProvider).Build(&nameAndPathProviderImpl{})
		require.Error(t, err)
		require.Equal(t, ErrProviderNotFound, errors.Cause(err))
	})
}

type st2 interface {
	Func4(arg int64) string
}

func creator1(np nameProvider, pp pathProvider, argsp argsProvider) *sysCC {
	return &sysCC{
		name:     np.GetValue1(),
		path:     pp.GetValue2(),
		initArgs: argsp.GetValue3(),
	}
}

func NoRetArgs() {
}

func InvalidRetArgs() error {
	return nil
}

func UnknownProvider(customProvider) *sysCC {
	return &sysCC{}
}

type sysCC struct {
	name     string
	path     string
	initArgs [][]byte
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
