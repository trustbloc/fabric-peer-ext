/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	mh "github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/require"

	olclient "github.com/trustbloc/fabric-peer-ext/pkg/collections/client"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/dcas"
	olmocks "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/mocks"
)

const (
	channelID = "testchannel"
	ns1       = "ns1"
	coll1     = "coll1"
)

func TestDCASClient_Put(t *testing.T) {
	cfg := &olmocks.DCASConfig{}
	cfg.GetDCASMaxBlockSizeReturns(32)
	cfg.GetDCASMaxLinksPerBlockReturns(5)
	cfg.IsDCASRawLeavesReturns(true)

	ds := newMockDataStore()

	c, err := createClient(cfg, ds)
	require.NoError(t, err)
	require.NotNil(t, c)

	value1 := []byte("value1")

	jsonValue := []byte(`{"fieldx":"valuex","field1":"value1","field2":"value2"}`)

	t.Run("Default node-type - Non-JSON -> Success", func(t *testing.T) {
		cID, err := c.Put(bytes.NewReader(value1), WithMultihash(mh.SHA2_256))
		require.NoError(t, err)
		require.NoError(t, dcas.ValidateCID(cID))
	})

	t.Run("Default node-type - JSON -> Success", func(t *testing.T) {
		cID, err := c.Put(bytes.NewReader(jsonValue))
		require.NoError(t, err)
		require.NoError(t, dcas.ValidateCID(cID))
	})

	t.Run("Object node-type - JSON -> Success", func(t *testing.T) {
		cID, err := c.Put(bytes.NewReader(jsonValue), WithNodeType(ObjectNodeType), WithFormat(CborFormat), WithInputEncoding(JSONEncoding))
		require.NoError(t, err)
		require.NoError(t, dcas.ValidateCID(cID))
	})

	t.Run("Object node-type - RAW - no data -> Success", func(t *testing.T) {
		cID, err := c.Put(bytes.NewReader(nil))
		require.NoError(t, err)
		t.Logf("Got CID: %s", cID)
		require.NoError(t, dcas.ValidateCID(cID))
	})

	t.Run("Object node-type - data store error -> Error", func(t *testing.T) {
		errExpected := fmt.Errorf("injected data store error")
		ds.WithError(errExpected)
		defer func() { ds.WithError(nil) }()

		cID, err := c.Put(bytes.NewReader(jsonValue), WithNodeType(ObjectNodeType), WithFormat(CborFormat), WithInputEncoding(JSONEncoding))
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.Empty(t, cID)
	})

	t.Run("Object node-type - non-JSON -> Error", func(t *testing.T) {
		cID, err := c.Put(bytes.NewReader(value1), WithNodeType(ObjectNodeType), WithFormat(CborFormat), WithInputEncoding(JSONEncoding))
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid character")
		require.Empty(t, cID)
	})

	t.Run("File node-type -> Success", func(t *testing.T) {
		cID, err := c.Put(bytes.NewReader(jsonValue), WithNodeType(FileNodeType))
		require.NoError(t, err)
		require.NoError(t, dcas.ValidateCID(cID))
	})

	t.Run("File node-type - datastore error -> Error", func(t *testing.T) {
		errExpected := fmt.Errorf("injected data store error")
		ds.WithError(errExpected)
		defer func() { ds.WithError(nil) }()

		cID, err := c.Put(bytes.NewReader(jsonValue), WithNodeType(FileNodeType))
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.Empty(t, cID)
	})

	t.Run("Unsupported node-type -> Success", func(t *testing.T) {
		cID, err := c.Put(bytes.NewReader(jsonValue), WithNodeType("invalid-type"))
		require.EqualError(t, err, "unsupported node type [invalid-type]")
		require.Empty(t, cID)
	})

	t.Run("Delete", func(t *testing.T) {
		cID, err := dcas.GetCID(value1, dcas.CIDV1, cid.Raw, mh.SHA2_256)
		require.NoError(t, err)
		require.NoError(t, c.Delete(cID))
	})
}

