// +build testing

/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dcas

// SetJSONMarshaller sets the JSON map marshaller for unit tests.
// Returns a function that resets the marshaller to the previous value.
func SetJSONMarshaller(marshaller func(m map[string]interface{}) ([]byte, error)) func() {
	prevMarshaller := marshalJSONMap
	marshalJSONMap = marshaller
	return func() {
		marshalJSONMap = prevMarshaller
	}
}
