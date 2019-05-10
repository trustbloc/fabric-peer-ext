/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transientstore

import (
	"sort"
	"sync"
)

// loadPvtRWSetMap is a loader function of store.cache
func loadPvtRWSetMap(key interface{}) (interface{}, error) {
	return &pvtRWSetMap{m: map[string]string{}}, nil
}

// pvtRWSetMap represents a cached element in store.cache
type pvtRWSetMap struct {
	mu sync.RWMutex
	m  map[string]string
}

//func (p *pvtRWSetMap) get(k string) string {
//	p.mu.RLock()
//	defer p.mu.RUnlock()
//
//	return p.m[k]
//}

func (p *pvtRWSetMap) set(k string, v string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.m[k] = v
}

func (p *pvtRWSetMap) length() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return len(p.m)
}

func (p *pvtRWSetMap) delete(k string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.m, k)
}

func (p *pvtRWSetMap) keys() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	keys := make([]string, 0, len(p.m))
	for k := range p.m {
		keys = append(keys, k)
	}
	return keys
}

// loadBlockHeight is a loader function of store.blockHeightCache
func loadBlockHeight(key interface{}) (interface{}, error) {
	return &txidsSlice{m: []string{}}, nil
}

// txidsSlice represents a cached element in store.blockHeightCache
type txidsSlice struct {
	mu sync.RWMutex
	m  []string
}

func (p *txidsSlice) add(v string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.m = append(p.m, v)
	sort.Strings(p.m) // ensures txids are sorted to help sort.search call in findTxidEntryInSlice below
}

// findTxidEntryInSlice will search for txid in txidsSlice
func (p *txidsSlice) findTxidEntryInSlice(txid string) (bool, int) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	i := sort.Search(len(p.m), func(i int) bool { return p.m[i] >= txid }) //  equivalent to sort.SearchStrings()
	return i < len(p.m) && p.m[i] == txid, i
}

// removeTxidEntryAtIndex removes an entry at index in the txidsSlice
func (p *txidsSlice) removeTxidEntryAtIndex(index int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.m = append(p.m[:index], p.m[index+1:]...)
}

func (p *txidsSlice) length() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return len(p.m)
}

func (p *txidsSlice) getTxids() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.m
}

// loadTxid is a loader function of store.txidCache
func loadTxid(key interface{}) (interface{}, error) {
	return &blockHeightsSlice{m: []uint64{}}, nil
}

// blockHeightsSlice represents a cached element in store.txidCache
type blockHeightsSlice struct {
	mu sync.RWMutex
	m  []uint64
}

func (p *blockHeightsSlice) add(v uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.m = append(p.m, v)
	sort.Slice(p.m, func(i, j int) bool { return p.m[i] < p.m[j] }) // ensures blockHeights are sorted to help sort.search call in findBlockHeightEntryInSlice below
}

// findBlockHeightEntryInSlice will search for a blockheight (uint64) entry in blockHeightsSlice
func (p *blockHeightsSlice) findBlockHeightEntryInSlice(blockHeight uint64) (bool, int) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	i := sort.Search(len(p.m), func(i int) bool { return p.m[i] >= blockHeight })
	return i < len(p.m) && p.m[i] == blockHeight, i
}

// removeBlockHeightEntryAtIndex removes an entry at index in blockHeightsSlice
func (p *blockHeightsSlice) removeBlockHeightEntryAtIndex(index int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.m = append(p.m[:index], p.m[index+1:]...)
}

func (p *blockHeightsSlice) length() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return len(p.m)
}

func (p *blockHeightsSlice) getBlockHeights() []uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.m
}

// sliceUniqueString will strip out any duplicate entries from a slice s (of type string)
func sliceUniqueString(s []string) []string {
	seen := make(map[string]struct{}, len(s))
	j := 0
	for _, v := range s {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		s[j] = v
		j++
	}
	return s[:j]
}

// sliceUniqueUint64 will strip out any duplicate entries from a slice s (of type uint64)
func sliceUniqueUint64(s []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(s))
	j := 0
	for _, v := range s {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		s[j] = v
		j++
	}
	return s[:j]
}
