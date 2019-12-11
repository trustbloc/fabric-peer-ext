/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pvtdatastorage

import (
	"encoding/hex"
	"testing"

	"github.com/hyperledger/fabric/core/ledger"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/pvtdatastorage/common"
)

func TestBlockPvtDataAssembler(t *testing.T) {
	dataKeyBytes := common.EncodeDataKey(&common.DataKey{})
	key := hex.EncodeToString(dataKeyBytes)

	t.Run("test decode hex string error", func(t *testing.T) {
		a := newBlockPvtDataAssembler(nil, nil, 0, 0, []string{"invalid data key"}, nil)
		data, err := a.assemble()
		require.Error(t, err)
		require.Contains(t, err.Error(), "encoding/hex: invalid byte")
		require.Nil(t, data)
	})

	t.Run("test v11 format error", func(t *testing.T) {
		a := newBlockPvtDataAssembler(nil, nil, 0, 0, []string{hex.EncodeToString([]byte("invalid format"))}, nil)
		data, err := a.assemble()
		require.Error(t, err)
		require.Nil(t, data)
	})

	t.Run("test decode key error", func(t *testing.T) {
		restoreV11Format := v11Format
		v11Format = func([]byte) (b bool, err error) { return false, nil }
		defer func() { v11Format = restoreV11Format }()

		a := newBlockPvtDataAssembler(
			nil, nil, 0, 0, []string{hex.EncodeToString([]byte("invalid format"))},
			func(*common.DataKey, ledger.PvtNsCollFilter, uint64) (b bool, err error) { return false, nil },
		)
		data, err := a.assemble()
		require.Error(t, err)
		require.Nil(t, data)
	})

	t.Run("test decode value error", func(t *testing.T) {
		restoreV11Format := v11Format
		v11Format = func([]byte) (b bool, err error) { return false, nil }
		defer func() { v11Format = restoreV11Format }()

		a := newBlockPvtDataAssembler(
			map[string][]byte{key: []byte("invalid value")}, nil, 0, 0, []string{key},
			func(*common.DataKey, ledger.PvtNsCollFilter, uint64) (b bool, err error) { return false, nil },
		)
		data, err := a.assemble()
		require.EqualError(t, err, "unexpected EOF")
		require.Nil(t, data)
	})

	t.Run("test v11RetrievePvtdata error", func(t *testing.T) {
		restoreV11Format := v11Format
		v11Format = func([]byte) (b bool, err error) { return true, nil }
		defer func() { v11Format = restoreV11Format }()

		a := newBlockPvtDataAssembler(
			map[string][]byte{"key1": []byte("value1")}, nil, 0, 0, []string{key},
			func(*common.DataKey, ledger.PvtNsCollFilter, uint64) (b bool, err error) { return false, nil },
		)
		data, err := a.assemble()
		require.Error(t, err)
		require.Contains(t, err.Error(), "decoded size from DecodeVarint is invalid")
		require.Nil(t, data)
	})

	t.Run("test checkIsExpired error", func(t *testing.T) {
		errExpected := errors.New("checkIsExpired error")

		a := newBlockPvtDataAssembler(
			nil, nil, 0, 0, []string{key},
			func(*common.DataKey, ledger.PvtNsCollFilter, uint64) (b bool, err error) { return false, errExpected },
		)
		data, err := a.assemble()
		require.EqualError(t, err, errExpected.Error())
		require.Nil(t, data)
	})

	t.Run("test checkIsExpired true", func(t *testing.T) {
		a := newBlockPvtDataAssembler(
			nil, nil, 0, 0, []string{key},
			func(*common.DataKey, ledger.PvtNsCollFilter, uint64) (b bool, err error) { return true, nil },
		)
		data, err := a.assemble()
		require.NoError(t, err)
		require.Nil(t, data)
	})
}
