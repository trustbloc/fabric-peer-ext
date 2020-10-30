/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"io"

	"github.com/bluele/gcache"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/pkg/errors"

	olclient "github.com/trustbloc/fabric-peer-ext/pkg/collections/client"
)

var logger = flogging.MustGetLogger("ext_offledger")

// Link is a link to another node in DCAS
type Link struct {
	// Hash contains the content ID of the node
	Hash string `json:"hash"`
	// Name is an optional name for the link
	Name string `json:"name,omitempty"`
	// Size contains the size of the target node
	Size uint64 `json:"size,omitempty"`
}

// Node contains data and optional links to other nodes
type Node struct {
	// Data contains the raw data of the node
	Data []byte `json:"data,omitempty"`
	// Links contains zero or more links to other nodes
	Links []Link `json:"links,omitempty"`
}

// NodeType specifies the type of node to be stored (object or file)
type NodeType string

const (
	// ObjectNodeType stores the data as an object using a specified format
	ObjectNodeType NodeType = "object"

	// FileNodeType stores the data either as a single 'raw' node or, if the data is to large then the data
	// is broken up into equal-sized chunks and stored as a Merkle DAG (see https://docs.ipfs.io/concepts/merkle-dag/).
	FileNodeType NodeType = "file"
)

// InputEncoding specifies the input encoding of the data being stored
type InputEncoding string

const (
	// JSONEncoding indicates that the input is encoded in JSON
	JSONEncoding InputEncoding = "json"
	// RawEncoding indicates that the input should not be interpreted
	RawEncoding InputEncoding = "raw"
	// CborEncoding indicates that the input is encoded in Concise Binary Object Representation (CBOR)
	// (see https://specs.ipld.io/block-layer/codecs/dag-cbor.html)
	CborEncoding InputEncoding = "cbor"
	// ProtobufEncoding indicates that the input is encoded as a protobuf
	ProtobufEncoding InputEncoding = "protobuf"
)

// Format specifies the format of the stored data
type Format string

const (
	// CborFormat is a Concise Binary Object Representation (CBOR) data format (see https://specs.ipld.io/block-layer/codecs/dag-cbor.html)
	CborFormat Format = "cbor"
	// ProtobufFormat stores a node which is marshalled using proto buf
	ProtobufFormat Format = "protobuf"
	// RawFormat stores the node as it was input
	RawFormat Format = "raw"
)

type options struct {
	multihashType uint64
	nodeType      NodeType
	inputEncoding InputEncoding
	format        Format
}

// Option is a DCAS client option
type Option func(opts *options)

// WithMultihash sets the multihash type
func WithMultihash(mhType uint64) Option {
	return func(opts *options) {
		opts.multihashType = mhType
	}
}

// WithNodeType sets the type of node to use, i.e. object or file
func WithNodeType(nodeType NodeType) Option {
	return func(opts *options) {
		opts.nodeType = nodeType
	}
}

// WithInputEncoding sets the encoding of the input data
// Note: This option is only applicable to object node types.
func WithInputEncoding(encoding InputEncoding) Option {
	return func(opts *options) {
		opts.inputEncoding = encoding
	}
}

// WithFormat sets the format of the stored object
// Note: This option is only applicable to object node types.
func WithFormat(format Format) Option {
	return func(opts *options) {
		opts.format = format
	}
}

// DCAS defines the functions of a DCAS client
type DCAS interface {
	// Put puts the data and returns the content ID (CID) for the value
	Put(data io.Reader, opts ...Option) (string, error)

	// Delete deletes the values for the given content IDs.
	Delete(cids ...string) error

	// Get retrieves the value for the given content ID (CID).
	Get(cid string, w io.Writer) error

	// GetNode retrieves the CAS Node for the given content ID (CID). A node contains data and/or links to other nodes.
	GetNode(cid string) (*Node, error)
}

// Provider manages multiple clients - one per channel
type Provider struct {
	config
	cache gcache.Cache
}

// NewProvider returns a new client provider
func NewProvider(providers *olclient.Providers, cfg config) *Provider {
	logger.Infof("Creating DCAS client provider.")
	return &Provider{
		config: cfg,
		cache: gcache.New(0).LoaderFunc(func(key interface{}) (i interface{}, e error) {
			return newClient(key.(cacheKey), providers, cfg)
		}).Build(),
	}
}

// GetDCASClient returns the client for the given channel
func (p *Provider) GetDCASClient(channelID string, namespace string, coll string) (DCAS, error) {
	if channelID == "" || namespace == "" || coll == "" {
		return nil, errors.New("channel ID, namespace, and collection must be specified")
	}

	c, err := p.cache.Get(newCacheKey(channelID, namespace, coll))
	if err != nil {
		return nil, err
	}

	return c.(DCAS), nil
}

// CreateDCASClientStubWrapper returns a new DCAS client that wraps the given chaincode stub using the given collection
func (p *Provider) CreateDCASClientStubWrapper(coll string, stub shim.ChaincodeStubInterface) (DCAS, error) {
	if coll == "" {
		return nil, errors.New("collection must be specified")
	}

	logger.Debugf("Creating stub wrapper for collection [%s]", coll)

	return createCCStubWrappedClient(p.config, coll, stub)
}

func newClient(key cacheKey, p *olclient.Providers, cfg config) (*DCASClient, error) {
	logger.Debugf("Creating client for [%s]", key)

	l := p.LedgerProvider.GetLedger(key.channelID)
	if l == nil {
		return nil, errors.Errorf("no ledger for channel [%s]", key.channelID)
	}

	return createOLWrappedClient(cfg, key.channelID, key.namespace, key.collection,
		&olclient.ChannelProviders{
			Ledger:           l,
			Distributor:      p.GossipProvider.GetGossipService(),
			ConfigRetriever:  p.ConfigProvider.ForChannel(key.channelID),
			IdentityProvider: p.IdentityProvider,
		},
	)
}

type cacheKey struct {
	channelID  string
	namespace  string
	collection string
}

func newCacheKey(channelID, ns, coll string) cacheKey {
	return cacheKey{
		channelID:  channelID,
		namespace:  ns,
		collection: coll,
	}
}
