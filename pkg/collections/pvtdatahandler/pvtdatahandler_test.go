/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pvtdatahandler

import (
	"testing"

	"github.com/hyperledger/fabric/protos/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

const (
	channelID = "testchannel"
	tx1       = "tx1"
	ns1       = "ns1"
	key1      = "key1"
	key2      = "key2"
)

func TestHandler_HandleGetPrivateData(t *testing.T) {
	storeProvider := mocks.NewDataProvider()

	h := New(channelID, storeProvider)
	require.NotNil(t, h)

	t.Run("Transient Data", func(t *testing.T) {
		config := &common.StaticCollectionConfig{
			Type: common.CollectionType_COL_TRANSIENT,
		}

		value, handled, err := h.HandleGetPrivateData(tx1, ns1, config, key1)
		assert.NoError(t, err)
		assert.True(t, handled)
		assert.NotNil(t, value)
	})
}

func TestHandler_HandleGetPrivateDataMultipleKeys(t *testing.T) {
	storeProvider := mocks.NewDataProvider()

	h := New(channelID, storeProvider)
	require.NotNil(t, h)

	t.Run("Transient Data", func(t *testing.T) {
		config := &common.StaticCollectionConfig{
			Type: common.CollectionType_COL_TRANSIENT,
		}

		value, handled, err := h.HandleGetPrivateDataMultipleKeys(tx1, ns1, config, []string{key1, key2})
		assert.NoError(t, err)
		assert.True(t, handled)
		assert.NotNil(t, value)
	})
}
