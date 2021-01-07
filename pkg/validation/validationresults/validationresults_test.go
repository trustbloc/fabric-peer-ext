/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validationresults

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	org1MSP = "Org1MSP"
	org2MSP = "Org2MSP"

	p1Org1 = "p1.org1.com"
	p1Org2 = "p1.org2.com"
	p2Org2 = "p2.org2.com"
)

func TestResults_String(t *testing.T) {
	t.Run("No error", func(t *testing.T) {
		r := &Results{
			BlockNumber: 1000,
			TxFlags:     []uint8{255, 0, 255, 0},
			MSPID:       org1MSP,
			Endpoint:    p1Org1,
		}
		require.Equal(t, "(MSP: [Org1MSP], Endpoint: [p1.org1.com], Block: 1000, TxFlags: [255 0 255 0])", r.String())
	})

	t.Run("With error", func(t *testing.T) {
		r := &Results{
			BlockNumber: 1000,
			Err:         fmt.Errorf("validation error"),
			MSPID:       org1MSP,
			Endpoint:    p1Org1,
		}
		require.Equal(t, "(MSP: [Org1MSP], Endpoint: [p1.org1.com], Block: 1000, Err: validation error)", r.String())
	})
}

func TestValidationResultsCache(t *testing.T) {
	cache := NewCache()

	blockNum := uint64(1000)

	r1 := &Results{
		BlockNumber: blockNum,
		TxFlags:     []uint8{255, 0, 255, 0},
		MSPID:       org1MSP,
		Endpoint:    p1Org1,
	}
	r2 := &Results{
		BlockNumber: blockNum,
		TxFlags:     []uint8{255, 0, 255, 0},
		MSPID:       org2MSP,
		Endpoint:    p2Org2,
	}
	r3 := &Results{
		BlockNumber: blockNum,
		TxFlags:     []uint8{0, 255, 0, 255},
		MSPID:       org2MSP,
		Endpoint:    p1Org2,
	}

	results := cache.Add(r1)
	require.Equal(t, 1, len(results))
	require.Equal(t, r1, results[0])

	results = cache.Add(r2)
	require.Equal(t, 2, len(results))
	require.Equal(t, r1, results[0])
	require.Equal(t, r2, results[1])

	results = cache.Add(r3)
	require.Equal(t, 1, len(results))
	require.Equal(t, r3, results[0])

	results = cache.Remove(blockNum)
	require.Equal(t, 3, len(results))

	results = cache.Remove(blockNum)
	require.Empty(t, results)
}
