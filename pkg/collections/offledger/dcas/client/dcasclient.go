/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"context"
	"fmt"
	"io"
	"reflect"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	chunker "github.com/ipfs/go-ipfs-chunker"
	cbornode "github.com/ipfs/go-ipld-cbor"
	format "github.com/ipfs/go-ipld-format"
	ipld "github.com/ipfs/go-ipld-format"
	dag "github.com/ipfs/go-merkledag"
	"github.com/ipfs/go-unixfs"
	"github.com/ipfs/go-unixfs/importer/balanced"
	unixfshelpers "github.com/ipfs/go-unixfs/importer/helpers"
	"github.com/ipfs/go-unixfs/importer/trickle"
	mh "github.com/multiformats/go-multihash"
	"github.com/pkg/errors"
	olclient "github.com/trustbloc/fabric-peer-ext/pkg/collections/client"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/dcas/ipfsdatastore"
	"github.com/trustbloc/fabric-peer-ext/pkg/internal/github.com/ipfs/go-ipfs/core/coredag"
)

const (
	balancedLayout = "balanced"
	trickleLayout  = "trickle"
)

type config interface {
	GetDCASMaxLinksPerBlock() int
	IsDCASRawLeaves() bool
	GetDCASMaxBlockSize() int64
	GetDCASBlockLayout() string
}

type layoutStrategy = func(db *unixfshelpers.DagBuilderHelper) (ipld.Node, error)

// DCASClient allows you to put and get DCASClient from outside of a chaincode
type DCASClient struct {
	config
	dagService format.DAGService
	layout     layoutStrategy
}

func createOLWrappedClient(cfg config, channelID, ns, coll string, providers *olclient.ChannelProviders) (*DCASClient, error) {
	return createClient(cfg, ipfsdatastore.NewOLClientWrapper(ns, coll, olclient.New(channelID, providers)))
}

func createCCStubWrappedClient(cfg config, coll string, stub shim.ChaincodeStubInterface) (*DCASClient, error) {
	return createClient(cfg, ipfsdatastore.NewStubWrapper(coll, stub))
}

func createClient(cfg config, ds datastore.Batching) (*DCASClient, error) {
	layout, err := resolveLayoutStrategy(cfg)
	if err != nil {
		return nil, err
	}

	return &DCASClient{
		config: cfg,
		layout: layout,
		dagService: dag.NewDAGService(
			blockservice.NewWriteThrough(
				blockstore.NewBlockstore(ds), nil,
			),
		),
	}, nil
}

// Put stores the given content and returns the content ID (CID) for the value
func (d *DCASClient) Put(data io.Reader, opts ...Option) (string, error) {
	options := getOptions(opts)
	switch options.nodeType {
	case FileNodeType:
		return d.putFile(data, options)
	case ObjectNodeType:
		fallthrough
	case "":
		return d.putObject(context.Background(), data, options)
	default:
		return "", fmt.Errorf("unsupported node type [%s]", options.nodeType)
	}
}

// Delete deletes the values for the given content IDs.
func (d *DCASClient) Delete(ids ...string) error {
	cids := make([]cid.Cid, len(ids))
	for i, key := range ids {
		cID, err := cid.Decode(key)
		if err != nil {
			return err
		}

		cids[i] = cID
	}

	return d.dagService.RemoveMany(context.Background(), cids)
}

// Get retrieves the value for the given content ID.
func (d *DCASClient) Get(id string, w io.Writer) error {
	ctx := context.Background()

	nd, err := d.getNode(ctx, id)
	if err != nil {
		logger.Infof("Error getting node for CID [%s]: %s", id, err)

		return err
	}

	if nd == nil {
		return nil
	}

	err = d.getContent(ctx, nd, w)
	if err != nil {
		logger.Warnf("Error getting content for CID [%s]: %s", id, err)

		return err
	}

	return nil
}

// GetNode retrieves the CAS Node for the given content ID. A node contains data and/or links to other nodes.
func (d *DCASClient) GetNode(id string) (*Node, error) {
	nd, err := d.getNode(context.Background(), id)
	if err != nil {
		return nil, err
	}

	if nd == nil {
		return nil, nil
	}

	links := make([]Link, len(nd.Links()))

	for i, l := range nd.Links() {
		links[i] = Link{
			Hash: l.Cid.String(),
			Size: l.Size,
		}
	}

	return &Node{
		Data:  nd.RawData(),
		Links: links,
	}, nil
}

func (d *DCASClient) putObject(ctx context.Context, data io.Reader, options *options) (string, error) {
	nds, err := coredag.ParseInputs(
		string(options.inputEncoding), string(options.format),
		data, options.multihashType, -1,
	)
	if err != nil {
		return "", errors.WithMessagef(err, "error in ParseInputs")
	}

	if len(nds) != 1 {
		// This should never happen since the implementation of ParseInputs always returns one node
		return "", fmt.Errorf("expecting one node but got %d from ParseInputs", len(nds))
	}

	nd := nds[0]

	err = d.dagService.Add(ctx, nd)
	if err != nil {
		return "", err
	}

	return nd.Cid().String(), nil
}

