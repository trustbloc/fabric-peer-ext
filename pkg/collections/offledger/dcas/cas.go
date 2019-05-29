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
	hash := getHash(content)
	buf := make([]byte, base64.URLEncoding.EncodedLen(len(hash)))
	base64.URLEncoding.Encode(buf, hash)
	return string(buf)
}

// GetFabricCASKey returns the content-addressable key for the given content,
// encoded in base58 so that it may be used as a key in Fabric.
func GetFabricCASKey(content []byte) string {
	return Base58Encode(GetCASKey(content))
}

// getHash will compute the hash for the supplied bytes using SHA256
func getHash(bytes []byte) []byte {
	h := crypto.SHA256.New()
	// added no lint directive because there's no error from source code
	// error cannot be produced, checked google source
	h.Write(bytes) //nolint
	return h.Sum(nil)
}

// Base58Encode encodes the given string in base 58
func Base58Encode(s string) string {
	return base58.Encode([]byte(s))
}
