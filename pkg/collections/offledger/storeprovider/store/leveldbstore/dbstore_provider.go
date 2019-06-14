/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package leveldbstore

import (
	"fmt"
	"sync"
	"time"

	"github.com/hyperledger/fabric/common/ledger/util/leveldbhelper"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/storeprovider/store/api"
	"github.com/trustbloc/fabric-peer-ext/pkg/config"
)

// LevelDBProvider provides a handle to a db
type LevelDBProvider struct {
	leveldbProvider *leveldbhelper.Provider
	stores          map[string]*store
	mutex           sync.RWMutex
	done            chan struct{}
	closed          bool
}

// NewDBProvider constructs new db provider
func NewDBProvider() *LevelDBProvider {
	dbPath := config.GetOLCollLevelDBPath()
	logger.Debugf("constructing DBProvider dbPath=%s", dbPath)
	p := &LevelDBProvider{
		stores: make(map[string]*store),
		done:   make(chan struct{}),
		leveldbProvider: leveldbhelper.NewProvider(
			&leveldbhelper.Conf{
				DBPath: dbPath,
			},
		),
	}

	p.periodicPurge()

	return p
}

// GetDB opens the db store
func (p *LevelDBProvider) GetDB(channelID string, coll string, ns string) (api.DB, error) {
	dbName := dbName(ns, coll)

	p.mutex.RLock()
	s, ok := p.stores[dbName]
	p.mutex.RUnlock()

	if ok {
		return s, nil
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	s, ok = p.stores[dbName]
	if !ok {
		indexStore := p.leveldbProvider.GetDBHandle(dbName)
		s = newDBStore(indexStore, dbName)
		p.stores[dbName] = s
	}

	return s, nil
}

// Close cleans up the Provider
func (p *LevelDBProvider) Close() {
	p.leveldbProvider.Close()

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.closed {
		p.done <- struct{}{}
		p.closed = true
	}
}

func (p *LevelDBProvider) getStores() []*store {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	var stores []*store
	for _, s := range p.stores {
		stores = append(stores, s)
	}
	return stores
}

func (p *LevelDBProvider) periodicPurge() {
	ticker := time.NewTicker(config.GetOLCollExpirationCheckInterval())
	go func() {
		for {
			select {
			case <-ticker.C:
				for _, s := range p.getStores() {
					err := s.DeleteExpiredKeys()
					if err != nil {
						logger.Errorf("Error deleting expired keys for [%s]", s.dbName)
					}
				}
			case <-p.done:
				logger.Infof("Periodic purge is exiting")
				return
			}
		}
	}()
}

func dbName(ns, coll string) string {
	return fmt.Sprintf("%s$%s", ns, coll)
}
