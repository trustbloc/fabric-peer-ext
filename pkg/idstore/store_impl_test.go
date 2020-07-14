/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idstore

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric/common/ledger/dataformat"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/ledger/kvledger/msgs"
	couchdb "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"
	"github.com/hyperledger/fabric/extensions/testutil"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
	xtestutil "github.com/trustbloc/fabric-peer-ext/pkg/testutil"
)

var couchDBConfig *ledger.CouchDBConfig
var destroy func()

// Ensure the roles are initialized
var _ = roles.GetRoles()

func TestMain(m *testing.M) {
	//setup extension test environment
	_, _, destroy = xtestutil.SetupExtTestEnv()
	defer destroy()

	// Create CouchDB definition from config parameters
	couchDBConfig = xtestutil.TestLedgerConf().StateDBConfig.CouchDB

	code := m.Run()
	os.Exit(code)
}

func TestOpenIDStore(t *testing.T) {
	t.Run("test error from createCouchInstance", func(t *testing.T) {
		_, err := OpenIDStore(nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "ledgerconfig is nil")
	})

	t.Run("test error from CreateCouchDatabase", func(t *testing.T) {
		v := systemID
		defer func() { systemID = v }()
		systemID = "_"
		_, err := OpenIDStore(testutil.TestLedgerConf())
		require.Error(t, err)
		require.Contains(t, err.Error(), "create new couchdb database failed")
	})

	t.Run("test error from NewCouchDatabase", func(t *testing.T) {
		roles.IsEndorser()
		rolesValue := make(map[roles.Role]struct{})
		rolesValue[roles.EndorserRole] = struct{}{}
		roles.SetRoles(rolesValue)
		v := systemID
		defer func() {
			systemID = v
			roles.SetRoles(nil)
		}()
		systemID = "_"
		_, err := OpenIDStore(testutil.TestLedgerConf())
		require.Error(t, err)
		require.Contains(t, err.Error(), "new couchdb database")
	})

}

func TestNewCommitterStore(t *testing.T) {
	t.Run("isEmpty error", func(t *testing.T) {
		errExpected := fmt.Errorf("injected isEmpty error")

		restore := isEmpty
		isEmpty = func(couchInstance *couchdb.CouchInstance) (bool, error) { return false, errExpected }
		defer func() { isEmpty = restore }()

		_, err := newCommitterStore("inventory", &couchdb.CouchInstance{},
			func(couchInstance *couchdb.CouchInstance, dbName string) (couchDatabase, error) {
				return mockCouchDB{}, nil
			},
			createMetadataDoc,
		)
		require.EqualError(t, err, errExpected.Error())
	})

	t.Run("createMetadataDoc error", func(t *testing.T) {
		errExpected := fmt.Errorf("injected createMetadataDoc error")

		restore := isEmpty
		isEmpty = func(couchInstance *couchdb.CouchInstance) (bool, error) { return true, nil }
		defer func() { isEmpty = restore }()

		_, err := newCommitterStore("inventory", &couchdb.CouchInstance{},
			func(couchInstance *couchdb.CouchInstance, dbName string) (couchDatabase, error) {
				return mockCouchDB{}, nil
			},
			func(constructionLedger, format string) ([]byte, error) {
				return nil, errExpected
			},
		)
		require.EqualError(t, err, errExpected.Error())
	})

	t.Run("test error from ExistsWithRetry", func(t *testing.T) {
		restore := isEmpty
		isEmpty = func(couchInstance *couchdb.CouchInstance) (bool, error) { return true, nil }
		defer func() { isEmpty = restore }()

		_, err := newCommitterStore("inventory", &couchdb.CouchInstance{},
			func(couchInstance *couchdb.CouchInstance, dbName string) (couchDatabase, error) {
				return mockCouchDB{createNewIndexWithRetryErr: fmt.Errorf("ExistsWithRetry error")}, nil
			},
			createMetadataDoc,
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "create couchdb index failed")
	})
}

func TestCreateCouchInstance(t *testing.T) {
	t.Run("test error from CreateCouchInstance", func(t *testing.T) {
		_, err := createCouchInstance(&ledger.Config{StateDBConfig: &ledger.StateDBConfig{CouchDB: &ledger.CouchDBConfig{}}})
		require.Error(t, err)
		require.Contains(t, err.Error(), "obtaining CouchDB instance failed")
	})
}

