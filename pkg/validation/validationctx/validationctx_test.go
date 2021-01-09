/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validationctx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidationCtx(t *testing.T) {
	const channelID = "channel1"

	provider := NewProvider()
	require.NotNil(t, provider)

	ctx1, err := provider.ValidationContextForBlock(channelID, 1000)
	require.NoError(t, err)
	require.NotNil(t, ctx1)

	_, err = provider.ValidationContextForBlock(channelID, 1000)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to create context for block 1000 since it is less than or equal to the current block")

	ctx2, _ := provider.ValidationContextForBlock(channelID, 1001)
	require.NotNil(t, ctx2)

	ctx3, _ := provider.ValidationContextForBlock(channelID, 1002)
	require.NotNil(t, ctx3)

	go provider.CancelBlockValidation(channelID, 1001)

	select {
	case <-ctx1.Done():
		t.Log("Context1 is done")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timed out waiting for cancel")
	}

	select {
	case <-ctx2.Done():
		t.Log("Context2 is done")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timed out waiting for cancel")
	}
}
