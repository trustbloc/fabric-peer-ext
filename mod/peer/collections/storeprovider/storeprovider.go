/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package storeprovider

import (
	extstoreprovider "github.com/trustbloc/fabric-peer-ext/pkg/collections/storeprovider"
	"github.com/trustbloc/fabric-peer-ext/pkg/resource"
)

// NewProviderFactory returns a new store provider factory
func NewProviderFactory() *extstoreprovider.StoreProvider {
	p := extstoreprovider.New()
	resource.Register(p.Initialize, resource.PriorityAboveNormal)
	return p
}
