/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"encoding/hex"
	"sync"

	"github.com/hyperledger/fabric/extensions/gossip/blockpublisher"

	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/common/flogging"
	commonledger "github.com/hyperledger/fabric/common/ledger"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/peer"
	gossipapi "github.com/hyperledger/fabric/extensions/gossip/api"
	"github.com/hyperledger/fabric/gossip/service"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/transientstore"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/common/support"
)

var logger = flogging.MustGetLogger("offledger")

// PeerLedger defines the ledger functions required by the client
type PeerLedger interface {
	// NewQueryExecutor gives handle to a query executor.
	// A client can obtain more than one 'QueryExecutor's for parallel execution.
	// Any synchronization should be performed at the implementation level if required
	NewQueryExecutor() (ledger.QueryExecutor, error)
	// NewTxSimulator gives handle to a transaction simulator.
	// A client can obtain more than one 'TxSimulator's for parallel execution.
	// Any snapshoting/synchronization should be performed at the implementation level if required
	NewTxSimulator(txid string) (ledger.TxSimulator, error)
	// GetBlockchainInfo returns basic info about blockchain
	GetBlockchainInfo() (*cb.BlockchainInfo, error)
}

// GossipAdapter defines the Gossip functions required by the client
type GossipAdapter interface {
	// DistributePrivateData distributes private data to the peers in the collections
	// according to policies induced by the PolicyStore and PolicyParser
	DistributePrivateData(chainID string, txID string, privateData *transientstore.TxPvtReadWriteSetWithConfigInfo, blkHt uint64) error
}

// CollectionConfigRetriever defines the collection config retrieval functions required by the client
type CollectionConfigRetriever interface {
	Config(ns, coll string) (*cb.StaticCollectionConfig, error)
}

// KeyValue holds a key-value pair
type KeyValue struct {
	Key   string
	Value []byte
}

// Client allows you to put and get Client from outside of a chaincode
type Client struct {
	channelID       string
	ledger          PeerLedger
	gossip          GossipAdapter
	configRetriever CollectionConfigRetriever
	creator         []byte
	mutex           sync.RWMutex
}

// New returns a new client
func New(channelID string) (*Client, error) {
	ledger := getLedger(channelID)
	if ledger == nil {
		return nil, errors.Errorf("ledger not found for channel [%s]", channelID)
	}

	blockPublisher := getBlockPublisher(channelID)

	return &Client{
		channelID:       channelID,
		ledger:          ledger,
		gossip:          getGossipAdapter(),
		configRetriever: getCollConfigRetriever(channelID, ledger, blockPublisher),
	}, nil
}

// Put puts the value for the given key
func (d *Client) Put(ns, coll, key string, value []byte) error {
	return d.PutMultipleValues(ns, coll, []*KeyValue{{Key: key, Value: value}})
}

// PutMultipleValues puts the given key/values
func (d *Client) PutMultipleValues(ns, coll string, kvs []*KeyValue) error {
	bcInfo, err := d.ledger.GetBlockchainInfo()
	if err != nil {
		logger.Warningf("[%s] Error getting blockchain info: %s", d.channelID, err)
		return errors.WithMessagef(err, "error getting blockchain info in channel [%s]", d.channelID)
	}

	// Generate a new TxID. The TxID doesn't really matter since this transaction is never committed.
	// It just has to be unique.
	txID, err := d.newTxID()
	if err != nil {
		logger.Warningf("[%s] Error generating transaction ID: %s", d.channelID, err)
		return errors.WithMessagef(err, "error generating transaction ID in channel [%s]", d.channelID)
	}

	sim, err := d.ledger.NewTxSimulator(txID)
	if err != nil {
		logger.Warningf("[%s] Error getting TxSimulator for transaction [%s]: %s", d.channelID, txID, err)
		return errors.WithMessagef(err, "error getting TxSimulator for transaction [%s] in channel [%s]", txID, d.channelID)
	}
	defer sim.Done()

	mapByKey := make(map[string][]byte)
	for _, kv := range kvs {
		mapByKey[kv.Key] = kv.Value
	}

	err = sim.SetPrivateDataMultipleKeys(ns, coll, mapByKey)
	if err != nil {
		logger.Warningf("[%s] Error setting values for transaction [%s]: %s", d.channelID, txID, err)
		return errors.WithMessagef(err, "error setting keys for transaction [%s] in channel [%s]", txID, d.channelID)
	}

	results, err := sim.GetTxSimulationResults()
	if err != nil {
		logger.Warningf("[%s] Error generating simulation results for transaction [%s]: %s", d.channelID, txID, err)
		return errors.WithMessagef(err, "error generating simulation results for transaction [%s] in channel [%s]", txID, d.channelID)
	}

	configPkg, err := d.getCollectionConfigPackage(ns, coll)
	if err != nil {
		logger.Warningf("[%s] Error getting collection config for [%s:%s]: %s", d.channelID, ns, coll, err)
		return errors.WithMessagef(err, "error getting collection config for [%s:%s] in channel [%s]", ns, coll, d.channelID)
	}

	pvtData := &transientstore.TxPvtReadWriteSetWithConfigInfo{
		EndorsedAt: bcInfo.Height,
		PvtRwset:   results.PvtSimulationResults,
		CollectionConfigs: map[string]*cb.CollectionConfigPackage{
			ns: configPkg,
		},
	}

	err = d.gossip.DistributePrivateData(d.channelID, txID, pvtData, bcInfo.Height)
	if err != nil {
		logger.Warningf("[%s] Failed to distribute private data: %s", d.channelID, err)
		return errors.WithMessagef(err, "error distributing private data in channel [%s]", d.channelID)
	}

	return nil
}

