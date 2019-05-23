/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package coordinator

import (
	"testing"

	"github.com/hyperledger/fabric/protos/transientstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

const (
	channelID = "testchannel"

	ns1 = "ns1"

	coll1 = "coll1"
	coll2 = "coll2"

	policy1 = "OR('Org1MSP.member','Org2MSP.member')"
)

func TestCoordinator_StorePvtData(t *testing.T) {
	tStore := &mockTransientStore{}
	collStore := mocks.NewDataStore()

	c := New(channelID, tStore, collStore)
	require.NotNil(t, c)

	b := mocks.NewPvtReadWriteSetBuilder()
	nsBuilder := b.Namespace(ns1)
	nsBuilder.Collection(coll1).StaticConfig(policy1, 2, 5, 1000)
	nsBuilder.Collection(coll2).TransientConfig(policy1, 2, 5, "1m")

	err := c.StorePvtData("tx1", b.Build(), 1000)
	assert.NoError(t, err)
}

type mockTransientStore struct {
}

func (m *mockTransientStore) PersistWithConfig(txid string, blockHeight uint64, privateSimulationResultsWithConfig *transientstore.TxPvtReadWriteSetWithConfigInfo) error {
	return nil
}
