/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validationresults

import (
	"fmt"
	"hash/fnv"
	"sync"

	"github.com/hyperledger/fabric/common/flogging"

	"github.com/trustbloc/fabric-peer-ext/pkg/common/txflags"
)

var logger = flogging.MustGetLogger("ext_validation")

// Results contains the validation results for the given block number.
type Results struct {
	BlockNumber uint64
	TxFlags     txflags.ValidationFlags
	TxIDs       []string
	Err         error
	Endpoint    string // Endpoint is the endpoint of the peer that provided the results.
	Local       bool   // If true then that means the results were generated locally and policy validation is not required
	MSPID       string
	Signature   []byte
	Identity    []byte
}

// Results returns a string representation of the validation results (used in logging).
func (vr *Results) String() string {
	if vr.Err == nil {
		return fmt.Sprintf("(MSP: [%s], Endpoint: [%s], Block: %d, TxFlags: %v)", vr.MSPID, vr.Endpoint, vr.BlockNumber, vr.TxFlags)
	}

	return fmt.Sprintf("(MSP: [%s], Endpoint: [%s], Block: %d, Err: %s)", vr.MSPID, vr.Endpoint, vr.BlockNumber, vr.Err)
}

// Cache is a cache of validation results segmented by TxFlags. One or more
// peers will validate a certain subset of transactions within a block. If two
// peers, say, validate the same set of transactions and come up with the exact
// same results then both of the results will be added to the same segment under
// the block.
type Cache struct {
	lock           sync.Mutex
	resultsByBlock map[uint64]results
}

// NewCache returns a new validation results cache
func NewCache() *Cache {
	return &Cache{
		resultsByBlock: make(map[uint64]results),
	}
}

// Add adds the given validation result to the cache under the appropriate
// block/segment and returns all validation results for the segment.
func (c *Cache) Add(result *Results) []*Results {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.get(result.BlockNumber).add(result)
}

// Remove removes validation results for the given block and deletes the results from any previous block
// TODO: Delete all previous blocks
func (c *Cache) Remove(blockNum uint64) []*Results {
	c.lock.Lock()
	defer c.lock.Unlock()

	segmentsForBlock, ok := c.resultsByBlock[blockNum]
	if !ok {
		return nil
	}

	var resultsForBlock []*Results
	for _, results := range segmentsForBlock {
		resultsForBlock = append(resultsForBlock, results...)
	}

	logger.Infof("Removing cached validation results for block %d", blockNum)

	delete(c.resultsByBlock, blockNum)

	return resultsForBlock
}

func (c *Cache) get(blockNum uint64) results {
	resultsForBlock, ok := c.resultsByBlock[blockNum]
	if !ok {
		resultsForBlock = make(results)
		c.resultsByBlock[blockNum] = resultsForBlock
	}
	return resultsForBlock
}

func hash(txFlags txflags.ValidationFlags) uint32 {
	h := fnv.New32a()
	_, err := h.Write(txFlags)
	if err != nil {
		panic(err.Error())
	}
	return h.Sum32()
}

type results map[uint32][]*Results

func (r results) add(result *Results) []*Results {
	segment := hash(result.TxFlags)

	results := append(r[segment], result)
	r[segment] = results

	return results
}