// Delete deletes the given key(s)
func (d *Client) Delete(ns, coll string, keys ...string) error {
	kvs := make([]*KeyValue, len(keys))
	for i, key := range keys {
		kvs[i] = &KeyValue{Key: key}
	}
	return d.PutMultipleValues(ns, coll, kvs)
}

// Get retrieves the value for the given key
func (d *Client) Get(ns, coll, key string) ([]byte, error) {
	qe, err := d.ledger.NewQueryExecutor()
	if err != nil {
		logger.Warningf("[%s] Error getting QueryExecutor: %s", d.channelID, err)
		return nil, errors.WithMessagef(err, "error getting QueryExecutor in channel [%s]", d.channelID)
	}
	defer qe.Done()

	return qe.GetPrivateData(ns, coll, key)
}

// GetMultipleKeys retrieves the values for the given keys
func (d *Client) GetMultipleKeys(ns, coll string, keys ...string) ([][]byte, error) {
	qe, err := d.ledger.NewQueryExecutor()
	if err != nil {
		logger.Warningf("[%s] Error getting QueryExecutor: %s", d.channelID, err)
		return nil, errors.WithMessagef(err, "error getting QueryExecutor in channel [%s]", d.channelID)
	}
	defer qe.Done()

	return qe.GetPrivateDataMultipleKeys(ns, coll, keys)
}

// Query executes the given query and returns an iterator that contains results.
// Only used for state databases that support query.
// (Note that this function is not supported by transient data collections)
// The returned ResultsIterator contains results of type *KV which is defined in protos/ledger/queryresult.
func (d *Client) Query(ns, coll, query string) (commonledger.ResultsIterator, error) {
	qe, err := d.ledger.NewQueryExecutor()
	if err != nil {
		logger.Warningf("[%s] Error getting QueryExecutor: %s", d.channelID, err)
		return nil, errors.WithMessagef(err, "error getting QueryExecutor in channel [%s]", d.channelID)
	}
	defer qe.Done()

	return qe.ExecuteQueryOnPrivateData(ns, coll, query)
}

func (d *Client) getCollectionConfigPackage(ns, coll string) (*cb.CollectionConfigPackage, error) {
	collConfig, err := d.configRetriever.Config(ns, coll)
	if err != nil {
		return nil, err
	}

	return &cb.CollectionConfigPackage{
		Config: []*cb.CollectionConfig{
			{
				Payload: &cb.CollectionConfig_StaticCollectionConfig{
					StaticCollectionConfig: collConfig,
				},
			},
		},
	}, nil
}

func (d *Client) newTxID() (string, error) {
	creator, err := d.getCreator()
	if err != nil {
		return "", errors.WithMessage(err, "error serializing local signing identity")
	}

	nonce, err := crypto.GetRandomNonce()
	if err != nil {
		return "", errors.WithMessage(err, "nonce creation failed")
	}

	txnID, err := computeTxID(nonce, creator)
	if err != nil {
		return "", errors.WithMessage(err, "txn ID computation failed")
	}

	return txnID, nil
}

func (d *Client) getCreator() ([]byte, error) {
	d.mutex.RLock()
	c := d.creator
	d.mutex.RUnlock()

	if c != nil {
		return c, nil
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.creator == nil {
		creator, err := newCreator()
		if err != nil {
			return nil, errors.WithMessage(err, "error serializing local signing identity")
		}
		d.creator = creator
	}

	return d.creator, nil
}

func computeTxID(nonce, creator []byte) (string, error) {
	digest, err := factory.GetDefault().Hash(append(nonce, creator...), &bccsp.SHA256Opts{})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(digest), nil
}

// getLedger returns the peer ledger. This var may be overridden in unit tests
var getLedger = func(channelID string) PeerLedger {
	return peer.GetLedger(channelID)
}

// getLedger returns the peer ledger. This var may be overridden in unit tests
var getBlockPublisher = func(channelID string) gossipapi.BlockPublisher {
	return blockpublisher.GetProvider().ForChannel(channelID)
}

// getGossipAdapter returns the gossip adapter. This var may be overridden in unit tests
var getGossipAdapter = func() GossipAdapter {
	return service.GetGossipService()
}

var getCollConfigRetriever = func(channelID string, ledger PeerLedger, blockPublisher gossipapi.BlockPublisher) CollectionConfigRetriever {
	return support.CollectionConfigRetrieverForChannel(channelID)
}

var newCreator = func() ([]byte, error) {
	id, err := mspmgmt.GetLocalMSP().GetDefaultSigningIdentity()
	if err != nil {
		return nil, errors.WithMessage(err, "error getting local signing identity")
	}
	return id.Serialize()
}