func TestNewStore(t *testing.T) {
	t.Run("test error from ExistsWithRetry", func(t *testing.T) {
		_, err := newStore("", &couchdb.CouchInstance{},
			func(couchInstance *couchdb.CouchInstance, dbName string) (couchDatabase, error) {
				return mockCouchDB{existsWithRetryErr: fmt.Errorf("ExistsWithRetry error")}, nil
			},
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "ExistsWithRetry error")
	})

	t.Run("test db not exists", func(t *testing.T) {
		_, err := newStore("", &couchdb.CouchInstance{},
			func(couchInstance *couchdb.CouchInstance, dbName string) (couchDatabase, error) {
				return mockCouchDB{existsWithRetryValue: false}, nil
			},
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "DB not found")
	})

	t.Run("test error from IndexDesignDocExistsWithRetry", func(t *testing.T) {
		_, err := newStore("", &couchdb.CouchInstance{},
			func(couchInstance *couchdb.CouchInstance, dbName string) (couchDatabase, error) {
				return mockCouchDB{existsWithRetryValue: true, indexDesignDocExistsWithRetryErr: fmt.Errorf("IndexDesignDocExistsWithRetry error")}, nil
			},
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "IndexDesignDocExistsWithRetry error")
	})

	t.Run("test index not exists", func(t *testing.T) {
		_, err := newStore("", &couchdb.CouchInstance{},
			func(couchInstance *couchdb.CouchInstance, dbName string) (couchDatabase, error) {
				return mockCouchDB{existsWithRetryValue: true, indexDesignDocExistsWithRetryValue: false}, nil
			},
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "DB index not found")
	})
}

