/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transientstore

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/transientstore"
	storageapi "github.com/hyperledger/fabric/extensions/storage/api"

	"github.com/trustbloc/fabric-peer-ext/pkg/config"
	exttransientstore "github.com/trustbloc/fabric-peer-ext/pkg/transientstore"
)

var logger = flogging.MustGetLogger("ext_storage")

// Provider is a transient store provider
type ProviderImpl struct {
	provider transientstore.StoreProvider
}

// OpenStore opens the transient store for the given ledger
func (p *ProviderImpl) OpenStore(ledgerID string) (storageapi.TransientStore, error) {
	return p.provider.OpenStore(ledgerID)
}

// Close closes all transient stores
func (p *ProviderImpl) Close() {
	p.provider.Close()
}

// NewStoreProvider returns a new store provider
func NewStoreProvider(path string) (storageapi.TransientStoreProvider, error) {
	if config.GetTransientStoreDBType() == config.MemDBType {
		logger.Info("Using in-memory transient store provider")

		return exttransientstore.NewStoreProvider(), nil
	}

	logger.Info("Using default transient store provider")

	provider, err := transientstore.NewStoreProvider(path)
	if err != nil {
		return nil, err
	}

	return &ProviderImpl{provider: provider}, nil
}
