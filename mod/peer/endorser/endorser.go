/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package endorser

import (
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
)

// FilterPubSimulationResults filters the public simulation results and returns the filtered results or error.
// TODO: Filter out transient data R/W sets (Issue #88).
func FilterPubSimulationResults(_ map[string]*common.CollectionConfigPackage, pubSimulationResults *rwset.TxReadWriteSet) (*rwset.TxReadWriteSet, error) {
	return pubSimulationResults, nil
}
