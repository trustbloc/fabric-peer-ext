/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"io"

	"github.com/bluele/gcache"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/pkg/errors"

	olclient "github.com/trustbloc/fabric-peer-ext/pkg/collections/client"
)

var logger = flogging.MustGetLogger("ext_offledger")

// DCAS defines the functions of a DCAS client
type DCAS interface {
	// Put puts the data and returns the content ID (CID) for the value
	Put(data io.Reader) (string, error)

	// Delete deletes the values for the given content IDs.
	Delete(cids ...string) error

	// Get retrieves the value for the given content ID (CID).
	Get(cid string, w io.Writer) error
}

// Provider manages multiple clients - one per channel
type Provider struct {
	cache gcache.Cache
}

// NewProvider returns a new client provider
func NewProvider(providers *olclient.Providers) *Provider {
	logger.Infof("Creating DCAS client provider.")
	return &Provider{
		cache: gcache.New(0).LoaderFunc(func(key interface{}) (i interface{}, e error) {
			return newClient(key.(cacheKey), providers)
		}).Build(),
	}
}

// GetDCASClient returns the client for the given channel
func (p *Provider) GetDCASClient(channelID string, namespace string, coll string) (DCAS, error) {
	if channelID == "" || namespace == "" || coll == "" {
		return nil, errors.New("channel ID, ns, and collection must be specified")
	}

	c, err := p.cache.Get(newCacheKey(channelID, namespace, coll))
	if err != nil {
		return nil, err
	}

	return c.(DCAS), nil
}

func newClient(key cacheKey, p *olclient.Providers) (*DCASClient, error) {
	logger.Debugf("Creating client for channel [%s], namespace [%s], and collection [%s]", key.channelID, key.namespace, key.collection)

	l := p.LedgerProvider.GetLedger(key.channelID)
	if l == nil {
		return nil, errors.Errorf("no ledger for channel [%s]", key.channelID)
	}

	return New(key.channelID, key.namespace, key.collection,
		&olclient.ChannelProviders{
			Ledger:           l,
			Distributor:      p.GossipProvider.GetGossipService(),
			ConfigRetriever:  p.ConfigProvider.ForChannel(key.channelID),
			IdentityProvider: p.IdentityProvider,
		},
	), nil
}

type cacheKey struct {
	channelID  string
	namespace  string
	collection string
}

func newCacheKey(channelID, ns, coll string) cacheKey {
	return cacheKey{
		channelID:  channelID,
		namespace:  ns,
		collection: coll,
	}
}
