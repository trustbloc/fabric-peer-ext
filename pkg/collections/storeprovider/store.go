/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package storeprovider

import (
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	pb "github.com/hyperledger/fabric/protos/transientstore"
	"github.com/pkg/errors"
	tdapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/api"
)

type targetStores struct {
	transientDataStore tdapi.Store
}

type store struct {
	channelID string
	targetStores
}

func newDelegatingStore(channelID string, targets targetStores) *store {
	return &store{
		channelID:    channelID,
		targetStores: targets,
	}
}

// Persist persists all transient data within the private data simulation results
func (d *store) Persist(txID string, privateSimulationResultsWithConfig *pb.TxPvtReadWriteSetWithConfigInfo) error {
	if err := d.transientDataStore.Persist(txID, privateSimulationResultsWithConfig); err != nil {
		return errors.WithMessage(err, "error persisting transient data")
	}

	return nil
}

// GetTransientData returns the transient data for the given key
func (d *store) GetTransientData(key *storeapi.Key) (*storeapi.ExpiringValue, error) {
	return d.transientDataStore.GetTransientData(key)
}

// GetTransientData returns the transient data for the given keys
func (d *store) GetTransientDataMultipleKeys(key *storeapi.MultiKey) (storeapi.ExpiringValues, error) {
	return d.transientDataStore.GetTransientDataMultipleKeys(key)
}

// Close closes all of the stores store
func (d *store) Close() {
	d.transientDataStore.Close()
}
