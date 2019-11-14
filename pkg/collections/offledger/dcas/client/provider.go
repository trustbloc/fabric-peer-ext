/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"github.com/bluele/gcache"
	"github.com/hyperledger/fabric/common/flogging"
	commonledger "github.com/hyperledger/fabric/common/ledger"
	"github.com/pkg/errors"
	olclient "github.com/trustbloc/fabric-peer-ext/pkg/collections/client"
)

var logger = flogging.MustGetLogger("ext_offledger")

// DCAS defines the functions of a DCAS client
type DCAS interface {
	// Put puts the DCAS value and returns the key for the value
	Put(ns, coll string, value []byte) (string, error)

	// PutMultipleValues puts the DCAS values and returns the keys for the values
	PutMultipleValues(ns, coll string, values [][]byte) ([]string, error)

	// Delete deletes the given key(s). The key(s) must be the base64-encoded hash of the value.
	Delete(ns, coll string, keys ...string) error

	// Get retrieves the value for the given key. The key must be the base64-encoded hash of the value.
	Get(ns, coll, key string) ([]byte, error)

	// GetMultipleKeys retrieves the values for the given keys. The key(s) must be the base64-encoded hash of the value.
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

// NewProvider returns a new client provider
func NewProvider(providers *olclient.Providers) *Provider {
	logger.Infof("Creating DCAS client provider.")
	return &Provider{
		cache: gcache.New(0).LoaderFunc(func(channelID interface{}) (i interface{}, e error) {
			return newClient(channelID.(string), providers)
		}).Build(),
	}
}

// ForChannel returns the client for the given channel
func (p *Provider) ForChannel(channelID string) (DCAS, error) {
	if channelID == "" {
		return nil, errors.New("channel ID is empty")
	}

	c, err := p.cache.Get(channelID)
	if err != nil {
		return nil, err
	}

	return c.(DCAS), nil
}

func newClient(channelID string, p *olclient.Providers) (*DCASClient, error) {
	logger.Debugf("Creating client for channel [%s]", channelID)

	l := p.LedgerProvider.GetLedger(channelID)
	if l == nil {
		return nil, errors.Errorf("no ledger for channel [%s]", channelID)
	}

	return New(channelID,
		&olclient.ChannelProviders{
			Ledger:           l,
			Distributor:      p.GossipAdapter,
			ConfigRetriever:  p.ConfigProvider.ForChannel(channelID),
			IdentityProvider: p.IdentityProvider,
		},
	), nil
}
