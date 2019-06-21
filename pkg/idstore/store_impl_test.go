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

	"github.com/hyperledger/fabric/core/ledger"

	"github.com/trustbloc/fabric-peer-ext/mod/peer/testutil"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/ledger/util/couchdb"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/roles"
	xtestutil "github.com/trustbloc/fabric-peer-ext/pkg/testutil"
)

var couchDBConfig *couchdb.Config

func TestMain(m *testing.M) {
	//setup extension test environment
	_, _, destroy := xtestutil.SetupExtTestEnv()

	// Create CouchDB definition from config parameters
	couchDBConfig = xtestutil.TestLedgerConf().StateDB.CouchDB

	code := m.Run()
	destroy()
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
	t.Run("test error from ExistsWithRetry", func(t *testing.T) {
		_, err := newCommitterStore(mockCouchDB{createNewIndexWithRetryErr: fmt.Errorf("ExistsWithRetry error")})
		require.Error(t, err)
		require.Contains(t, err.Error(), "create couchdb index failed")

	})
}

func TestCreateCouchInstance(t *testing.T) {
	t.Run("test error from CreateCouchInstance", func(t *testing.T) {
		_, err := createCouchInstance(&ledger.Config{StateDB: &ledger.StateDB{CouchDB: &couchdb.Config{}}})
		require.Error(t, err)
		require.Contains(t, err.Error(), "obtaining CouchDB instance failed")

	})
}

func TestNewStore(t *testing.T) {
	t.Run("test error from ExistsWithRetry", func(t *testing.T) {
		_, err := newStore(mockCouchDB{existsWithRetryErr: fmt.Errorf("ExistsWithRetry error")}, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "ExistsWithRetry error")
	})

	t.Run("test db not exists", func(t *testing.T) {
		_, err := newStore(mockCouchDB{existsWithRetryValue: false}, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "DB not found")
	})

	t.Run("test error from IndexDesignDocExistsWithRetry", func(t *testing.T) {
		_, err := newStore(mockCouchDB{existsWithRetryValue: true, indexDesignDocExistsWithRetryErr: fmt.Errorf("IndexDesignDocExistsWithRetry error")}, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "IndexDesignDocExistsWithRetry error")
	})

	t.Run("test index not exists", func(t *testing.T) {
		_, err := newStore(mockCouchDB{existsWithRetryValue: true, indexDesignDocExistsWithRetryValue: false}, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "DB index not found")
	})
}

