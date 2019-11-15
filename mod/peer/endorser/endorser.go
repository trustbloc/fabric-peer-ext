/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package endorser

import (
	extendorser "github.com/trustbloc/fabric-peer-ext/pkg/endorser"
	"github.com/trustbloc/fabric-peer-ext/pkg/resource"
)

// NewCollRWSetFilter returns a new collection RW set filter which filters out all off-ledger (including transient data)
// read-write sets from the simulation results so that they won't be included in the block.
func NewCollRWSetFilter() *extendorser.CollRWSetFilter {
	f := extendorser.NewCollRWSetFilter()
	resource.Register(f.Initialize, resource.PriorityAboveNormal)
	return f
}
