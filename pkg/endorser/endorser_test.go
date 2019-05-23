/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package endorser

import (
	"testing"

	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

const (
	ns1 = "ns1"
	ns2 = "ns2"
	ns3 = "ns3"

	coll1 = "coll1"
	coll2 = "coll2"
)

func TestFilterPubSimulationResults(t *testing.T) {
	pvtBuilder := mocks.NewPvtReadWriteSetBuilder()
	pvtNSBuilder1 := pvtBuilder.Namespace(ns1)
	pvtNSBuilder1.Collection(coll1).StaticConfig("OR('Org1.member','Org2.member')", 2, 3, 100)
	pvtNSBuilder1.Collection(coll2).TransientConfig("OR('Org1.member','Org2.member')", 2, 3, "1m")
	pvtNSBuilder3 := pvtBuilder.Namespace(ns3)
	pvtNSBuilder3.Collection(coll2).OffLedgerConfig("OR('Org1.member','Org2.member')", 2, 3, "1m")

	builder := mocks.NewReadWriteSetBuilder()
	nsBuilder1 := builder.Namespace(ns1)
	nsBuilder1.Write("key1", []byte("value1"))
	nsBuilder1.Collection(coll1)
	nsBuilder1.Collection(coll2)

	nsBuilder2 := builder.Namespace(ns2)
	nsBuilder2.Write("key1", []byte("value1"))

	nsBuilder3 := builder.Namespace(ns3)
	nsBuilder3.Collection(coll2)

	results := builder.Build()
	require.NotNil(t, results)

	// Should be 3 namespaces
	require.Equal(t, 3, len(results.NsRwset))

	// ns1 should have a rw-set and 2 coll-hashed rw-sets
	assert.Equal(t, ns1, results.NsRwset[0].Namespace)
	require.Equal(t, 2, len(results.NsRwset[0].CollectionHashedRwset))
	assert.Equal(t, coll1, results.NsRwset[0].CollectionHashedRwset[0].CollectionName)
	assert.Equal(t, coll2, results.NsRwset[0].CollectionHashedRwset[1].CollectionName)
	assert.NotEmpty(t, len(results.NsRwset[0].Rwset))

	// ns2 should have a rw-set and no coll-hashed rw-sets
	assert.Equal(t, ns2, results.NsRwset[1].Namespace)
	require.Equal(t, 0, len(results.NsRwset[1].CollectionHashedRwset))
	assert.NotEmpty(t, len(results.NsRwset[1].Rwset))

	// ns3 should have no rw-set and 1 coll-hashed rw-sets
	assert.Equal(t, ns3, results.NsRwset[2].Namespace)
	require.Equal(t, 1, len(results.NsRwset[2].CollectionHashedRwset))
	assert.Empty(t, len(results.NsRwset[2].Rwset))

	filteredResults, err := FilterPubSimulationResults(pvtBuilder.BuildCollectionConfigs(), results)
	assert.NoError(t, err)
	require.NotNil(t, filteredResults)

	require.Equalf(t, 2, len(filteredResults.NsRwset), "expecting the rw-set for [%s] to be filtered out completely", ns1)
	assert.Equal(t, ns1, filteredResults.NsRwset[0].Namespace)
	require.Equalf(t, 1, len(filteredResults.NsRwset[0].CollectionHashedRwset), "expecting only one collection rw-set since the off-ledger rw-set should have been filtered out")
	assert.Equal(t, coll1, filteredResults.NsRwset[0].CollectionHashedRwset[0].CollectionName)
	assert.NotEmptyf(t, len(filteredResults.NsRwset[0].Rwset), "expecting rw-set to still exist for [%s]", ns1)

	assert.Equal(t, ns2, filteredResults.NsRwset[1].Namespace)
	require.Equalf(t, 0, len(filteredResults.NsRwset[1].CollectionHashedRwset), "expecting rw-set to still exist for [%s]", ns2)
	assert.NotEmpty(t, len(filteredResults.NsRwset[1].Rwset))
}

func TestFilterPubSimulationResults_NoCollections(t *testing.T) {
	var collConfigs map[string]*common.CollectionConfigPackage

	results := &rwset.TxReadWriteSet{
		DataModel: rwset.TxReadWriteSet_KV,
		NsRwset:   []*rwset.NsReadWriteSet{},
	}

	filteredResults, err := FilterPubSimulationResults(collConfigs, results)
	assert.NoError(t, err)
	assert.Equal(t, results, filteredResults)
}