func TestDCASClient_Get(t *testing.T) {
	fileData1 := []byte("Here is some data which is longer than the maximum size for a file node in DCAS so it will have to be split into multiple chunks.")
	fileData2 := []byte("Here is a small file.")

	value1 := []byte("value1")
	value2 := []byte("value2")

	key1, err := dcas.GetCASKey(value1, dcas.CIDV1, cid.Raw, mh.SHA2_256)
	require.NoError(t, err)

	cidRaw1, err := dcas.GetCID(value1, dcas.CIDV1, cid.Raw, mh.SHA2_256)
	require.NoError(t, err)
	cidRaw2, err := dcas.GetCID(value2, dcas.CIDV1, cid.Raw, mh.SHA2_256)
	require.NoError(t, err)

	cfg := &olmocks.DCASConfig{}
	cfg.GetDCASMaxBlockSizeReturns(32)
	cfg.GetDCASMaxLinksPerBlockReturns(5)
	cfg.IsDCASRawLeavesReturns(true)

	ds := newMockDataStore().WithData(key1, value1)

	c, err := createClient(cfg, ds)
	require.NoError(t, err)
	require.NotNil(t, c)

	cidFile1, err := c.Put(bytes.NewReader(fileData1), WithNodeType(FileNodeType))
	require.NoError(t, err)
	require.NotEmpty(t, cidFile1)

	cidFile2, err := c.Put(bytes.NewReader(fileData2), WithNodeType(FileNodeType))
	require.NoError(t, err)
	require.NotEmpty(t, cidFile2)

	t.Run("Get invalid CID - success", func(t *testing.T) {
		b := bytes.NewBuffer(nil)
		err := c.Get("invalid", b)
		require.EqualError(t, err, "selected encoding not supported")
		require.Empty(t, b.Bytes())
	})

	t.Run("Get raw - success", func(t *testing.T) {
		b := bytes.NewBuffer(nil)
		require.NoError(t, c.Get(cidRaw1, b))
		require.Equal(t, value1, b.Bytes())
	})

	t.Run("Get raw - not found", func(t *testing.T) {
		b := bytes.NewBuffer(nil)
		require.NoError(t, c.Get(cidRaw2, b))
		require.Empty(t, b.Bytes())
	})

	t.Run("Get CBOR - success", func(t *testing.T) {
		cborValue1 := []byte(fmt.Sprintf(`{"field1":"value1","files":[{"/":"%s"},{"/":"%s"}]}`, cidFile1, cidFile2))

		cID, err := c.Put(bytes.NewReader(cborValue1), WithNodeType(ObjectNodeType), WithInputEncoding(JSONEncoding), WithFormat(CborFormat))
		require.NoError(t, err)
		require.NotEmpty(t, cID)

		b := bytes.NewBuffer(nil)
		require.NoError(t, c.Get(cID, b))
		require.Equal(t, string(cborValue1), b.String())

		nd, err := c.GetNode(cID)
		require.NoError(t, err)
		require.NotNil(t, nd)
		require.Len(t, nd.Links, 2)
		require.Equal(t, cidFile1, nd.Links[0].Hash)
		require.Equal(t, cidFile2, nd.Links[1].Hash)
	})

	t.Run("Get protobuf - success", func(t *testing.T) {
		// Create a protobuf file with two links
		pbValue1 := []byte(fmt.Sprintf(`{"data":"","links":[{"Cid":{"/":"%s"}},{"Cid":{"/":"%s"}}]}`, cidFile1, cidFile2))

		cID, err := c.Put(bytes.NewReader(pbValue1), WithNodeType(ObjectNodeType), WithInputEncoding(JSONEncoding), WithFormat(ProtobufFormat))
		require.NoError(t, err)
		require.NotEmpty(t, cID)

		nd, err := c.GetNode(cID)
		require.NoError(t, err)
		require.NotNil(t, nd)
		require.Len(t, nd.Links, 2)
		require.Equal(t, cidFile1, nd.Links[0].Hash)
		require.Equal(t, cidFile2, nd.Links[1].Hash)
	})

	t.Run("Get file - success", func(t *testing.T) {
		b := bytes.NewBuffer(nil)
		require.NoError(t, c.Get(cidFile1, b))
		require.Equal(t, string(fileData1), b.String())
	})

	t.Run("Get file node - success", func(t *testing.T) {
		nd, err := c.GetNode(cidFile1)
		require.NoError(t, err)
		require.NotNil(t, nd)
		require.Equal(t, 5, len(nd.Links))
	})

	t.Run("Get node - not found", func(t *testing.T) {
		nd, err := c.GetNode(cidRaw2)
		require.NoError(t, err)
		require.Nil(t, nd)
	})

	t.Run("Get file node - data store error - success", func(t *testing.T) {
		errExpected := fmt.Errorf("injected data store error")
		ds.WithError(errExpected)
		defer func() { ds.WithError(nil) }()

		nd, err := c.GetNode(cidFile1)
		require.Error(t, err)
		require.Nil(t, nd)
		require.Contains(t, err.Error(), errExpected.Error())
	})
}