func (d *DCASClient) putFile(data io.Reader, options *options) (string, error) {
	dbp := unixfshelpers.DagBuilderParams{
		Maxlinks:   d.GetDCASMaxLinksPerBlock(),
		RawLeaves:  d.IsDCASRawLeaves(),
		CidBuilder: cid.V1Builder{MhType: options.multihashType},
		Dagserv:    d.dagService,
		NoCopy:     false,
	}

	db, err := dbp.New(chunker.NewSizeSplitter(data, d.GetDCASMaxBlockSize()))
	if err != nil {
		return "", errors.WithMessage(err, "error creating DagBuilderParams")
	}

	root, err := d.layout(db)
	if err != nil {
		logger.Errorf("Error laying out file data: %s", err)

		return "", errors.WithMessage(err, "error laying out Merkle DAG")
	}

	logger.Debugf("Successfully put file. CID [%s]", root.Cid())

	return root.Cid().String(), nil
}

func (d *DCASClient) getNode(ctx context.Context, id string) (ipld.Node, error) {
	logger.Debugf("Getting node for CID: %s", id)

	cID, err := cid.Decode(id)
	if err != nil {
		logger.Debugf("Invalid CID [%s]: %s", id, err)

		return nil, err
	}

	nd, err := d.dagService.Get(ctx, cID)
	if err != nil {
		if err == format.ErrNotFound {
			return nil, nil
		}

		logger.Warnf("Error getting root node for CID [%s]: %s", id, err)

		return nil, err
	}

	return nd, nil
}

func (d *DCASClient) getContent(ctx context.Context, nd ipld.Node, w io.Writer) error {
	logger.Debugf("Getting content from CID [%s], Node Type: %s, CID Version: %d, Multi-hash type: %d, Codec: %d",
		nd.String(), reflect.TypeOf(nd), nd.Cid().Prefix().Version, nd.Cid().Prefix().MhType, nd.Cid().Prefix().Codec)

	switch node := nd.(type) {
	case *dag.ProtoNode:
		return d.getContentFromProtoNode(ctx, node, w)
	case *cbornode.Node:
		return d.getContentFromCBORNode(node, w)
	default:
		return d.getContentFromRawNode(node, w)
	}
}

func (d *DCASClient) getContentFromRawNode(nd ipld.Node, w io.Writer) error {
	logger.Debugf("Writing %d bytes of data for CID [%s]", len(nd.RawData()), nd.String())

	_, err := w.Write(nd.RawData())
	return err
}

func (d *DCASClient) getContentFromCBORNode(nd *cbornode.Node, w io.Writer) error {
	j, err := nd.MarshalJSON()
	if err != nil {
		return err
	}

	logger.Debugf("Writing %d bytes of data for CID [%s]", len(nd.RawData()), nd.String())

	_, err = w.Write(j)

	return err
}

func (d *DCASClient) getContentFromProtoNode(ctx context.Context, nd *dag.ProtoNode, w io.Writer) error {
	if len(nd.Links()) == 0 {
		data, err := unixfs.ReadUnixFSNodeData(nd)
		if err != nil {
			return err
		}

		logger.Debugf("Writing %d bytes of data for CID [%d]", len(data), nd.String())

		_, err = w.Write(data)

		return err
	}

	logger.Debugf("Resolving %d links from CID [%s] ...", len(nd.Links()), nd.String())

	for _, l := range nd.Links() {
		logger.Debugf("[%s] -> resolving link [%s]", nd.String(), l.Cid.String())

		node, err := d.dagService.Get(ctx, l.Cid)
		if err != nil {
			return err
		}

		err = d.getContent(ctx, node, w)
		if err != nil {
			return err
		}
	}

	return nil
}

func getOptions(opts []Option) *options {
	o := &options{
		multihashType: mh.SHA2_256,
		nodeType:      ObjectNodeType,
		inputEncoding: RawEncoding, // Only applies to object nodes
		format:        RawFormat,   // Only applies to object nodes
	}

	for _, opt := range opts {
		opt(o)
	}

	return o
}

func resolveLayoutStrategy(cfg config) (layoutStrategy, error) {
	switch cfg.GetDCASBlockLayout() {
	case trickleLayout:
		logger.Debugf("Using 'trickle' layout for DCAS client")
		return trickle.Layout, nil
	case balancedLayout:
		fallthrough
	case "":
		logger.Debugf("Using 'balanced' layout for DCAS client")
		return balanced.Layout, nil
	default:
		return nil, fmt.Errorf("invalid block layout strategy [%s]", cfg.GetDCASBlockLayout())
	}
}
