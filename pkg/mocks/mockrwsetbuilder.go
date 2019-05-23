/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	"github.com/hyperledger/fabric/protos/transientstore"
)

// ReadWriteSetBuilder is a utility that builds a TxReadWriteSet for unit testing
type ReadWriteSetBuilder struct {
	namespaces []*NamespaceBuilder
}

// NewReadWriteSetBuilder returns a new ReadWriteSetBuilder
func NewReadWriteSetBuilder() *ReadWriteSetBuilder {
	return &ReadWriteSetBuilder{}
}

// Namespace returns a new NamespaceBuilder
func (b *ReadWriteSetBuilder) Namespace(name string) *NamespaceBuilder {
	ns := NewNamespaceBuilder(name)
	b.namespaces = append(b.namespaces, ns)
	return ns
}

// Build builds the read-write sets
func (b *ReadWriteSetBuilder) Build() *rwset.TxReadWriteSet {
	txRWSet := &rwset.TxReadWriteSet{
		DataModel: rwset.TxReadWriteSet_KV,
	}

	for _, ns := range b.namespaces {
		txRWSet.NsRwset = append(txRWSet.NsRwset,
			&rwset.NsReadWriteSet{
				Namespace:             ns.name,
				Rwset:                 ns.BuildNSReadWriteSets(),
				CollectionHashedRwset: ns.BuildCollectionHashedRWSets(),
			},
		)
	}

	return txRWSet
}

// PvtReadWriteSetBuilder is a utility that builds a TxPvtReadWriteSetWithConfigInfo for unit testing
type PvtReadWriteSetBuilder struct {
	namespaces []*NamespaceBuilder
}

// NewPvtReadWriteSetBuilder returns a new PvtReadWriteSetBuilder
func NewPvtReadWriteSetBuilder() *PvtReadWriteSetBuilder {
	return &PvtReadWriteSetBuilder{}
}

// Namespace returns a new NamespaceBuilder
func (b *PvtReadWriteSetBuilder) Namespace(name string) *NamespaceBuilder {
	ns := NewNamespaceBuilder(name)
	b.namespaces = append(b.namespaces, ns)
	return ns
}

// Build builds a TxPvtReadWriteSetWithConfigInfo
func (b *PvtReadWriteSetBuilder) Build() *transientstore.TxPvtReadWriteSetWithConfigInfo {
	return &transientstore.TxPvtReadWriteSetWithConfigInfo{
		PvtRwset:          b.BuildReadWriteSet(),
		CollectionConfigs: b.BuildCollectionConfigs(),
	}
}

// BuildReadWriteSet builds the private read-write sets
func (b *PvtReadWriteSetBuilder) BuildReadWriteSet() *rwset.TxPvtReadWriteSet {
	pvtWriteSet := &rwset.TxPvtReadWriteSet{
		DataModel: rwset.TxReadWriteSet_KV,
	}

	for _, ns := range b.namespaces {
		pvtWriteSet.NsPvtRwset = append(pvtWriteSet.NsPvtRwset,
			&rwset.NsPvtReadWriteSet{
				Namespace:          ns.name,
				CollectionPvtRwset: ns.BuildReadWriteSets(),
			},
		)
	}

	return pvtWriteSet
}

// BuildCollectionConfigs builds the collection config package
func (b *PvtReadWriteSetBuilder) BuildCollectionConfigs() map[string]*common.CollectionConfigPackage {
	configs := make(map[string]*common.CollectionConfigPackage)
	for _, ns := range b.namespaces {
		configs[ns.name] = ns.BuildCollectionConfig()
	}
	return configs
}

// NamespaceBuilder is a utility that builds a CollectionPvtReadWriteSet and CollectionConfigPackage for unit testing
type NamespaceBuilder struct {
	name        string
	reads       map[string]*kvrwset.Version
	writes      map[string][]byte
	collections []*CollectionBuilder
	marshalErr  bool
}

// NewNamespaceBuilder returns a new namespace builder
func NewNamespaceBuilder(name string) *NamespaceBuilder {
	return &NamespaceBuilder{
		name:   name,
		reads:  make(map[string]*kvrwset.Version),
		writes: make(map[string][]byte),
	}
}

