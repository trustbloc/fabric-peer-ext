/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package txn

import (
	"github.com/trustbloc/fabric-peer-ext/pkg/common/discovery"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/api"
)

type endorserFilter struct {
	*discovery.Discovery
	next api.PeerFilter
}

func newEndorserFilter(discovery *discovery.Discovery, next api.PeerFilter) *endorserFilter {
	return &endorserFilter{
		Discovery: discovery,
		next:      next,
	}
}

func (f *endorserFilter) Accept(p api.Peer) bool {
	endpoints := f.GetMembers(func(m *discovery.Member) bool {
		return m.Endpoint == p.Endpoint() && m.Roles().Contains(roles.EndorserRole)
	})

	if len(endpoints) == 0 {
		logger.Debugf("Peer [%s] is NOT an endorsing peer for channel [%s]", p.Endpoint(), f.ChannelID())

		return false
	}

	logger.Debugf("Peer [%s] is an endorsing peer for channel [%s]", p.Endpoint(), f.ChannelID())

	if f.next != nil {
		logger.Debugf("Invoking next filter for peer [%s]", p.Endpoint())

		return f.next.Accept(p)
	}

	return true
}
