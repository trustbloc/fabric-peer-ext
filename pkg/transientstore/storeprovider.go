/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transientstore

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/transientstore"
)

var logger = flogging.MustGetLogger("transientstore")

// Provider represents a trasientstore provider
type Provider struct {
}

// NewStoreProvider instantiates a transient data storage provider backed by Memory
func NewStoreProvider() *Provider {
	logger.Debugf("constructing transient mem data storage provider")
	return &Provider{}
}

// OpenStore creates a handle to the transient data store for the given ledger ID
func (p *Provider) OpenStore(ledgerid string) (transientstore.Store, error) {
	return newStore(), nil
}

// Close cleans up the provider
func (p *Provider) Close() {
}