func TestUnderConstructionFlag(t *testing.T) {
	const ledgerID = "testunderconstructiongflag"

	t.Run("test error from SetUnderConstructionFlag SaveDoc", func(t *testing.T) {
		env := NewTestStoreEnv(t, ledgerID, couchDBConfig)
		env.TestStore.db = mockCouchDB{saveDocErr: fmt.Errorf("SaveDoc error")}
		err := env.TestStore.SetUnderConstructionFlag(ledgerID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "update of metadata in CouchDB failed")

	})

	t.Run("test error from UnsetUnderConstructionFlag SaveDoc", func(t *testing.T) {
		env := NewTestStoreEnv(t, ledgerID, couchDBConfig)
		env.TestStore.db = mockCouchDB{saveDocErr: fmt.Errorf("SaveDoc error")}
		err := env.TestStore.UnsetUnderConstructionFlag()
		require.Error(t, err)
		require.Contains(t, err.Error(), "update of metadata in CouchDB failed")

	})

	t.Run("test error from SetUnderConstructionFlag SaveDoc", func(t *testing.T) {
		env := NewTestStoreEnv(t, ledgerID, couchDBConfig)
		env.TestStore.db = mockCouchDB{saveDocErr: fmt.Errorf("SaveDoc error")}
		err := env.TestStore.SetUnderConstructionFlag(ledgerID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "update of metadata in CouchDB failed")

	})

	t.Run("test error from GetUnderConstructionFlag ReadDoc", func(t *testing.T) {
		env := NewTestStoreEnv(t, ledgerID, couchDBConfig)
		env.TestStore.db = mockCouchDB{readDocErr: fmt.Errorf("SaveDoc error")}
		_, err := env.TestStore.GetUnderConstructionFlag()
		require.Error(t, err)
		require.Contains(t, err.Error(), "retrieval of metadata from CouchDB inventory failed")

	})

	t.Run("test doc is empty", func(t *testing.T) {
		env := NewTestStoreEnv(t, ledgerID, couchDBConfig)
		env.TestStore.db = mockCouchDB{}
		value, err := env.TestStore.GetUnderConstructionFlag()
		require.NoError(t, err)
		require.Empty(t, value)
	})

	t.Run("test metadata is invalid", func(t *testing.T) {
		env := NewTestStoreEnv(t, ledgerID, couchDBConfig)
		env.TestStore.db = mockCouchDB{readDocValue: &couchdb.CouchDoc{}}
		_, err := env.TestStore.GetUnderConstructionFlag()
		require.Error(t, err)
		require.Contains(t, err.Error(), "metadata in CouchDB inventory is invalid")

	})

	t.Run("test metadata under construction key is invalid", func(t *testing.T) {
		env := NewTestStoreEnv(t, ledgerID, couchDBConfig)
		v, err := json.Marshal(make(map[string]interface{}))
		require.NoError(t, err)
		env.TestStore.db = mockCouchDB{readDocValue: &couchdb.CouchDoc{JSONValue: v}}
		_, err = env.TestStore.GetUnderConstructionFlag()
		require.Error(t, err)
		require.Contains(t, err.Error(), "metadata under construction key in CouchDB inventory is invalid")
	})

	t.Run("SetUnderConstructionFlag createMetadataDoc error", func(t *testing.T) {
		errExpected := fmt.Errorf("injected createMetadataDoc error")
		env := NewTestStoreEnv(t, ledgerID, couchDBConfig)
		env.TestStore.createMetadataDoc = func(constructionLedger, format string) ([]byte, error) { return nil, errExpected }
		v, err := json.Marshal(make(map[string]interface{}))
		require.NoError(t, err)
		env.TestStore.db = mockCouchDB{readDocValue: &couchdb.CouchDoc{JSONValue: v}}
		require.EqualError(t, env.TestStore.SetUnderConstructionFlag(ledgerID), errExpected.Error())
	})

	t.Run("SetUnderConstructionFlag getMetadata error", func(t *testing.T) {
		errExpected := fmt.Errorf("injected getMetadata error")
		env := NewTestStoreEnv(t, ledgerID, couchDBConfig)
		env.TestStore.db = mockCouchDB{readDocErr: errExpected}
		err := env.TestStore.SetUnderConstructionFlag(ledgerID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "retrieval of metadata from CouchDB inventory failed")
	})

	t.Run("UnsetUnderConstructionFlag getMetadata error", func(t *testing.T) {
		errExpected := fmt.Errorf("injected getMetadata error")
		env := NewTestStoreEnv(t, ledgerID, couchDBConfig)
		env.TestStore.db = mockCouchDB{readDocErr: errExpected}
		err := env.TestStore.UnsetUnderConstructionFlag()
		require.Error(t, err)
		require.Contains(t, err.Error(), "retrieval of metadata from CouchDB inventory failed")
	})

	t.Run("SetUnderConstructionFlag store closed", func(t *testing.T) {
		env := NewTestStoreEnv(t, ledgerID, couchDBConfig)
		env.TestStore.Close()
		require.EqualError(t, env.TestStore.SetUnderConstructionFlag(ledgerID), "ID store is closed")
	})

	t.Run("UnsetUnderConstructionFlag store closed", func(t *testing.T) {
		env := NewTestStoreEnv(t, ledgerID, couchDBConfig)
		env.TestStore.Close()
		require.EqualError(t, env.TestStore.UnsetUnderConstructionFlag(), "ID store is closed")
	})

	t.Run("GetUnderConstructionFlag store closed", func(t *testing.T) {
		env := NewTestStoreEnv(t, ledgerID, couchDBConfig)
		env.TestStore.Close()
		_, err := env.TestStore.GetUnderConstructionFlag()
		require.EqualError(t, err, "ID store is closed")
	})

	t.Run("test success", func(t *testing.T) {
		env := NewTestStoreEnv(t, ledgerID, couchDBConfig)
		req := require.New(t)
		store := env.TestStore
		// set under construction flag
		req.NoError(store.SetUnderConstructionFlag(ledgerID))

		// get under construction flag should exist
		value, err := store.GetUnderConstructionFlag()
		req.NoError(err)
		req.Equal(ledgerID, value)

		// unset under construction flag
		req.NoError(store.UnsetUnderConstructionFlag())

		// get under construction flag should not exist after unset
		value, err = store.GetUnderConstructionFlag()
		req.NoError(err)
		req.Empty(value)
	})
}

