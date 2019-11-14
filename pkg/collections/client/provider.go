/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"github.com/bluele/gcache"
	commonledger "github.com/hyperledger/fabric/common/ledger"
	"github.com/pkg/errors"
	collcommon "github.com/trustbloc/fabric-peer-ext/pkg/collections/common"
)

// OffLedger defines the functions of an off-ledger client
type OffLedger interface {
	// Put puts the value for the given key
	Put(ns, coll, key string, value []byte) error

	// PutMultipleValues puts the given key/values
	PutMultipleValues(ns, coll string, kvs []*KeyValue) error

	// Delete deletes the given key(s)
	Delete(ns, coll string, keys ...string) error

	// Get retrieves the value for the given key
	Get(ns, coll, key string) ([]byte, error)

	// GetMultipleKeys retrieves the values for the given keys
	GetMultipleKeys(ns, coll string, keys ...string) ([][]byte, error)

	// Query executes the given query and returns an iterator that contains results.
	// Only used for state databases that support query.
	// The returned ResultsIterator contains results of type *KV which is defined in protos/ledger/queryresult.
	Query(ns, coll, query string) (commonledger.ResultsIterator, error)
}

// Provider manages multiple clients - one per channel
type Provider struct {
	cache gcache.Cache
}

// Providers contains all of the dependencies for the client
type Providers struct {
	LedgerProvider   collcommon.LedgerProvider
	GossipAdapter    PvtDataDistributor
	ConfigProvider   collcommon.CollectionConfigProvider
	IdentityProvider collcommon.IdentityProvider
}

// NewProvider returns a new client provider
func NewProvider(providers *Providers) *Provider {
	logger.Infof("Creating collection client provider.")
	return &Provider{
		cache: gcache.New(0).LoaderFunc(func(channelID interface{}) (i interface{}, e error) {
			return newClient(channelID.(string), providers)
		}).Build(),
	}
}

// ForChannel returns the client for the given channel
func (p *Provider) ForChannel(channelID string) (OffLedger, error) {
	if channelID == "" {
		return nil, errors.New("channel ID is empty")
	}

	c, err := p.cache.Get(channelID)
	if err != nil {
		return nil, err
	}

	return c.(OffLedger), nil
}

func newClient(channelID string, p *Providers) (*Client, error) {
	logger.Debugf("Creating client for channel [%s]", channelID)

	l := p.LedgerProvider.GetLedger(channelID)
	if l == nil {
		return nil, errors.Errorf("no ledger for channel [%s]", channelID)
	}

	return New(channelID,
		&ChannelProviders{
			Ledger:           l,
			Distributor:      p.GossipAdapter,
			ConfigRetriever:  p.ConfigProvider.ForChannel(channelID),
			IdentityProvider: p.IdentityProvider,
		},
	), nil
}
