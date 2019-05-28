/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dcas

import (
	"crypto"
	"encoding/base64"

	"github.com/btcsuite/btcutil/base58"
)

// GetCASKey returns the content-addressable key for the given content.
func GetCASKey(content []byte) string {
	address := calculateAddress(content)

	// Address above is as per CAS spec(sha256 hash + base64 URL encoding),
	// however since fabric/couchdb doesn't support keys that start with _
	// we have to do additional transformation
	return base58.Encode(address)
}

func calculateAddress(content []byte) []byte {
	hash := getHash(content)
	buf := make([]byte, base64.URLEncoding.EncodedLen(len(hash)))
	base64.URLEncoding.Encode(buf, hash)

	return buf
}

// getHash will compute the hash for the supplied bytes using SHA256
func getHash(bytes []byte) []byte {
	h := crypto.SHA256.New()
	// added no lint directive because there's no error from source code
	// error cannot be produced, checked google source
	h.Write(bytes) //nolint
	return h.Sum(nil)
}