// Read adds a new read to the namespace
func (b *NamespaceBuilder) Read(key string, blockNum uint64, txIdx uint64) *NamespaceBuilder {
	b.reads[key] = &kvrwset.Version{BlockNum: blockNum, TxNum: txIdx}
	return b
}

// Write adds a new write to the namespace
func (b *NamespaceBuilder) Write(key string, value []byte) *NamespaceBuilder {
	b.writes[key] = value
	return b
}

// Delete adds a new write with 'IsDelete=true' to the namespace
func (b *NamespaceBuilder) Delete(key string) *NamespaceBuilder {
	b.writes[key] = nil
	return b
}

// Collection adds a new collection
func (b *NamespaceBuilder) Collection(name string) *CollectionBuilder {
	cb := NewPvtReadWriteSetCollectionBuilder(name)
	b.collections = append(b.collections, cb)
	return cb
}

// WithMarshalError simulates a marshalling error
func (b *NamespaceBuilder) WithMarshalError() *NamespaceBuilder {
	b.marshalErr = true
	return b
}

// BuildReadWriteSets builds the collection read-write sets for the namespace
func (b *NamespaceBuilder) BuildReadWriteSets() []*rwset.CollectionPvtReadWriteSet {
	var rwSets []*rwset.CollectionPvtReadWriteSet
	for _, coll := range b.collections {
		rwSets = append(rwSets, coll.Build())
	}
	return rwSets
}

// BuildNSReadWriteSets builds the read-write sets
func (b *NamespaceBuilder) BuildNSReadWriteSets() []byte {
	kvRWSet := &kvrwset.KVRWSet{}
	for key, version := range b.reads {
		kvRWSet.Reads = append(kvRWSet.Reads, &kvrwset.KVRead{Key: key, Version: version})
	}
	for key, value := range b.writes {
		kvRWSet.Writes = append(kvRWSet.Writes, &kvrwset.KVWrite{Key: key, Value: value, IsDelete: value == nil})
	}

	if b.marshalErr {
		return []byte("invalid proto buf")
	}

	bytes, err := proto.Marshal(kvRWSet)
	if err != nil {
		panic(err.Error())
	}
	return bytes
}

// BuildCollectionHashedRWSets builds the collection-hashed read-write sets
func (b *NamespaceBuilder) BuildCollectionHashedRWSets() []*rwset.CollectionHashedReadWriteSet {
	var collHashedRWSets []*rwset.CollectionHashedReadWriteSet
	for _, coll := range b.collections {
		collHashedRWSets = append(collHashedRWSets, &rwset.CollectionHashedReadWriteSet{
			CollectionName: coll.name,
			HashedRwset:    []byte("hashed-rw-set"),
			PvtRwsetHash:   []byte("pvt-rw-set-hash"),
		})
	}
	return collHashedRWSets
}

// BuildCollectionConfig builds the collection config package for the namespace
func (b *NamespaceBuilder) BuildCollectionConfig() *common.CollectionConfigPackage {
	cp := &common.CollectionConfigPackage{}
	for _, coll := range b.collections {
		config := coll.buildConfig()
		cp.Config = append(cp.Config, config)
	}
	return cp
}

// CollectionBuilder is a utility that builds a CollectionConfig and private data read/write sets for unit testing
type CollectionBuilder struct {
	name              string
	reads             map[string]*kvrwset.Version
	writes            map[string][]byte
	policy            string
	requiredPeerCount int32
	maximumPeerCount  int32
	blocksToLive      uint64
	collType          common.CollectionType
	marshalErr        bool
	ttl               string
}

// NewPvtReadWriteSetCollectionBuilder returns a new private read-write set collection builder
func NewPvtReadWriteSetCollectionBuilder(name string) *CollectionBuilder {
	return &CollectionBuilder{
		name:   name,
		reads:  make(map[string]*kvrwset.Version),
		writes: make(map[string][]byte),
	}
}

// Read adds a new read to the collection
func (c *CollectionBuilder) Read(key string, blockNum uint64, txIdx uint64) *CollectionBuilder {
	c.reads[key] = &kvrwset.Version{BlockNum: blockNum, TxNum: txIdx}
	return c
}

