/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package endorser

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/support"
)

var endorserLogger = flogging.MustGetLogger("ext_endorser")

// CollRWSetFilter filters out all off-ledger (including transient data) read-write sets from the simulation results
// so that they won't be included in the block.
type CollRWSetFilter struct {
}

// NewCollRWSetFilter returns a new collection RW set filter
func NewCollRWSetFilter() *CollRWSetFilter {
	return &CollRWSetFilter{}
}

// Filter filters out all off-ledger (including transient data) read-write sets from the simulation results
// so that they won't be included in the block.
func (f *CollRWSetFilter) Filter(channelID string, pubSimulationResults *rwset.TxReadWriteSet) (*rwset.TxReadWriteSet, error) {
	endorserLogger.Debugf("Filtering off-ledger collection types...")
	filteredResults := &rwset.TxReadWriteSet{
		DataModel: pubSimulationResults.DataModel,
	}

	// Filter out off-ledger collections from read/write sets
	for _, rwSet := range pubSimulationResults.NsRwset {
		endorserLogger.Debugf("Checking chaincode [%s] for off-ledger collection types...", rwSet.Namespace)

		filteredRWSet, err := f.filterNamespace(channelID, rwSet)
		if err != nil {
			return nil, err
		}

		if len(filteredRWSet.Rwset) > 0 || len(filteredRWSet.CollectionHashedRwset) > 0 {
			endorserLogger.Debugf("Adding rw-set for [%s]", rwSet.Namespace)
			filteredResults.NsRwset = append(filteredResults.NsRwset, filteredRWSet)
		} else {
			endorserLogger.Debugf("Not adding rw-set for [%s] since everything has been filtered out", rwSet.Namespace)
		}
	}

	return filteredResults, nil
}

func (f *CollRWSetFilter) filterNamespace(channelID string, nsRWSet *rwset.NsReadWriteSet) (*rwset.NsReadWriteSet, error) {
	var filteredCollRWSets []*rwset.CollectionHashedReadWriteSet
	for _, collRWSet := range nsRWSet.CollectionHashedRwset {
		endorserLogger.Debugf("[%s] Checking collection [%s:%s] to see if it is an off-ledger type...", channelID, nsRWSet.Namespace, collRWSet.CollectionName)
		offLedger, err := f.isOffLedger(channelID, nsRWSet.Namespace, collRWSet.CollectionName)
		if err != nil {
			return nil, err
		}
		if !offLedger {
			endorserLogger.Debugf("[%s] ... adding hashed rw-set for collection [%s:%s] since it IS NOT an off-ledger type", channelID, nsRWSet.Namespace, collRWSet.CollectionName)
			filteredCollRWSets = append(filteredCollRWSets, collRWSet)
		} else {
			endorserLogger.Debugf("[%s] ... removing hashed rw-set for collection [%s:%s] since it IS an off-ledger type", channelID, nsRWSet.Namespace, collRWSet.CollectionName)
		}
	}

	return &rwset.NsReadWriteSet{
		Namespace:             nsRWSet.Namespace,
		Rwset:                 nsRWSet.Rwset,
		CollectionHashedRwset: filteredCollRWSets,
	}, nil
}

func (f *CollRWSetFilter) isOffLedger(channelID, ns, coll string) (bool, error) {
	staticConfig, err := support.CollectionConfigRetrieverForChannel(channelID).Config(ns, coll)
	if err != nil {
		return false, err
	}
	return isCollOffLedger(staticConfig), nil
}

func isCollOffLedger(collConfig *common.StaticCollectionConfig) bool {
	return collConfig.Type == common.CollectionType_COL_TRANSIENT ||
		collConfig.Type == common.CollectionType_COL_OFFLEDGER ||
		collConfig.Type == common.CollectionType_COL_DCAS
}
