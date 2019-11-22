/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package support

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/bluele/gcache"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/common/privdata"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/extensions/collections/api/support"
	"github.com/hyperledger/fabric/extensions/endorser/api"
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/pkg/errors"
	collcommon "github.com/trustbloc/fabric-peer-ext/pkg/collections/common"
)

var logger = flogging.MustGetLogger("ext_support")

type ledgerProvider interface {
	GetLedger(cid string) ledger.PeerLedger
}

// CollectionConfigRetrieverProvider is a collection config retriever provider
type CollectionConfigRetrieverProvider struct {
	mutex                  sync.RWMutex
	retrievers             map[string]*CollectionConfigRetriever
	ledgerProvider         ledgerProvider
	blockPublisherProvider api.BlockPublisherProvider
	idProvider             collcommon.IdentityDeserializerProvider
}

// NewCollectionConfigRetrieverProvider returns a new CollectionConfigRetrieverProvider
func NewCollectionConfigRetrieverProvider(ledgerProvider ledgerProvider, blockPublisherProvider api.BlockPublisherProvider, idProvider collcommon.IdentityDeserializerProvider) *CollectionConfigRetrieverProvider {
	logger.Info("Creating collection config retriever provider")
	return &CollectionConfigRetrieverProvider{
		retrievers:             make(map[string]*CollectionConfigRetriever),
		ledgerProvider:         ledgerProvider,
		blockPublisherProvider: blockPublisherProvider,
		idProvider:             idProvider,
	}
}

// ForChannel returns the retriever for the given channel
func (rc *CollectionConfigRetrieverProvider) ForChannel(channelID string) support.CollectionConfigRetriever {
	rc.mutex.RLock()
	r, ok := rc.retrievers[channelID]
	rc.mutex.RUnlock()

	if ok {
		logger.Debugf("Returning cached collection config retriever for channel [%s]", channelID)
		return r
	}

	rc.mutex.Lock()
	defer rc.mutex.Unlock()

	r, ok = rc.retrievers[channelID]
	if !ok {
		logger.Debugf("Creating new collection config retriever for channel [%s]", channelID)
		r = newCollectionConfigRetriever(
			channelID,
			rc.ledgerProvider.GetLedger(channelID),
			rc.blockPublisherProvider.ForChannel(channelID),
			rc.idProvider.GetIdentityDeserializer(channelID),
		)
		rc.retrievers[channelID] = r
	}
	return r
}

type peerLedger interface {
	// NewQueryExecutor gives handle to a query executor.
	// A client can obtain more than one 'QueryExecutor's for parallel execution.
	// Any synchronization should be performed at the implementation level if required
	NewQueryExecutor() (ledger.QueryExecutor, error)
}

// CollectionConfigRetriever loads and caches collection configuration and policies
type CollectionConfigRetriever struct {
	channelID            string
	ledger               peerLedger
	identityDeserializer msp.IdentityDeserializer
	cache                gcache.Cache
}

type blockPublisher interface {
	AddLSCCWriteHandler(handler gossipapi.LSCCWriteHandler)
}

func newCollectionConfigRetriever(channelID string, ledger peerLedger, blockPublisher blockPublisher, identityDeserializer msp.IdentityDeserializer) *CollectionConfigRetriever {
	r := &CollectionConfigRetriever{
		channelID:            channelID,
		ledger:               ledger,
		identityDeserializer: identityDeserializer,
	}

	r.cache = gcache.New(0).Simple().LoaderFunc(
		func(key interface{}) (interface{}, error) {
			ccID := key.(string)
			logger.Infof("Collection configs for chaincode [%s] are not cached. Loading...", ccID)
			configs, err := r.loadConfigAndPolicy(ccID)
			if err != nil {
				logger.Debugf("Error loading collection configs for chaincode [%s]: %s", ccID, err)
				return nil, err
			}
			return configs, nil
		}).Build()

	// Add a handler to cache the collection config and policy when the chaincode is instantiated/upgraded
	blockPublisher.AddLSCCWriteHandler(func(txnMetadata gossipapi.TxMetadata, ccID string, ccData *ccprovider.ChaincodeData, ccp *common.CollectionConfigPackage) error {
		if ccp != nil {
			logger.Infof("Updating collection configs for chaincode [%s].", ccID)
			configs, err := r.getConfigAndPolicy(ccID, ccp)
			if err != nil {
				return errors.WithMessagef(err, "error getting collection configs for chaincode [%s]", ccID)
			}
			if err := r.cache.Set(ccID, configs); err != nil {
				return errors.WithMessagef(err, "error setting collection configs for chaincode [%s]", ccID)
			}
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
	return s.cacheItemsFromConfigs(ns, configs)
}

func (s *CollectionConfigRetriever) getConfigAndPolicy(ns string, ccp *common.CollectionConfigPackage) (cacheItems, error) {
	configs, err := s.configsFromCCP(ns, ccp)
	if err != nil {
		return nil, err
	}
	return s.cacheItemsFromConfigs(ns, configs)
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

	ccp := &common.CollectionConfigPackage{}
	err = proto.Unmarshal(cpBytes, ccp)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid collection configuration for [%s]", ns)
	}
	return s.configsFromCCP(ns, ccp)
}

func (s *CollectionConfigRetriever) cacheItemsFromConfigs(ns string, configs []*common.StaticCollectionConfig) (cacheItems, error) {
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

func (s *CollectionConfigRetriever) configsFromCCP(ns string, ccp *common.CollectionConfigPackage) ([]*common.StaticCollectionConfig, error) {
	var configs []*common.StaticCollectionConfig
	for _, collConfig := range ccp.Config {
		config := collConfig.GetStaticCollectionConfig()
		logger.Debugf("[%s] Checking collection config for [%s:%+v]", s.channelID, ns, config)
		if config == nil {
			return nil, errors.Errorf("no config found for a collection in namespace [%s]", ns)
		}
		configs = append(configs, config)
	}

	return configs, nil
}

func (s *CollectionConfigRetriever) loadPolicy(ns string, config *common.StaticCollectionConfig) (privdata.CollectionAccessPolicy, error) {
	logger.Debugf("[%s] Loading collection policy for [%s:%s]", s.channelID, ns, config.Name)

	colAP := &privdata.SimpleCollection{}
	err := colAP.Setup(config, s.identityDeserializer)
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
