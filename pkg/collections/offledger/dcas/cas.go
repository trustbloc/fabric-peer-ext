/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dcas

import (
	"crypto"
	"encoding/base64"
	"encoding/json"

	"github.com/btcsuite/btcutil/base58"
)

// GetCASKeyAndValue first normalizes the content (i.e. if the content is a JSON doc then the fields
// are marshaled in a deterministic order) and returns the content-addressable key
// (encoded in base64) along with the normalized value.
func GetCASKeyAndValue(content []byte) (string, []byte, error) {
	bytes, err := getNormalizedContent(content)
	if err != nil {
		return "", nil, err
	}
	return string(getCASKey(bytes)), bytes, nil
}

// GetCASKeyAndValueBase58 first normalizes the content (i.e. if the content is a JSON doc then the fields
// are marshaled in a deterministic order) and returns the content-addressable key
// (first encoded in base64 and then in base58) along with the normalized value.
func GetCASKeyAndValueBase58(content []byte) (string, []byte, error) {
	bytes, err := getNormalizedContent(content)
	if err != nil {
		return "", nil, err
	}
	return base58.Encode(getCASKey(bytes)), bytes, nil
}

// getNormalizedContent ensures that, if the content is a JSON doc, then the fields are marshaled in a deterministic order
// so that the hash of the content is also deterministic.
func getNormalizedContent(content []byte) ([]byte, error) {
	m, err := unmarshalJSONMap(content)
	if err != nil {
		// This is not a JSON document
		return content, nil
	}

	// This is a JSON doc. Re-marshal it in order to ensure that the JSON fields are marshaled in a deterministic order.
	bytes, err := marshalJSONMap(m)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

// getHash will compute the hash for the supplied bytes using SHA256
func getHash(bytes []byte) []byte {
	h := crypto.SHA256.New()
	// added no lint directive because there's no error from source code
	// error cannot be produced, checked google source
	h.Write(bytes) //nolint
	return h.Sum(nil)
}

func getCASKey(content []byte) []byte {
	hash := getHash(content)
	buf := make([]byte, base64.URLEncoding.EncodedLen(len(hash)))
	base64.URLEncoding.Encode(buf, hash)
	return buf
}

// marshalJSONMap marshals a JSON map. This variable may be overridden by unit tests.
var marshalJSONMap = func(m map[string]interface{}) ([]byte, error) {
	return json.Marshal(&m)
}

// unmarshalJSONMap unmarshals a JSON map from the given bytes. This variable may be overridden by unit tests.
var unmarshalJSONMap = func(bytes []byte) (map[string]interface{}, error) {
	m := make(map[string]interface{})
	err := json.Unmarshal(bytes, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}
