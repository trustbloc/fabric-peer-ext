/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package storeprovider

import (
	pb "github.com/hyperledger/fabric-protos-go/peer"
	proto "github.com/hyperledger/fabric-protos-go/transientstore"
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	"github.com/pkg/errors"
	olapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/api"
	tdapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/api"
)

type targetStores struct {
	transientDataStore tdapi.Store
	offLedgerStore     olapi.Store
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
func (d *store) Persist(txID string, privateSimulationResultsWithConfig *proto.TxPvtReadWriteSetWithConfigInfo) error {
	if err := d.transientDataStore.Persist(txID, privateSimulationResultsWithConfig); err != nil {
		return errors.WithMessage(err, "error persisting transient data")
	}

	if err := d.offLedgerStore.Persist(txID, privateSimulationResultsWithConfig); err != nil {
		return errors.WithMessage(err, "error persisting off-ledger data")
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

// GetData gets the value for the given key
func (d *store) GetData(key *storeapi.Key) (*storeapi.ExpiringValue, error) {
	return d.offLedgerStore.GetData(key)
}

// PutData stores the key/value.
func (d *store) PutData(config *pb.StaticCollectionConfig, key *storeapi.Key, value *storeapi.ExpiringValue) error {
	return d.offLedgerStore.PutData(config, key, value)
}

// GetDataMultipleKeys gets the values for multiple keys in a single call
func (d *store) GetDataMultipleKeys(key *storeapi.MultiKey) (storeapi.ExpiringValues, error) {
	return d.offLedgerStore.GetDataMultipleKeys(key)
}

// Query executes the given rich query against the off-ledger store
func (d *store) Query(key *storeapi.QueryKey) (storeapi.ResultsIterator, error) {
	return d.offLedgerStore.Query(key)
}

// Close closes all of the stores store
func (d *store) Close() {
	d.transientDataStore.Close()
	d.offLedgerStore.Close()
}
