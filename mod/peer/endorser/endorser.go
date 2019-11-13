/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package endorser

import (
	supportapi "github.com/hyperledger/fabric/extensions/collections/api/support"
	"github.com/hyperledger/fabric/extensions/endorser/api"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/support"
	extendorser "github.com/trustbloc/fabric-peer-ext/pkg/endorser"
)

// CollRWSetFilter filters out all off-ledger (including transient data) read-write sets from the simulation results
// so that they won't be included in the block.
type CollRWSetFilter interface {
	Filter(channelID string, pubSimulationResults *rwset.TxReadWriteSet) (*rwset.TxReadWriteSet, error)
}

// NewCollRWSetFilter returns a new collection RW set filter
func NewCollRWSetFilter(api.QueryExecutorProviderFactory, api.BlockPublisherProvider) CollRWSetFilter {
	f := extendorser.NewCollRWSetFilter()

	// For now, explicitly call Initialize. After the CollRWSetFilter is registered as a resource,
	// this code should be removed since Initialize will be called by the resource manager.
	f.Initialize(&ccProvider{})

	return f
}

// The code below is temporarily added to facilitate the migration to dependency injected resources.
// This code should be removed once all resources have been converted to use dependency injection.

type ccProvider struct {
}

func (p *ccProvider) ForChannel(channelID string) supportapi.CollectionConfigRetriever {
	return support.CollectionConfigRetrieverForChannel(channelID)
}