func TestCreateClient(t *testing.T) {
	providers := &olclient.ChannelProviders{}

	cfg := &olmocks.DCASConfig{}
	cfg.GetDCASMaxBlockSizeReturns(1024)
	cfg.GetDCASMaxLinksPerBlockReturns(10)
	cfg.IsDCASRawLeavesReturns(false)

	c, err := createOLWrappedClient(cfg, channelID, ns1, coll1, providers)
	require.NoError(t, err)
	require.NotNil(t, c)

	cfg.GetDCASBlockLayoutReturns(trickleLayout)
	c, err = createOLWrappedClient(cfg, channelID, ns1, coll1, providers)
	require.NoError(t, err)
	require.NotNil(t, c)

	cfg.GetDCASBlockLayoutReturns(balancedLayout)
	c, err = createOLWrappedClient(cfg, channelID, ns1, coll1, providers)
	require.NoError(t, err)
	require.NotNil(t, c)

	cfg.GetDCASBlockLayoutReturns("invalid-layout")
	c, err = createOLWrappedClient(cfg, channelID, ns1, coll1, providers)
	require.EqualError(t, err, "invalid block layout strategy [invalid-layout]")
	require.Nil(t, c)
}

type mockDataStore struct {
	data map[datastore.Key][]byte
	err  error
}

func newMockDataStore() *mockDataStore {
	return &mockDataStore{
		data: make(map[datastore.Key][]byte),
	}
}

func (m *mockDataStore) WithData(key string, value []byte) *mockDataStore {
	m.data[datastore.NewKey(key)] = value
	return m
}

func (m *mockDataStore) WithError(err error) *mockDataStore {
	m.err = err
	return m
}

func (m *mockDataStore) Get(key datastore.Key) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}

	logger.Infof("Getting data for key [%s]", key)
	v, ok := m.data[key]
	if !ok {
		return nil, datastore.ErrNotFound
	}

	return v, nil
}

func (m *mockDataStore) GetSize(datastore.Key) (int, error) {
	panic("not implemented")
}

func (m *mockDataStore) Put(key datastore.Key, value []byte) error {
	if m.err != nil {
		return m.err
	}

	logger.Infof("Putting data for key [%s]", key)
	m.data[key] = value

	return nil
}

func (m *mockDataStore) Delete(key datastore.Key) error {
	if m.err != nil {
		return m.err
	}

	delete(m.data, key)

	return nil
}

func (m *mockDataStore) Has(key datastore.Key) (bool, error) {
	return false, nil
}

func (m *mockDataStore) Query(q query.Query) (query.Results, error) {
	panic("not implemented")
}

func (m *mockDataStore) Sync(prefix datastore.Key) error {
	// No-op
	return nil
}

func (m *mockDataStore) Close() error {
	// No-op
	return nil
}

func (m *mockDataStore) Batch() (datastore.Batch, error) {
	// Not supported
	return nil, datastore.ErrBatchUnsupported
}
