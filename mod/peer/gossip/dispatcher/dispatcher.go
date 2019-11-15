/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dispatcher

import (
	extdispatcher "github.com/trustbloc/fabric-peer-ext/pkg/gossip/dispatcher"
	"github.com/trustbloc/fabric-peer-ext/pkg/resource"
)

// New returns a new Gossip message dispatcher provider
func NewProvider() *extdispatcher.Provider {
	p := extdispatcher.NewProvider()
	resource.Register(p.Initialize)
	return p
}
