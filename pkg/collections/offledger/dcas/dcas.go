/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dcas

import (
	"strings"

	"github.com/hyperledger/fabric/common/flogging"
	storeapi "github.com/hyperledger/fabric/extensions/collections/api/store"
	"github.com/ipfs/go-datastore"
	dshelp "github.com/ipfs/go-ipfs-ds-help"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("ext_offledger")

// Validator is an off-ledger validator that validates the CAS key against the value
func Validator(_, _, _, key string, value []byte) error {
	return ValidateDatastoreKey(key, value)
}

// Decorator is an off-ledger decorator that ensures the key is the hash of the value.
var Decorator = &decorator{}

type decorator struct {
}

// BeforeSave ensures that the given key is the base58 encoded hash of the value.
func (d *decorator) BeforeSave(key *storeapi.Key, value *storeapi.ExpiringValue) (*storeapi.Key, *storeapi.ExpiringValue, error) {
	if err := ValidateDatastoreKey(key.Key, value.Value); err != nil {
		return nil, nil, err
	}

	return key, value, nil
}

// BeforeLoad returns the key.
func (d *decorator) BeforeLoad(key *storeapi.Key) (*storeapi.Key, error) {
	return key, nil
}

// AfterQuery returns the key/value.
func (d *decorator) AfterQuery(key *storeapi.Key, value *storeapi.ExpiringValue) (*storeapi.Key, *storeapi.ExpiringValue, error) {
	return key, value, nil
}

// ValidateDatastoreKey validates the given data store key, ensuring that it conforms to the
// encoding used by the DCAS store store
func ValidateDatastoreKey(key string, value []byte) error {
	if value == nil {
		return errors.Errorf("attempt to put nil value for key [%s]", key)
	}

	logger.Debugf("Validating provided key [%s]", key)

	// The datastore key may have a prefix (e.g. /blocks/AFKREI...). Validation should start at the last "/".
	k := key
	i := strings.LastIndex(k, "/")
	if i > 0 {
		k = key[i:]
	}

	cID, err := dshelp.DsKeyToCid(datastore.NewKey(k))
	if err != nil {
		return errors.WithMessagef(err, "invalid CAS key [%s]", key)
	}

	prefix := cID.Prefix()

	logger.Debugf("Validating provided key [%s] using CID version [%d], codec [%d] and multi-hash [%d]", key, prefix.Version, prefix.Codec, prefix.MhType)

	expectedCid, err := getCID(value, prefix.Version, prefix.Codec, prefix.MhType)
	if err != nil {
		logger.Debugf("Error creating CID for value using CID version [%d], codec [%d] and multi-hash type [%d]: %s", prefix.Version, prefix.Codec, prefix.MhType, err)

		return errors.WithMessagef(err, "error creating CID using using CID version [%d], codec [%d] and multi-hash type [%d]", prefix.Version, prefix.Codec, prefix.MhType)
	}

	if cID.String() != expectedCid.String() {
		return errors.Errorf("invalid data store key [%s] - CID [%s] for CID version [%d], codec [%d] and multi-hash type [%d] - it should be [%s]", key, cID, prefix.Version, prefix.Codec, prefix.MhType, expectedCid)
	}

	logger.Debugf("Validated CAS key [%s] using CID version [%d], codec [%d] and multi-hash type [%d]. CID: %s", key, prefix.Version, prefix.Codec, prefix.MhType, cID)

	return nil
}
