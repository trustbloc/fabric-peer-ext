/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package endorser

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/pkg/errors"
)

var endorserLogger = flogging.MustGetLogger("ext_endorser")

// FilterPubSimulationResults filters out all off-ledger (including transient data) read-write sets from the simulation results
// so that they won't be included in the block.
func FilterPubSimulationResults(collConfigs map[string]*common.CollectionConfigPackage, pubSimulationResults *rwset.TxReadWriteSet) (*rwset.TxReadWriteSet, error) {
	if collConfigs != nil {
		// Filter out all off-ledger hashed read/write sets
		return newFilter(collConfigs).filter(pubSimulationResults)
	}

	endorserLogger.Debugf("No collection r/w sets.")
	return pubSimulationResults, nil
}

type collRWSetFilter struct {
	collConfigs map[string]*common.CollectionConfigPackage
}

func newFilter(collConfigs map[string]*common.CollectionConfigPackage) *collRWSetFilter {
	return &collRWSetFilter{
		collConfigs: collConfigs,
	}
}

func (f *collRWSetFilter) filter(pubSimulationResults *rwset.TxReadWriteSet) (*rwset.TxReadWriteSet, error) {
	endorserLogger.Debugf("Filtering off-ledger collection types...")
	filteredResults := &rwset.TxReadWriteSet{
		DataModel: pubSimulationResults.DataModel,
	}

	// Filter out off-ledger collections from read/write sets
	for _, rwSet := range pubSimulationResults.NsRwset {
		endorserLogger.Debugf("Checking chaincode [%s] for off-ledger collection types...", rwSet.Namespace)

		filteredRWSet, err := f.filterNamespace(rwSet)
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

func (f *collRWSetFilter) filterNamespace(nsRWSet *rwset.NsReadWriteSet) (*rwset.NsReadWriteSet, error) {
	var filteredCollRWSets []*rwset.CollectionHashedReadWriteSet
	for _, collRWSet := range nsRWSet.CollectionHashedRwset {
		endorserLogger.Debugf("Checking collection [%s:%s] to see if it is an off-ledger type...", nsRWSet.Namespace, collRWSet.CollectionName)
		offLedger, err := f.isOffLedger(nsRWSet.Namespace, collRWSet.CollectionName)
		if err != nil {
			return nil, err
		}
		if !offLedger {
			endorserLogger.Debugf("... adding hashed rw-set for collection [%s:%s] since it IS NOT an off-ledger type", nsRWSet.Namespace, collRWSet.CollectionName)
			filteredCollRWSets = append(filteredCollRWSets, collRWSet)
		} else {
			endorserLogger.Debugf("... removing hashed rw-set for collection [%s:%s] since it IS an off-ledger type", nsRWSet.Namespace, collRWSet.CollectionName)
		}
	}

	return &rwset.NsReadWriteSet{
		Namespace:             nsRWSet.Namespace,
		Rwset:                 nsRWSet.Rwset,
		CollectionHashedRwset: filteredCollRWSets,
	}, nil
}

func (f *collRWSetFilter) isOffLedger(ns, coll string) (bool, error) {
	collConfig, ok := f.collConfigs[ns]
	if !ok {
		return false, errors.Errorf("config for collection [%s:%s] not found", ns, coll)
	}

	for _, config := range collConfig.Config {
		staticConfig := config.GetStaticCollectionConfig()
		if staticConfig == nil {
			return false, errors.Errorf("config for collection [%s:%s] not found", ns, coll)
		}
		if staticConfig.Name == coll {
			return isCollOffLedger(staticConfig), nil
		}
	}

	return false, errors.Errorf("config for collection [%s:%s] not found", ns, coll)
}

func isCollOffLedger(collConfig *common.StaticCollectionConfig) bool {
	return collConfig.Type == common.CollectionType_COL_TRANSIENT ||
		collConfig.Type == common.CollectionType_COL_OFFLEDGER ||
		collConfig.Type == common.CollectionType_COL_DCAS
}