func TestLedgerID(t *testing.T) {
	t.Run("test error from CreateLedgerID closed", func(t *testing.T) {
		env := NewTestStoreEnv(t, "", couchDBConfig)
		env.TestStore.Close()
		require.EqualError(t, env.TestStore.CreateLedgerID("", nil), "ID store is closed")
	})

	t.Run("test error from CreateLedgerID LedgerIDExists", func(t *testing.T) {
		env := NewTestStoreEnv(t, "", couchDBConfig)
		env.TestStore.db = mockCouchDB{readDocErr: fmt.Errorf("ReadDoc error")}
		err := env.TestStore.CreateLedgerID("", nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "ReadDoc error")

	})

	t.Run("test error from CreateLedgerID BatchUpdateDocuments", func(t *testing.T) {
		env := NewTestStoreEnv(t, "", couchDBConfig)
		env.TestStore.db = mockCouchDB{batchUpdateDocumentsErr: fmt.Errorf("BatchUpdateDocuments error")}
		block := &common.Block{}
		block.Data = &common.BlockData{Data: [][]byte{[]byte("testblock")}}
		err := env.TestStore.CreateLedgerID("", block)
		require.Error(t, err)
		require.Contains(t, err.Error(), "creation of ledger failed ")

	})

	t.Run("test error from CreateLedgerID BatchUpdateDocuments", func(t *testing.T) {
		env := NewTestStoreEnv(t, "", couchDBConfig)
		env.TestStore.db = mockCouchDB{batchUpdateDocumentsErr: fmt.Errorf("BatchUpdateDocuments error")}
		block := &common.Block{}
		block.Data = &common.BlockData{Data: [][]byte{[]byte("testblock")}}
		err := env.TestStore.CreateLedgerID("", block)
		require.Error(t, err)
		require.Contains(t, err.Error(), "creation of ledger failed ")

	})

	t.Run("test error from GetLedgeIDValue ReadDoc", func(t *testing.T) {
		env := NewTestStoreEnv(t, "", couchDBConfig)
		env.TestStore.db = mockCouchDB{readDocErr: fmt.Errorf("ReadDoc error")}
		_, _, _, err := env.TestStore.getLedgerIDValue("")
		require.Error(t, err)
		require.Contains(t, err.Error(), "ReadDoc error")

	})

	t.Run("test GetLedgerIDValue ReadDoc return empty doc", func(t *testing.T) {
		env := NewTestStoreEnv(t, "", couchDBConfig)
		env.TestStore.db = mockCouchDB{readDocValue: &couchdb.CouchDoc{}}
		_, v, _, err := env.TestStore.getLedgerIDValue("")
		require.NoError(t, err)
		require.Empty(t, v)

	})

	t.Run("test error from GetAllLedgerIds queryInventory", func(t *testing.T) {
		env := NewTestStoreEnv(t, "", couchDBConfig)
		env.TestStore.db = mockCouchDB{queryDocumentsErr: fmt.Errorf("QueryDocuments error")}
		_, err := env.TestStore.GetActiveLedgerIDs()
		require.Error(t, err)
		require.Contains(t, err.Error(), "QueryDocuments error")
	})

	t.Run("test error from GetAllLedgerIds couchValueToJSON", func(t *testing.T) {
		env := NewTestStoreEnv(t, "", couchDBConfig)
		env.TestStore.db = mockCouchDB{queryDocumentsValue: []*couchdb.QueryResult{{Value: []byte("wrongData")}}}
		_, err := env.TestStore.GetActiveLedgerIDs()
		require.Error(t, err)
		require.Contains(t, err.Error(), "couchValueToJSON failed")
	})

	t.Run("test empty doc from GetAllLedgerIds queryInventory", func(t *testing.T) {
		env := NewTestStoreEnv(t, "", couchDBConfig)
		v, err := json.Marshal(make(map[string]interface{}))
		require.NoError(t, err)
		env.TestStore.db = mockCouchDB{queryDocumentsValue: []*couchdb.QueryResult{{Value: v}}}
		_, err = env.TestStore.GetActiveLedgerIDs()
		require.Error(t, err)
		require.Contains(t, err.Error(), "ledger inventory document is invalid")
	})

	t.Run("test wrong doc from GetAllLedgerIds queryInventory", func(t *testing.T) {
		env := NewTestStoreEnv(t, "", couchDBConfig)
		m := make(map[string]interface{})
		m[inventoryNameLedgerIDField] = 1
		v, err := json.Marshal(m)
		require.NoError(t, err)
		env.TestStore.db = mockCouchDB{queryDocumentsValue: []*couchdb.QueryResult{{Value: v}}}
		_, err = env.TestStore.GetActiveLedgerIDs()
		require.Error(t, err)
		require.Contains(t, err.Error(), "ledger inventory document value is invalid")
	})

	t.Run("test error from GetActiveLedgerIDs closed", func(t *testing.T) {
		env := NewTestStoreEnv(t, "", couchDBConfig)
		env.TestStore.Close()
		_, err := env.TestStore.GetActiveLedgerIDs()
		require.EqualError(t, err, "ID store is closed")
	})

	t.Run("test error from LedgerIDActive", func(t *testing.T) {
		errExpected := fmt.Errorf("injected ReadDoc error")
		env := NewTestStoreEnv(t, "", couchDBConfig)
		env.TestStore.db = mockCouchDB{readDocErr: errExpected}
		_, _, err := env.TestStore.LedgerIDActive("")
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
	})

	t.Run("test error from GetGenesisBlock", func(t *testing.T) {
		errExpected := fmt.Errorf("injected error")
		env := NewTestStoreEnv(t, "", couchDBConfig)
		env.TestStore.db = mockCouchDB{readDocErr: errExpected}
		_, err := env.TestStore.GetGenesisBlock("")
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
	})

	t.Run("test error from GetGenesisBlock - store closed", func(t *testing.T) {
		env := NewTestStoreEnv(t, "", couchDBConfig)
		env.TestStore.Close()
		_, err := env.TestStore.GetGenesisBlock("")
		require.EqualError(t, err, "ID store is closed")
	})

	t.Run("test success", func(t *testing.T) {
		ledgerID := "testledgerid"
		ledgerID1 := "testledgerid1"
		env := NewTestStoreEnv(t, ledgerID, couchDBConfig)
		req := require.New(t)
		store := env.TestStore

		block := &common.Block{}
		block.Data = &common.BlockData{Data: [][]byte{[]byte("testblock")}}

		// create ledger id
		req.NoError(store.CreateLedgerID(ledgerID, block))
		req.NoError(store.CreateLedgerID(ledgerID1, block))

		// create exist ledger id should fail
		req.Error(store.CreateLedgerID(ledgerID, block))

		// get ledger ids
		ledgerIDs, err := store.GetActiveLedgerIDs()
		req.NoError(err)
		req.Equal(2, len(ledgerIDs))
		req.Contains(ledgerIDs, ledgerID)
		req.Contains(ledgerIDs, ledgerID1)

		// get ledger id value
		_, ledgerIdValue, _, err := store.getLedgerIDValue(ledgerID)
		req.NoError(err)
		gb := &common.Block{}
		req.NoError(proto.Unmarshal(ledgerIdValue, gb))
		req.Equal("testblock", string(gb.Data.Data[0]))

		//check ledger id exist
		exist, err := store.LedgerIDExists(ledgerID)
		req.NoError(err)
		req.Equal(exist, true)

		active, exist, err := store.LedgerIDActive(ledgerID)
		req.NoError(err)
		req.True(active)
		req.True(exist)

		b, err := store.GetGenesisBlock(ledgerID)
		req.NoError(err)
		req.Equal(block.Metadata, b.Metadata)
		req.Equal(block.Header, b.Header)
		req.Equal(block.Data.Data, b.Data.Data)

		f, err := store.GetFormat()
		req.NoError(err)
		req.Equal([]byte(dataformat.CurrentFormat), f)

		store.Close()
		require.NotPanicsf(t, func() { store.Close() }, "Close() should be able to be called multiple times")

		_, err = env.TestStore.LedgerIDExists(ledgerID)
		require.EqualError(t, err, "ID store is closed")

		_, _, err = env.TestStore.LedgerIDActive(ledgerID)
		require.EqualError(t, err, "ID store is closed")
	})
}

