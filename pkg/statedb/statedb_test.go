/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package statedb

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
	statedbmocks "github.com/trustbloc/fabric-peer-ext/pkg/statedb/mocks"
)

//go:generate counterfeiter -o ../mocks/statedb.gen.go --fake-name StateDB . StateDB
//go:generate counterfeiter -o ./mocks/queryexecutorprovider.gen.go --fake-name QueryExecutorProvider . QueryExecutorProvider

func TestProvider(t *testing.T) {
	db := &mocks.StateDB{}
	qep := &statedbmocks.QueryExecutorProvider{}

	require.NotPanics(t, func() { GetProvider().Register("channel1", db, qep) })

	require.True(t, GetProvider().StateDBForChannel("channel1") == db)
}
