/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"sync"
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/ledger/util/leveldbhelper"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
)

// todo add pinning script to include copied code into this file, original file from fabric is found in fabric/core/ledger/pvtdatastorage/store_imp.go
// todo below functions are originally unexported, the pinning script must capitalize these functions to export them

var logger = flogging.MustGetLogger("collelgproc")

type CollElgProc struct {
	notification, procComplete chan bool
	purgerLock                 *sync.Mutex
	db                         *leveldbhelper.DBHandle
}

func NewCollElgProc(purgerLock *sync.Mutex, missingKeysIndexDB *leveldbhelper.DBHandle) *CollElgProc {

	return &CollElgProc{
		notification: make(chan bool, 1),
		procComplete: make(chan bool, 1),
		purgerLock:   purgerLock,
		db:           missingKeysIndexDB,
	}
}

func (c *CollElgProc) notify() {
	select {
	case c.notification <- true:
		logger.Debugf("Signaled to collection eligibility processing routine")
	default: //noop
		logger.Debugf("Previous signal still pending. Skipping new signal")
	}
}

func (c *CollElgProc) waitForNotification() {
	<-c.notification
}

func (c *CollElgProc) done() {
	select {
	case c.procComplete <- true:
	default:
	}
}

func (c *CollElgProc) WaitForDone() {
	<-c.procComplete
}

func (c *CollElgProc) LaunchCollElgProc() {
	maxBatchSize := ledgerconfig.GetPvtdataStoreCollElgProcMaxDbBatchSize()
	batchesInterval := ledgerconfig.GetPvtdataStoreCollElgProcDbBatchesInterval()
	go func() {
		c.processCollElgEvents(maxBatchSize, batchesInterval) // process collection eligibility events when store is opened - in case there is an unprocessed events from previous run
		for {
			logger.Debugf("Waiting for collection eligibility event")
			c.waitForNotification()
			c.processCollElgEvents(maxBatchSize, batchesInterval)
			c.done()
		}
	}()
}

func (c *CollElgProc) processCollElgEvents(maxBatchSize, batchesInterval int) {
	logger.Debugf("Starting to process collection eligibility events")
	c.purgerLock.Lock()
	defer c.purgerLock.Unlock()
	collElgStartKey, collElgEndKey := createRangeScanKeysForCollElg()
	eventItr := c.db.GetIterator(collElgStartKey, collElgEndKey)
	defer eventItr.Release()
	batch := leveldbhelper.NewUpdateBatch()
	totalEntriesConverted := 0

	for eventItr.Next() {
		collElgKey, collElgVal := eventItr.Key(), eventItr.Value()
		blkNum := decodeCollElgKey(collElgKey)
		CollElgInfo, err := decodeCollElgVal(collElgVal)
		logger.Debugf("Processing collection eligibility event [blkNum=%d], CollElgInfo=%s", blkNum, CollElgInfo)
		if err != nil {
			logger.Errorf("This error is not expected %s", err)
			continue
		}
		for ns, colls := range CollElgInfo.NsCollMap {
			var coll string
			for _, coll = range colls.Entries {
				logger.Infof("Converting missing data entries from ineligible to eligible for [ns=%s, coll=%s]", ns, coll)
				startKey, endKey := createRangeScanKeysForIneligibleMissingData(blkNum, ns, coll)
				collItr := c.db.GetIterator(startKey, endKey)
				collEntriesConverted := 0

				for collItr.Next() { // each entry
					originalKey, originalVal := collItr.Key(), collItr.Value()
					modifiedKey := decodeMissingDataKey(originalKey)
					modifiedKey.IsEligible = true
					batch.Delete(originalKey)
					copyVal := make([]byte, len(originalVal))
					copy(copyVal, originalVal)
					batch.Put(EncodeMissingDataKey(modifiedKey), copyVal)
					collEntriesConverted++
					if batch.Len() > maxBatchSize {
						if err := c.db.WriteBatch(batch, true); err != nil {
							logger.Error(err.Error())
						}
						batch = leveldbhelper.NewUpdateBatch()
						sleepTime := time.Duration(batchesInterval)
						logger.Infof("Going to sleep for %d milliseconds between batches. Entries for [ns=%s, coll=%s] converted so far = %d",
							sleepTime, ns, coll, collEntriesConverted)
						c.purgerLock.Unlock()
						time.Sleep(sleepTime * time.Millisecond)
						c.purgerLock.Lock()
					}
				} // entry loop

				collItr.Release()
				logger.Infof("Converted all [%d] entries for [ns=%s, coll=%s]", collEntriesConverted, ns, coll)
				totalEntriesConverted += collEntriesConverted
			} // coll loop
		} // ns loop
		batch.Delete(collElgKey) // delete the collection eligibility event key as well
	} // event loop

	if err := c.db.WriteBatch(batch, true); err != nil {
		logger.Error(err.Error())
	}
	logger.Debugf("Converted [%d] inelligible mising data entries to elligible", totalEntriesConverted)
}
