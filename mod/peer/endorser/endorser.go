/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package endorser

import (
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	extendorser "github.com/trustbloc/fabric-peer-ext/pkg/endorser"
)

// FilterPubSimulationResults filters the public simulation results and returns the filtered results or error.
func FilterPubSimulationResults(collConfigs map[string]*common.CollectionConfigPackage, pubSimulationResults *rwset.TxReadWriteSet) (*rwset.TxReadWriteSet, error) {
	return extendorser.FilterPubSimulationResults(collConfigs, pubSimulationResults)
}
