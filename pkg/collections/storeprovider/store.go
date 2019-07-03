/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package storeprovider

import (
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	cb "github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/transientstore"
	"github.com/pkg/errors"
	olapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/api"
	tdapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
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
func (d *store) Persist(txID string, privateSimulationResultsWithConfig *pb.TxPvtReadWriteSetWithConfigInfo) error {
	if err := d.transientDataStore.Persist(txID, privateSimulationResultsWithConfig); err != nil {
		return errors.WithMessage(err, "error persisting transient data")
	}

	// Off-ledger data should only be persisted on committers
	if isCommitter() {
		if err := d.offLedgerStore.Persist(txID, privateSimulationResultsWithConfig); err != nil {
			return errors.WithMessage(err, "error persisting off-ledger data")
		}
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
func (d *store) PutData(config *cb.StaticCollectionConfig, key *storeapi.Key, value *storeapi.ExpiringValue) error {
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

// isCommitter may be overridden in unit tests
var isCommitter = func() bool {
	return roles.IsCommitter()
}
