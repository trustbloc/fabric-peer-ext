/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package dbstore

import (
	"github.com/hyperledger/fabric/common/ledger/util/leveldbhelper"
	"github.com/trustbloc/fabric-peer-ext/pkg/config"
)

// DBProvider provides an handle to a transientdata db
type DBProvider interface {
	OpenDBStore(id string) (DBStore, error)
	Close()
}

// LevelDBProvider provides an handle to a transientdata db
type LevelDBProvider struct {
	leveldbProvider *leveldbhelper.Provider
}

// NewDBProvider constructs new db provider
func NewDBProvider() (*LevelDBProvider, error) {
	dbPath := config.GetTransientDataLevelDBPath()
	logger.Debugf("constructing DBProvider dbPath=%s", dbPath)
	p, err := newLevelDBProvider(dbPath)
	if err != nil {
		return nil, err
	}
	return &LevelDBProvider{leveldbProvider: p}, nil
}

// OpenDBStore opens the db store
func (p *LevelDBProvider) OpenDBStore(dbName string) (*DBStore, error) {
	indexStore := p.leveldbProvider.GetDBHandle(dbName)
	return newDBStore(indexStore, dbName), nil
}

// Close cleans up the Provider
func (p *LevelDBProvider) Close() {
	p.leveldbProvider.Close()
}

var newLevelDBProvider = func(dbPath string) (*leveldbhelper.Provider, error) {
	return leveldbhelper.NewProvider(&leveldbhelper.Conf{DBPath: dbPath})
}
