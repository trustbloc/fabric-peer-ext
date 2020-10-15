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
)

// todo add pinning script to include copied code into this file, original file from fabric is found in fabric/core/ledger/pvtdatastorage/store.go
// todo below functions are originally unexported, the pinning script must capitalize these functions to export them

var logger = flogging.MustGetLogger("collelgproc")

type CollElgProcSync struct {
	notification, procComplete chan bool
	purgerLock                 *sync.Mutex
	db                         *leveldbhelper.DBHandle
	batchesInterval            int
	maxBatchSize               int
}

func NewCollElgProcSync(purgerLock *sync.Mutex, missingKeysIndexDB *leveldbhelper.DBHandle, batchesInterval, maxBatchSize int) *CollElgProcSync {
	return &CollElgProcSync{
		notification:    make(chan bool, 1),
		procComplete:    make(chan bool, 1),
		purgerLock:      purgerLock,
		db:              missingKeysIndexDB,
		batchesInterval: batchesInterval,
		maxBatchSize:    maxBatchSize,
	}
}

func (c *CollElgProcSync) notify() {
	select {
	case c.notification <- true:
		logger.Debugf("Signaled to collection eligibility processing routine")
	default: //noop
		logger.Debugf("Previous signal still pending. Skipping new signal")
	}
}

func (c *CollElgProcSync) waitForNotification() {
	<-c.notification
}

func (c *CollElgProcSync) done() {
	select {
	case c.procComplete <- true:
	default:
	}
}

func (c *CollElgProcSync) WaitForDone() {
	<-c.procComplete
}

func (c *CollElgProcSync) LaunchCollElgProc() {
	go func() {
		if err := c.processCollElgEvents(); err != nil {
			// process collection eligibility events when store is opened -
			// in case there is an unprocessed events from previous run
			logger.Errorf("Failed to process collection eligibility events: %s", err)
		}
		for {
			logger.Debugf("Waiting for collection eligibility event")
			c.waitForNotification()
			if err := c.processCollElgEvents(); err != nil {
				logger.Errorw("failed to process collection eligibility events", "err", err)
			}
			c.done()
		}
	}()
}

func (c *CollElgProcSync) processCollElgEvents() error {
	logger.Debugf("Starting to process collection eligibility events")
	c.purgerLock.Lock()
	defer c.purgerLock.Unlock()
	collElgStartKey, collElgEndKey := createRangeScanKeysForCollElg()
	eventItr, err := c.db.GetIterator(collElgStartKey, collElgEndKey)
	if err != nil {
		return err
	}
	defer eventItr.Release()
	batch := c.db.NewUpdateBatch()
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
				startKey, endKey := createRangeScanKeysForInelgMissingData(blkNum, ns, coll)
				collItr, err := c.db.GetIterator(startKey, endKey)
				if err != nil {
					return err
				}
				collEntriesConverted := 0

				for collItr.Next() { // each entry
					originalKey, originalVal := collItr.Key(), collItr.Value()
					modifiedKey := DecodeInelgMissingDataKey(originalKey)
					batch.Delete(originalKey)
					copyVal := make([]byte, len(originalVal))
					copy(copyVal, originalVal)
					batch.Put(EncodeElgPrioMissingDataKey(modifiedKey), copyVal)
					collEntriesConverted++
					if batch.Len() > c.maxBatchSize {
						if err := c.db.WriteBatch(batch, true); err != nil {
							logger.Error(err.Error())
						}
						batch = c.db.NewUpdateBatch()
						sleepTime := time.Duration(c.batchesInterval)
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
	return nil
}