func TestClose(t *testing.T) {
	ledgerID := "testclose"
	env := NewTestStoreEnv(t, ledgerID, couchDBConfig)
	store := env.TestStore

	require.False(t, store.isClosed())
	store.Close()
	require.True(t, store.isClosed())
}

func TestOpenStoreWithEndorserRole(t *testing.T) {
	// create committer store
	ledgerID := "testopenstorewithendorserrole"
	env := NewTestStoreEnv(t, ledgerID, couchDBConfig)

	// create endorser store
	rolesValue := make(map[roles.Role]struct{})
	rolesValue[roles.EndorserRole] = struct{}{}
	roles.SetRoles(rolesValue)
	defer func() { roles.SetRoles(nil) }()
	env = NewTestStoreEnv(t, ledgerID, couchDBConfig)
	req := require.New(t)
	endorserStore := env.TestStore

	// set under construction flag
	req.NoError(endorserStore.SetUnderConstructionFlag(ledgerID))

	// get under construction flag should exist
	value, err := endorserStore.GetUnderConstructionFlag()
	req.NoError(err)
	req.Equal(ledgerID, value)
}

func TestStore_UpdateLedgerStatus(t *testing.T) {
	ledgerID := "testupdateledgerstatus"
	env := NewTestStoreEnv(t, ledgerID, couchDBConfig)
	req := require.New(t)
	store := env.TestStore

	active, exists, err := store.LedgerIDActive(ledgerID)
	require.NoError(t, err)
	require.False(t, active)
	require.False(t, exists)

	req.EqualError(store.UpdateLedgerStatus(ledgerID, msgs.Status_INACTIVE), "ledger ID does not exist")

	block := &common.Block{}
	block.Data = &common.BlockData{Data: [][]byte{[]byte("testblock")}}

	require.NoError(t, store.CreateLedgerID(ledgerID, block))

	active, exists, err = store.LedgerIDActive(ledgerID)
	require.NoError(t, err)
	require.True(t, active)
	require.True(t, exists)

	req.NoError(store.UpdateLedgerStatus(ledgerID, msgs.Status_INACTIVE))
	req.NoErrorf(store.UpdateLedgerStatus(ledgerID, msgs.Status_INACTIVE), "should be no error if updating to same status")

	active, exists, err = store.LedgerIDActive(ledgerID)
	require.NoError(t, err)
	require.False(t, active)
	require.True(t, exists)

	store.Close()
	require.EqualError(t, env.TestStore.UpdateLedgerStatus(ledgerID, msgs.Status_INACTIVE), "ID store is closed")

	_, err = env.TestStore.GetFormat()
	require.EqualError(t, err, "ID store is closed")

	t.Run("ReadDoc error", func(t *testing.T) {
		errExpected := fmt.Errorf("injected ReadDoc error")
		env := NewTestStoreEnv(t, "", couchDBConfig)
		env.TestStore.db = mockCouchDB{readDocErr: errExpected}
		require.EqualError(t, env.TestStore.UpdateLedgerStatus(ledgerID, msgs.Status_INACTIVE), errExpected.Error())
	})

	t.Run("SaveDoc error", func(t *testing.T) {
		errExpected := fmt.Errorf("injected SaveDoc error")
		v, err := json.Marshal(make(map[string]interface{}))
		require.NoError(t, err)
		env := NewTestStoreEnv(t, "", couchDBConfig)
		env.TestStore.db = mockCouchDB{saveDocErr: errExpected, readDocValue: &couchdb.CouchDoc{JSONValue: v}}
		require.EqualError(t, env.TestStore.UpdateLedgerStatus(ledgerID, msgs.Status_INACTIVE), errExpected.Error())
	})

	t.Run("Unmarshal error", func(t *testing.T) {
		env := NewTestStoreEnv(t, "", couchDBConfig)
		env.TestStore.db = mockCouchDB{readDocValue: &couchdb.CouchDoc{JSONValue: []byte("{")}}
		err := env.TestStore.UpdateLedgerStatus(ledgerID, msgs.Status_INACTIVE)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unable to unmarshal JSON value")
	})
}

