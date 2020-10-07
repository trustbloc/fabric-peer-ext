/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package testutil

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"
	"github.com/hyperledger/fabric/core/ledger/pvtdatastorage"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/testutil"
)

//SetupExtTestEnv creates new couchdb instance for test
//returns couchdbd address, cleanup and stop function handle.
func SetupExtTestEnv() (addr string, cleanup func(string), stop func()) {
	return testutil.SetupExtTestEnv()
}

// SetupResources sets up all of the mock resource providers
func SetupResources() func() {
	return testutil.SetupResources()
}

// GetExtStateDBProvider returns the implementation of the versionedDBProvider
func GetExtStateDBProvider(t testing.TB, dbProvider statedb.VersionedDBProvider) statedb.VersionedDBProvider {
	return nil
}

// TestLedgerConf return the ledger configs
func TestLedgerConf() *ledger.Config {
	return testutil.TestLedgerConf()
}

// TestPrivateDataConf return the private data configs
func TestPrivateDataConf() *pvtdatastorage.PrivateDataConfig {
	return testutil.TestPrivateDataConf()
}

func SetupV2Data(t *testing.T, config *ledger.Config) func() {
	require.NoError(t, unzip("testdata/v20_couchdb/sample_ledgers/ledgersData.zip",
		config.RootFSPath, false))

	couchdbConfig, cleanup := startCouchDBWithV2Data(t, config.RootFSPath)
	config.StateDBConfig.StateDatabase = "CouchDB"
	config.StateDBConfig.CouchDB = couchdbConfig
	return cleanup
}

func startCouchDBWithV2Data(t *testing.T, ledgerFSRoot string) (*ledger.CouchDBConfig, func()) {
	// unzip couchdb data to prepare the mount dir
	couchdbDataUnzipDir := filepath.Join(ledgerFSRoot, "couchdbData")
	require.NoError(t, os.Mkdir(couchdbDataUnzipDir, os.ModePerm))
	require.NoError(t, unzip("testdata/v20_couchdb/sample_ledgers/couchdb.zip", couchdbDataUnzipDir, false))

	// start couchdb using couchdbDataUnzipDir and localdHostDir as mount dirs
	couchdbBinds := []string{
		fmt.Sprintf("%s:%s", couchdbDataUnzipDir, "/opt/couchdb/data"),
	}
	couchAddress, cleanup := statecouchdb.StartCouchDB(t, couchdbBinds)

	// set required config data to use state couchdb
	couchdbConfig := &ledger.CouchDBConfig{
		Address:             couchAddress,
		Username:            "admin",
		Password:            "adminpw",
		MaxRetries:          3,
		MaxRetriesOnStartup: 3,
		RequestTimeout:      10 * time.Second,
		RedoLogPath:         filepath.Join(ledgerFSRoot, "couchdbRedoLogs"),
	}

	return couchdbConfig, cleanup
}

func unzip(src string, dest string, createTopLevelDirInZip bool) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	// iterate all the dirs and files in the zip file
	for _, file := range r.File {
		filePath := file.Name
		if !createTopLevelDirInZip {
			// trim off the top level dir - for example, trim ledgersData/historydb/abc to historydb/abc
			index := strings.Index(filePath, string(filepath.Separator))
			filePath = filePath[index+1:]
		}

		fullPath := filepath.Join(dest, filePath)
		if file.FileInfo().IsDir() {
			os.MkdirAll(fullPath, os.ModePerm)
			continue
		}
		if err = os.MkdirAll(filepath.Dir(fullPath), os.ModePerm); err != nil {
			return err
		}
		outFile, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		rc, err := file.Open()
		if err != nil {
			return err
		}
		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

func MarbleValue(name, color, owner string, size int) string {
	return fmt.Sprintf(`{"color":"%s","docType":"marble","name":"%s","owner":"%s","size":%d}`, color, name, owner, size)
}

// Skip skips the unit test for extensions
func Skip(t *testing.T, msg string) {
	t.Skip(msg)
}

// InvokeOrSkip skips the given function.
func InvokeOrSkip(func()) {
	// Skip the function
}
