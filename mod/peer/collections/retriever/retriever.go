/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package retriever

import (
	extretriever "github.com/trustbloc/fabric-peer-ext/pkg/collections/retriever"
	"github.com/trustbloc/fabric-peer-ext/pkg/resource"
)

// NewProvider returns a new private data Retriever provider
func NewProvider() *extretriever.Provider {
	p := extretriever.NewProvider()
	resource.Register(p.Initialize)
	return p
}
