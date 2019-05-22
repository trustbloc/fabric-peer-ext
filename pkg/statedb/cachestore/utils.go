/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package cachestore

// ConstructCompositeKey constructs a string composite key
func ConstructCompositeKey(ns string, key string) string {
	return ns + key
}