func TestStore_UpgradeFormat(t *testing.T) {
	req := require.New(t)

	// Ensure that we have a fresh database
	destroy()
	_, _, destroy = xtestutil.SetupExtTestEnv()

	restore := isEmpty
	isEmpty = func(couchInstance *couchdb.CouchInstance) (bool, error) { return false, nil }
	defer func() { isEmpty = restore }()

	ledgerID := "testupgradeformat"
	env := NewTestStoreEnv(t, ledgerID, couchDBConfig)
	store := env.TestStore

	format, err := store.GetFormat()
	req.NoError(err)
	req.Equal(dataformat.PreviousFormat, string(format))

	block := &common.Block{}
	block.Data = &common.BlockData{Data: [][]byte{[]byte("testblock")}}

	require.NoError(t, store.CreateLedgerID(ledgerID, block))

	require.NoError(t, store.UpgradeFormat())
	format, err = store.GetFormat()
	req.NoError(err)
	req.Equal(dataformat.CurrentFormat, string(format))

	require.NoErrorf(t, store.UpgradeFormat(), "should be able to call UpgradeFormat on a DB that has already been upgraded")

	store.Close()
	require.EqualError(t, store.UpgradeFormat(), "ID store is closed")
}

func TestStore_GetFormat(t *testing.T) {
	t.Run("current format -> success", func(t *testing.T) {
		env := NewTestStoreEnv(t, "testledger", couchDBConfig)
		metadata := map[string]interface{}{
			formatField: dataformat.CurrentFormat,
		}
		metadataBytes, err := jsonValue(metadata).toBytes()
		require.NoError(t, err)

		env.TestStore.db = mockCouchDB{readDocValue: &couchdb.CouchDoc{JSONValue: metadataBytes}}
		format, err := env.TestStore.GetFormat()
		require.NoError(t, err)
		require.Equal(t, dataformat.CurrentFormat, string(format))
	})

	t.Run("previous format -> success", func(t *testing.T) {
		env := NewTestStoreEnv(t, "testledger", couchDBConfig)
		metadata := map[string]interface{}{}
		metadataBytes, err := jsonValue(metadata).toBytes()
		require.NoError(t, err)

		env.TestStore.db = mockCouchDB{readDocValue: &couchdb.CouchDoc{JSONValue: metadataBytes}}
		format, err := env.TestStore.GetFormat()
		require.NoError(t, err)
		require.Equal(t, dataformat.PreviousFormat, string(format))
	})

	t.Run("getMetadata error", func(t *testing.T) {
		errExpected := fmt.Errorf("injected getMetadata error")
		env := NewTestStoreEnv(t, "testledger", couchDBConfig)
		env.TestStore.db = mockCouchDB{readDocErr: errExpected}
		_, err := env.TestStore.GetFormat()
		require.Error(t, err)
		require.Contains(t, err.Error(), "retrieval of metadata from CouchDB inventory failed")
	})

	t.Run("closed error", func(t *testing.T) {
		env := NewTestStoreEnv(t, "testledger", couchDBConfig)
		env.TestStore.Close()
		_, err := env.TestStore.GetFormat()
		require.EqualError(t, err, "ID store is closed")
	})
}

