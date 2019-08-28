/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
/*
Notice: This file has been modified for fabric-peer-ext usage.
*/

package compositekey

import (
	"unicode/utf8"
)

const (
	minUnicodeRuneValue   = 0            //U+0000
	maxUnicodeRuneValue   = utf8.MaxRune //U+10FFFF - maximum (and unallocated) code point
	compositeKeyNamespace = "\x00"
)

// Create combines the given `attributes` to form a composite
// key. The objectType and attributes are expected to have only valid utf8
// strings and should not contain U+0000 (nil byte) and U+10FFFF
// (biggest and unallocated code point).
// The resulting composite key can be used as the key in PutState().
func Create(objectType string, attributes []string) string {
	ck := compositeKeyNamespace + objectType + string(minUnicodeRuneValue)
	for _, att := range attributes {
		ck += att + string(minUnicodeRuneValue)
	}
	return ck
}

// Split splits the specified key into attributes on which the
// composite key was formed. Composite keys found during range queries
// or partial composite key queries can therefore be split into their
// composite parts.
func Split(compositeKey string) (string, []string) {
	componentIndex := 1
	components := []string{}
	for i := 1; i < len(compositeKey); i++ {
		if compositeKey[i] == minUnicodeRuneValue {
			components = append(components, compositeKey[componentIndex:i])
			componentIndex = i + 1
		}
	}
	return components[0], components[1:]
}

// CreateRangeKeysForPartialCompositeKey returns a start and an end key for the given partial composite key
func CreateRangeKeysForPartialCompositeKey(objectType string, attributes []string) (string, string) {
	startKey := Create(objectType, attributes)
	return startKey, startKey + string(maxUnicodeRuneValue)
}
