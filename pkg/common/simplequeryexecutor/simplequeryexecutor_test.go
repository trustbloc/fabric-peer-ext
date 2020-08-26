/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package simplequeryexecutor

import (
	"fmt"
	"testing"

	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/stretchr/testify/require"
	cmocks "github.com/trustbloc/fabric-peer-ext/pkg/common/mocks"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

//go:generate counterfeiter -o ../mocks/resultsiterator.gen.go -fake-name ResultsIterator github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb.ResultsIterator

const (
	ns1  = "ns1"
	key1 = "key1"
	key2 = "key2"
)

func TestQueryExecutor_GetState(t *testing.T) {
	db := &mocks.StateDB{}

	v1 := []byte("value1")
	db.GetStateReturns(v1, nil)

	qe := New(db)
	require.NotNil(t, qe)

	v, err := qe.GetState(ns1, key1)
	require.NoError(t, err)
	require.Equal(t, v1, v)
}

func TestQueryExecutor_GetStateRangeScanIterator(t *testing.T) {
	db := &mocks.StateDB{}

	qe := New(db)
	require.NotNil(t, qe)

	t.Run("Success", func(t *testing.T) {
		db.GetStateRangeScanIteratorReturns(nil, nil)
		it, err := qe.GetStateRangeScanIterator(ns1, key1, key2)
		require.NoError(t, err)
		require.NotNil(t, it)
	})

	t.Run("Error", func(t *testing.T) {
		errExpected := fmt.Errorf("injected error")
		db.GetStateRangeScanIteratorReturns(nil, errExpected)
		it, err := qe.GetStateRangeScanIterator(ns1, key1, key2)
		require.EqualError(t, err, errExpected.Error())
		require.Nil(t, it)
	})
}

func TestQueryExecutor_GetPrivateDataHash(t *testing.T) {
	db := &mocks.StateDB{}

	qe := New(db)
	require.NotNil(t, qe)

	t.Run("Success", func(t *testing.T) {
		v1 := []byte("v1")
		db.GetStateReturns(v1, nil)
		v, err := qe.GetPrivateDataHash(ns1, key1, key2)
		require.NoError(t, err)
		require.Equal(t, v1, v)
	})

	t.Run("Error", func(t *testing.T) {
		errExpected := fmt.Errorf("injected error")
		db.GetStateReturns(nil, errExpected)
		it, err := qe.GetPrivateDataHash(ns1, key1, key2)
		require.EqualError(t, err, errExpected.Error())
		require.Nil(t, it)
	})
}

func TestResultsItr(t *testing.T) {
	dbIt := &cmocks.ResultsIterator{}

	it := &resultsItr{
		ns:    ns1,
		dbItr: dbIt,
	}
	defer it.Close()

	t.Run("Success", func(t *testing.T) {
		dbIt.NextReturns(&statedb.VersionedKV{}, nil)
		r, err := it.Next()
		require.NoError(t, err)
		require.NotNil(t, r)
	})

	t.Run("Nil result", func(t *testing.T) {
		dbIt.NextReturns(nil, nil)
		r, err := it.Next()
		require.NoError(t, err)
		require.Nil(t, r)
	})

	t.Run("Error", func(t *testing.T) {
		errExpected := fmt.Errorf("injected error")
		dbIt.NextReturns(nil, errExpected)
		r, err := it.Next()
		require.EqualError(t, err, errExpected.Error())
		require.Nil(t, r)
	})
}