func TestStore_GetUnderConstructionID(t *testing.T) {
	require.Equal(t, "", getUnderConstructionID(nil))
	require.Equal(t, "", getUnderConstructionID(jsonValue{}))
	require.Equal(t, "ledger1", getUnderConstructionID(jsonValue{underConstructionLedgerKey: "ledger1"}))
}

func TestStore_GetGenesysBlock(t *testing.T) {
	require.Nil(t, getGenesysBlock(nil))

	attachementBytes := []byte("block")
	require.Equal(t,
		attachementBytes,
		getGenesysBlock([]*couchdb.AttachmentInfo{{
			Name:            blockAttachmentName,
			AttachmentBytes: attachementBytes,
		}}),
	)
}

type mockCouchDB struct {
	existsWithRetryValue               bool
	existsWithRetryErr                 error
	indexDesignDocExistsWithRetryValue bool
	indexDesignDocExistsWithRetryErr   error
	createNewIndexWithRetryErr         error
	saveDocValue                       string
	saveDocErr                         error
	readDocValue                       *couchdb.CouchDoc
	readDocErr                         error
	batchUpdateDocumentsValue          []*couchdb.BatchUpdateResponse
	batchUpdateDocumentsErr            error
	queryDocumentsValue                []*couchdb.QueryResult
	queryDocumentsErr                  error
	empty                              bool
	emptyErr                           error
}

func (m mockCouchDB) ExistsWithRetry() (bool, error) {
	return m.existsWithRetryValue, m.existsWithRetryErr
}
func (m mockCouchDB) IndexDesignDocExistsWithRetry(designDocs ...string) (bool, error) {
	return m.indexDesignDocExistsWithRetryValue, m.indexDesignDocExistsWithRetryErr
}

func (m mockCouchDB) CreateNewIndexWithRetry(indexdefinition string, designDoc string) error {
	return m.createNewIndexWithRetryErr
}

func (m mockCouchDB) SaveDoc(id string, rev string, couchDoc *couchdb.CouchDoc) (string, error) {
	return m.saveDocValue, m.saveDocErr
}

func (m mockCouchDB) ReadDoc(id string) (*couchdb.CouchDoc, string, error) {
	return m.readDocValue, "", m.readDocErr
}

func (m mockCouchDB) BatchUpdateDocuments(documents []*couchdb.CouchDoc) ([]*couchdb.BatchUpdateResponse, error) {
	return m.batchUpdateDocumentsValue, m.batchUpdateDocumentsErr
}

func (m mockCouchDB) QueryDocuments(query string) ([]*couchdb.QueryResult, string, error) {
	return m.queryDocumentsValue, "", m.queryDocumentsErr
}

func (m mockCouchDB) IsEmpty() (bool, error) {
	return m.empty, m.emptyErr
}
