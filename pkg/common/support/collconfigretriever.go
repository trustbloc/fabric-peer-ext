/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package support

import (
	"fmt"
	"reflect"

	"github.com/bluele/gcache"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/core/ledger"
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/pkg/errors"
)

type peerLedger interface {
	// NewQueryExecutor gives handle to a query executor.
	// A client can obtain more than one 'QueryExecutor's for parallel execution.
	// Any synchronization should be performed at the implementation level if required
	NewQueryExecutor() (ledger.QueryExecutor, error)
}

// CollectionConfigRetriever loads and caches collection configuration and policies
type CollectionConfigRetriever struct {
	channelID string
	ledger    peerLedger
	cache     gcache.Cache
}

type blockPublisher interface {
	AddCCUpgradeHandler(handler gossipapi.ChaincodeUpgradeHandler)
}

// NewCollectionConfigRetriever returns a new collection configuration retriever
func NewCollectionConfigRetriever(channelID string, ledger peerLedger, blockPublisher blockPublisher) *CollectionConfigRetriever {
	r := &CollectionConfigRetriever{
		channelID: channelID,
		ledger:    ledger,
	}

	r.cache = gcache.New(0).Simple().LoaderFunc(
		func(key interface{}) (interface{}, error) {
			ccID := key.(string)
			configs, err := r.loadConfigAndPolicy(ccID)
			if err != nil {
				logger.Debugf("Error loading collection configs for chaincode [%s]: %s", ccID, err)
				return nil, err
			}
			return configs, nil
		}).Build()

	// Add a handler to remove the collection configs from cache when the chaincode is upgraded
	blockPublisher.AddCCUpgradeHandler(func(blockNum uint64, txID string, chaincodeName string) error {
		if r.cache.Remove(chaincodeName) {
			logger.Infof("Chaincode [%s] was upgraded. Removed collection configs from cache.", chaincodeName)
		}
		return nil
	})

	return r
}

type cacheItem struct {
	config *common.StaticCollectionConfig
	policy privdata.CollectionAccessPolicy
}

type cacheItems []*cacheItem

func (c cacheItems) get(coll string) (*cacheItem, error) {
	for _, item := range c {
		if item.config.Name == coll {
			return item, nil
		}
	}
	return nil, errors.Errorf("configuration not found for collection [%s]", coll)
}

func (c cacheItems) config(coll string) (*common.StaticCollectionConfig, error) {
	item, err := c.get(coll)
	if err != nil {
		return nil, err
	}
	return item.config, nil
}

func (c cacheItems) policy(coll string) (privdata.CollectionAccessPolicy, error) {
	item, err := c.get(coll)
	if err != nil {
		return nil, err
	}
	return item.policy, nil
}

// Config returns the configuration for the given collection
func (s *CollectionConfigRetriever) Config(ns, coll string) (*common.StaticCollectionConfig, error) {
	logger.Debugf("[%s] Retrieving collection configuration for chaincode [%s]", s.channelID, ns)
	item, err := s.cache.Get(ns)
	if err != nil {
		return nil, err
	}

	configs, ok := item.(cacheItems)
	if !ok {
		panic(fmt.Sprintf("unexpected type in cache: %s", reflect.TypeOf(item)))
	}

	return configs.config(coll)
}

// Policy returns the collection access policy
func (s *CollectionConfigRetriever) Policy(ns, coll string) (privdata.CollectionAccessPolicy, error) {
	logger.Debugf("[%s] Retrieving collection policy for chaincode [%s]", s.channelID, ns)
	item, err := s.cache.Get(ns)
	if err != nil {
		return nil, err
	}

	configs, ok := item.(cacheItems)
	if !ok {
		panic(fmt.Sprintf("unexpected type in cache: %s", reflect.TypeOf(item)))
	}

	return configs.policy(coll)
}

func (s *CollectionConfigRetriever) loadConfigAndPolicy(ns string) (cacheItems, error) {
	configs, err := s.loadConfigs(ns)
	if err != nil {
		return nil, err
	}

	var items []*cacheItem
	for _, config := range configs {
		policy, err := s.loadPolicy(ns, config)
		if err != nil {
			return nil, err
		}
		items = append(items, &cacheItem{
			config: config,
			policy: policy,
		})
	}

	return items, nil
}

func (s *CollectionConfigRetriever) loadConfigs(ns string) ([]*common.StaticCollectionConfig, error) {
	logger.Debugf("[%s] Loading collection configs for chaincode [%s]", s.channelID, ns)

	cpBytes, err := s.getCCPBytes(ns)
	if err != nil {
		return nil, errors.Wrapf(err, "error retrieving collection config for chaincode [%s]", ns)
	}
	if cpBytes == nil {
		return nil, errors.Errorf("no collection config for chaincode [%s]", ns)
	}

	cp := &common.CollectionConfigPackage{}
	err = proto.Unmarshal(cpBytes, cp)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid collection configuration for [%s]", ns)
	}

	var configs []*common.StaticCollectionConfig
	for _, collConfig := range cp.Config {
		config := collConfig.GetStaticCollectionConfig()
		logger.Debugf("[%s] Checking collection config for [%s:%+v]", s.channelID, ns, config)
		if config == nil {
			logger.Warningf("[%s] No config found for a collection in namespace [%s]", s.channelID, ns)
			continue
		}
		configs = append(configs, config)
	}

	return configs, nil
}

func (s *CollectionConfigRetriever) loadPolicy(ns string, config *common.StaticCollectionConfig) (privdata.CollectionAccessPolicy, error) {
	logger.Debugf("[%s] Loading collection policy for [%s:%s]", s.channelID, ns, config.Name)

	colAP := &privdata.SimpleCollection{}
	err := colAP.Setup(config, mspmgmt.GetIdentityDeserializer(s.channelID))
	if err != nil {
		return nil, errors.Wrapf(err, "error setting up collection policy %s", config.Name)
	}

	return colAP, nil
}

func (s *CollectionConfigRetriever) getCCPBytes(ns string) ([]byte, error) {
	qe, err := s.ledger.NewQueryExecutor()
	if err != nil {
		return nil, err
	}
	defer qe.Done()

	return qe.GetState("lscc", privdata.BuildCollectionKVSKey(ns))
}
