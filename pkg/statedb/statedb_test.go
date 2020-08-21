/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package statedb

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

//go:generate counterfeiter -o ../mocks/statedb.gen.go --fake-name StateDB . StateDB

func TestProvider(t *testing.T) {
	db := &mocks.StateDB{}

	require.NotPanics(t, func() { GetProvider().Register("channel1", db) })

	require.True(t, GetProvider().StateDBForChannel("channel1") == db)
}
