/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package leveldbstore

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strings"
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/ledger/util/leveldbhelper"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/storeprovider/store/api"
)

var logger = flogging.MustGetLogger("ext_offledger")

var compositeKeySep = "!"

type store struct {
	db     *leveldbhelper.DBHandle
	dbName string
}

// newDBStore constructs an instance of db store
func newDBStore(db *leveldbhelper.DBHandle, dbName string) *store {
	return &store{db, dbName}
}

// AddKey add cache key to db
func (s *store) Put(keyVal ...*api.KeyValue) error {
	batch := leveldbhelper.NewUpdateBatch()
	for _, kv := range keyVal {
		err := s.addToBatch(batch, kv)
		if err != nil {
			return err
		}
	}
	return s.db.WriteBatch(batch, true)
}

func (s *store) addToBatch(batch *leveldbhelper.UpdateBatch, kv *api.KeyValue) error {
	if kv.Value == nil {
		logger.Debugf("Deleting key [%s]", kv.Key)
		batch.Delete(encodeKey(kv.Key, time.Time{}))
		return nil
	}

	logger.Debugf("Adding key [%s]", kv.Key)
	encodedVal, err := encodeVal(kv.Value)
	if err != nil {
		return errors.WithMessagef(err, "failed to encode value for key [%s]", kv.Value)
	}
	batch.Put(encodeKey(kv.Key, time.Time{}), encodedVal)

	if !kv.ExpiryTime.IsZero() {
		// put same previous key with prefix expiryTime so the clean up can remove all expired keys
		err = s.db.Put(encodeKey(kv.Key, kv.ExpiryTime), []byte(""), true)
		if err != nil {
			return errors.Wrapf(err, "failed to save key [%s] in db", kv.Key)
		}
	}
	return nil
}

// GetKey get cache key from db
func (s *store) Get(key string) (*api.Value, error) {
	logger.Debugf("load key [%s] from db", key)
	value, err := s.db.Get(encodeKey(key, time.Time{}))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load key [%s] from db", key)
	}
	if value != nil {
		val, err := decodeVal(value)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to decode value [%s] for key [%s]", value, key)
		}
		return val, nil
	}
	return nil, nil
}

// GetMultiple retrieves values for multiple keys at once
func (s *store) GetMultiple(keys ...string) ([]*api.Value, error) {
	values := make([]*api.Value, len(keys))
	for i, k := range keys {
		v, err := s.Get(k)
		if err != nil {
			return nil, err
		}
		values[i] = v
	}
	return values, nil
}

// Query is not implemented
func (s *store) Query(query string) ([]*api.KeyValue, error) {
	panic("not implemented")
}

// DeleteExpiredKeys delete expired keys from db
func (s *store) DeleteExpiredKeys() error {
	dbBatch := leveldbhelper.NewUpdateBatch()
	itr := s.db.GetIterator(nil, []byte(fmt.Sprintf("%d%s", time.Now().UTC().UnixNano(), compositeKeySep)))
	for itr.Next() {
		key := string(itr.Key())
		dbBatch.Delete([]byte(key))
		dbBatch.Delete([]byte(key[strings.Index(key, compositeKeySep)+1:]))
	}
	if dbBatch.Len() > 0 {
		err := s.db.WriteBatch(dbBatch, true)
		if err != nil {
			return errors.Errorf("failed to delete keys %s in db %s", dbBatch.KVs, err.Error())
		}
		logger.Debugf("delete expired keys %s from db", dbBatch.KVs)
	}

	return nil
}

// Close db
func (s *store) Close() {
	// Nothing to do
}

func encodeKey(key string, expiryTime time.Time) []byte {
	var compositeKey []byte
	if !expiryTime.IsZero() {
		compositeKey = append(compositeKey, []byte(fmt.Sprintf("%d", expiryTime.UnixNano()))...)
		compositeKey = append(compositeKey, compositeKeySep...)
	}
	compositeKey = append(compositeKey, []byte(key)...)
	return compositeKey
}

func decodeVal(b []byte) (*api.Value, error) {
	decoder := gob.NewDecoder(bytes.NewBuffer(b))
	var v *api.Value
	if err := decoder.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

func encodeVal(v *api.Value) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	encoder := gob.NewEncoder(buf)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