func TestUnderConstructionFlag(t *testing.T) {

	t.Run("test error from SetUnderConstructionFlag SaveDoc", func(t *testing.T) {
		ledgerID := "testunderconstructiongflag"
		env := NewTestStoreEnv(t, ledgerID, couchDBConfig)
		store := env.TestStore
		s := store.(*Store)
		s.db = mockCouchDB{saveDocErr: fmt.Errorf("SaveDoc error")}
		err := store.SetUnderConstructionFlag(ledgerID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "update of metadata in CouchDB failed")

	})

	t.Run("test error from UnsetUnderConstructionFlag SaveDoc", func(t *testing.T) {
		ledgerID := "testunderconstructiongflag"
		env := NewTestStoreEnv(t, ledgerID, couchDBConfig)
		store := env.TestStore
		s := store.(*Store)
		s.db = mockCouchDB{saveDocErr: fmt.Errorf("SaveDoc error")}
		err := store.UnsetUnderConstructionFlag()
		require.Error(t, err)
		require.Contains(t, err.Error(), "update of metadata in CouchDB failed")

	})

	t.Run("test error from SetUnderConstructionFlag SaveDoc", func(t *testing.T) {
		ledgerID := "testunderconstructiongflag"
		env := NewTestStoreEnv(t, ledgerID, couchDBConfig)
		store := env.TestStore
		s := store.(*Store)
		s.db = mockCouchDB{saveDocErr: fmt.Errorf("SaveDoc error")}
		err := store.SetUnderConstructionFlag(ledgerID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "update of metadata in CouchDB failed")

	})

	t.Run("test error from GetUnderConstructionFlag ReadDoc", func(t *testing.T) {
		ledgerID := "testunderconstructiongflag"
		env := NewTestStoreEnv(t, ledgerID, couchDBConfig)
		store := env.TestStore
		s := store.(*Store)
		s.db = mockCouchDB{readDocErr: fmt.Errorf("SaveDoc error")}
		_, err := store.GetUnderConstructionFlag()
		require.Error(t, err)
		require.Contains(t, err.Error(), "retrieval of metadata from CouchDB inventory failed")

	})

	t.Run("test doc is empty", func(t *testing.T) {
		ledgerID := "testunderconstructiongflag"
		env := NewTestStoreEnv(t, ledgerID, couchDBConfig)
		store := env.TestStore
		s := store.(*Store)
		s.db = mockCouchDB{}
		value, err := store.GetUnderConstructionFlag()
		require.NoError(t, err)
		require.Empty(t, value)
	})

	t.Run("test metadata is invalid", func(t *testing.T) {
		ledgerID := "testunderconstructiongflag"
		env := NewTestStoreEnv(t, ledgerID, couchDBConfig)
		store := env.TestStore
		s := store.(*Store)
		s.db = mockCouchDB{readDocValue: &couchdb.CouchDoc{}}
		_, err := store.GetUnderConstructionFlag()
		require.Error(t, err)
		require.Contains(t, err.Error(), "metadata in CouchDB inventory is invalid")

	})

	t.Run("test metadata under construction key is invalid", func(t *testing.T) {
		ledgerID := "testunderconstructiongflag"
		env := NewTestStoreEnv(t, ledgerID, couchDBConfig)
		store := env.TestStore
		s := store.(*Store)
		v, err := json.Marshal(make(map[string]interface{}))
		require.NoError(t, err)
		s.db = mockCouchDB{readDocValue: &couchdb.CouchDoc{JSONValue: v}}
		_, err = store.GetUnderConstructionFlag()
		require.Error(t, err)
		require.Contains(t, err.Error(), "metadata under construction key in CouchDB inventory is invalid")

	})

	t.Run("test success", func(t *testing.T) {
		ledgerID := "testunderconstructiongflag"
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

	t.Run("test error from CreateLedgerID LedgerIDExists", func(t *testing.T) {
		env := NewTestStoreEnv(t, "", couchDBConfig)
		store := env.TestStore
		s := store.(*Store)
		s.db = mockCouchDB{readDocErr: fmt.Errorf("ReadDoc error")}
		err := s.CreateLedgerID("", nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "ReadDoc error")

	})

	t.Run("test error from CreateLedgerID BatchUpdateDocuments", func(t *testing.T) {
		env := NewTestStoreEnv(t, "", couchDBConfig)
		store := env.TestStore
		s := store.(*Store)
		s.db = mockCouchDB{batchUpdateDocumentsErr: fmt.Errorf("BatchUpdateDocuments error")}
		block := &common.Block{}
		block.Data = &common.BlockData{Data: [][]byte{[]byte("testblock")}}
		err := s.CreateLedgerID("", block)
		require.Error(t, err)
		require.Contains(t, err.Error(), "creation of ledger failed ")

	})

	t.Run("test error from CreateLedgerID BatchUpdateDocuments", func(t *testing.T) {
		env := NewTestStoreEnv(t, "", couchDBConfig)
		store := env.TestStore
		s := store.(*Store)
		s.db = mockCouchDB{batchUpdateDocumentsErr: fmt.Errorf("BatchUpdateDocuments error")}
		block := &common.Block{}
		block.Data = &common.BlockData{Data: [][]byte{[]byte("testblock")}}
		err := s.CreateLedgerID("", block)
		require.Error(t, err)
		require.Contains(t, err.Error(), "creation of ledger failed ")

	})

	t.Run("test error from GetLedgeIDValue ReadDoc", func(t *testing.T) {
		env := NewTestStoreEnv(t, "", couchDBConfig)
		store := env.TestStore
		s := store.(*Store)
		s.db = mockCouchDB{readDocErr: fmt.Errorf("ReadDoc error")}
		_, err := s.GetLedgeIDValue("")
		require.Error(t, err)
		require.Contains(t, err.Error(), "ReadDoc error")

	})

	t.Run("test GetLedgeIDValue ReadDoc return empty doc", func(t *testing.T) {
		env := NewTestStoreEnv(t, "", couchDBConfig)
		store := env.TestStore
		s := store.(*Store)
		s.db = mockCouchDB{readDocValue: &couchdb.CouchDoc{}}
		v, err := s.GetLedgeIDValue("")
		require.NoError(t, err)
		require.Empty(t, v)

	})

	t.Run("test error from GetAllLedgerIds queryInventory", func(t *testing.T) {
		env := NewTestStoreEnv(t, "", couchDBConfig)
		store := env.TestStore
		s := store.(*Store)
		s.db = mockCouchDB{queryDocumentsErr: fmt.Errorf("QueryDocuments error")}
		_, err := s.GetAllLedgerIds()
		require.Error(t, err)
		require.Contains(t, err.Error(), "QueryDocuments error")
	})

	t.Run("test error from GetAllLedgerIds couchValueToJSON", func(t *testing.T) {
		env := NewTestStoreEnv(t, "", couchDBConfig)
		store := env.TestStore
		s := store.(*Store)
		s.db = mockCouchDB{queryDocumentsValue: []*couchdb.QueryResult{{Value: []byte("wrongData")}}}
		_, err := s.GetAllLedgerIds()
		require.Error(t, err)
		require.Contains(t, err.Error(), "couchValueToJSON failed")
	})

	t.Run("test empty doc from GetAllLedgerIds queryInventory", func(t *testing.T) {
		env := NewTestStoreEnv(t, "", couchDBConfig)
		store := env.TestStore
		s := store.(*Store)
		v, err := json.Marshal(make(map[string]interface{}))
		require.NoError(t, err)
		s.db = mockCouchDB{queryDocumentsValue: []*couchdb.QueryResult{{Value: v}}}
		_, err = s.GetAllLedgerIds()
		require.Error(t, err)
		require.Contains(t, err.Error(), "ledger inventory document is invalid")
	})

	t.Run("test wrong doc from GetAllLedgerIds queryInventory", func(t *testing.T) {
		env := NewTestStoreEnv(t, "", couchDBConfig)
		store := env.TestStore
		s := store.(*Store)
		m := make(map[string]interface{})
		m[inventoryNameLedgerIDField] = 1
		v, err := json.Marshal(m)
		require.NoError(t, err)
		s.db = mockCouchDB{queryDocumentsValue: []*couchdb.QueryResult{{Value: v}}}
		_, err = s.GetAllLedgerIds()
		require.Error(t, err)
		require.Contains(t, err.Error(), "ledger inventory document value is invalid")
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
		ledgerIDs, err := store.GetAllLedgerIds()
		req.NoError(err)
		req.Equal(2, len(ledgerIDs))
		req.Contains(ledgerIDs, ledgerID)
		req.Contains(ledgerIDs, ledgerID1)

		// get ledger id value
		ledgerIdValue, err := store.GetLedgeIDValue(ledgerID)
		req.NoError(err)
		gb := &common.Block{}
		req.NoError(proto.Unmarshal(ledgerIdValue, gb))
		req.Equal("testblock", string(gb.Data.Data[0]))

		//check ledger id exist
		exist, err := store.LedgerIDExists(ledgerID)
		req.NoError(err)
		req.Equal(exist, true)
	})

}
func TestClose(t *testing.T) {
	ledgerID := "testclose"
	env := NewTestStoreEnv(t, ledgerID, couchDBConfig)
	store := env.TestStore
	store.Close()
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