// Write adds a new write to the collection
func (c *CollectionBuilder) Write(key string, value []byte) *CollectionBuilder {
	c.writes[key] = value
	return c
}

// Delete adds a new write with 'IsDelete=true' to the collection
func (c *CollectionBuilder) Delete(key string) *CollectionBuilder {
	c.writes[key] = nil
	return c
}

// TransientConfig sets the transient collection config
func (c *CollectionBuilder) TransientConfig(policy string, requiredPeerCount, maximumPeerCount int32, ttl string) *CollectionBuilder {
	c.policy = policy
	c.requiredPeerCount = requiredPeerCount
	c.maximumPeerCount = maximumPeerCount
	c.collType = common.CollectionType_COL_TRANSIENT
	c.ttl = ttl
	return c
}

// StaticConfig sets the static collection config
func (c *CollectionBuilder) StaticConfig(policy string, requiredPeerCount, maximumPeerCount int32, btl uint64) *CollectionBuilder {
	c.policy = policy
	c.requiredPeerCount = requiredPeerCount
	c.maximumPeerCount = maximumPeerCount
	c.blocksToLive = btl
	return c
}

// OffLedgerConfig sets the off-ledger collection config
func (c *CollectionBuilder) OffLedgerConfig(policy string, requiredPeerCount, maximumPeerCount int32, ttl string) *CollectionBuilder {
	c.policy = policy
	c.requiredPeerCount = requiredPeerCount
	c.maximumPeerCount = maximumPeerCount
	c.collType = common.CollectionType_COL_OFFLEDGER
	c.ttl = ttl
	return c
}

// DCASConfig sets the DCAS collection config
func (c *CollectionBuilder) DCASConfig(policy string, requiredPeerCount, maximumPeerCount int32, ttl string) *CollectionBuilder {
	c.policy = policy
	c.requiredPeerCount = requiredPeerCount
	c.maximumPeerCount = maximumPeerCount
	c.collType = common.CollectionType_COL_DCAS
	c.ttl = ttl
	return c
}

// WithMarshalError simulates a marshalling error
func (c *CollectionBuilder) WithMarshalError() *CollectionBuilder {
	c.marshalErr = true
	return c
}

// Build builds the collection private read-write set
func (c *CollectionBuilder) Build() *rwset.CollectionPvtReadWriteSet {
	return &rwset.CollectionPvtReadWriteSet{
		CollectionName: c.name,
		Rwset:          c.buildReadWriteSet(),
	}
}

func (c *CollectionBuilder) buildReadWriteSet() []byte {
	kvRWSet := &kvrwset.KVRWSet{}
	for key, version := range c.reads {
		kvRWSet.Reads = append(kvRWSet.Reads, &kvrwset.KVRead{Key: key, Version: version})
	}
	for key, value := range c.writes {
		kvRWSet.Writes = append(kvRWSet.Writes, &kvrwset.KVWrite{Key: key, Value: value, IsDelete: value == nil})
	}

	if c.marshalErr {
		return []byte("invalid proto buf")
	}

	bytes, err := proto.Marshal(kvRWSet)
	if err != nil {
		panic(err.Error())
	}
	return bytes
}

func (c *CollectionBuilder) buildConfig() *common.CollectionConfig {
	signaturePolicyEnvelope, err := cauthdsl.FromString(c.policy)
	if err != nil {
		panic(err.Error())
	}

	return &common.CollectionConfig{
		Payload: &common.CollectionConfig_StaticCollectionConfig{
			StaticCollectionConfig: &common.StaticCollectionConfig{
				Type: c.collType,
				Name: c.name,
				MemberOrgsPolicy: &common.CollectionPolicyConfig{
					Payload: &common.CollectionPolicyConfig_SignaturePolicy{
						SignaturePolicy: signaturePolicyEnvelope,
					},
				},
				RequiredPeerCount: c.requiredPeerCount,
				MaximumPeerCount:  c.maximumPeerCount,
				BlockToLive:       c.blocksToLive,
				TimeToLive:        c.ttl,
			},
		},
	}
}
