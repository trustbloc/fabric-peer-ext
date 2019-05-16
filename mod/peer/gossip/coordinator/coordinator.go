/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package coordinator

import (
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	"github.com/hyperledger/fabric/protos/transientstore"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/pvtdatastore"
)

type transientStore interface {
	// PersistWithConfig stores the private write set of a transaction along with the collection config
	// in the transient store based on txid and the block height the private data was received at
	PersistWithConfig(txid string, blockHeight uint64, privateSimulationResultsWithConfig *transientstore.TxPvtReadWriteSetWithConfigInfo) error
}

// New returns a new PvtDataStore
func New(channelID string, transientStore transientStore, collDataStore storeapi.Store) *pvtdatastore.Store {
	return pvtdatastore.New(channelID, transientStore, collDataStore)
}
