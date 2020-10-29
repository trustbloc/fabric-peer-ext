/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dcas

import (
	"testing"

	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/require"
)

func TestGetCASKey(t *testing.T) {
	t.Run("Binary value", func(t *testing.T) {
		v1 := []byte("value1")
		k, err := GetCASKey(v1, CIDV1, cid.Raw, mh.SHA2_256)
		require.NoError(t, err)
		require.NotNil(t, k)
	})

	t.Run("Invalid multihash type", func(t *testing.T) {
		v1 := []byte("value1")
		k, err := GetCASKey(v1, CIDV1, cid.Raw, 989898)
		require.EqualError(t, err, "invalid multihash code 989898")
		require.Empty(t, k)
	})
}

func TestGetCID(t *testing.T) {
	t.Run("CID V0", func(t *testing.T) {
		v1 := []byte("value1")
		cID, err := GetCID(v1, CIDV0, cid.Raw, mh.SHA2_256)
		require.NoError(t, err)
		require.NotNil(t, cID)
		require.NoError(t, ValidateCID(cID))
	})

	t.Run("CID V1", func(t *testing.T) {
		v1 := []byte("value1")
		cID, err := GetCID(v1, CIDV1, cid.Raw, mh.SHA2_256)
		require.NoError(t, err)
		require.NotNil(t, cID)
		require.NoError(t, ValidateCID(cID))
	})

	t.Run("Unsupported CID version", func(t *testing.T) {
		v1 := []byte("value1")
		cID, err := GetCID(v1, 2, cid.Raw, mh.SHA2_256)
		require.EqualError(t, err, "unsupported CID version [2]")
		require.Empty(t, cID)
	})

	t.Run("Invalid multihash type", func(t *testing.T) {
		v1 := []byte("value1")
		cID, err := GetCID(v1, CIDV1, cid.Raw, 989898)
		require.EqualError(t, err, "invalid multihash code 989898")
		require.Empty(t, cID)
	})
}
