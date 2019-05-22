/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"reflect"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
)

// Values contains a slice of values
type Values []interface{}

// IsEmpty returns true if all of the values are nil
func (v Values) IsEmpty() bool {
	for _, value := range v {
		if !IsNil(value) {
			return false
		}
	}
	return true
}

// AllSet returns true if all of the values are not nil
func (v Values) AllSet() bool {
	for _, value := range v {
		if IsNil(value) {
			return false
		}
	}
	return true
}

// Merge merges this set of values with the given set
// and returns the new set
func (v Values) Merge(other Values) Values {
	var max int
	if len(other) < len(v) {
		max = len(v)
	} else {
		max = len(other)
	}

	retVal := make(Values, max)
	copy(retVal, v)

	for i, o := range other {
		if IsNil(retVal[i]) {
			retVal[i] = o
		}
	}

	return retVal
}

// IsNil returns true if the given value is nil
func IsNil(p interface{}) bool {
	if p == nil {
		return true
	}

	v := reflect.ValueOf(p)

	switch {
	case v.Kind() == reflect.Ptr:
		return v.IsNil()
	case v.Kind() == reflect.Array || v.Kind() == reflect.Slice:
		return v.Len() == 0
	default:
		return false
	}
}

// ToTimestamp converts the Time into Timestamp
func ToTimestamp(t time.Time) *timestamp.Timestamp {
	now := time.Now().UTC()
	return &(timestamp.Timestamp{Seconds: now.Unix()})
}

// FromTimestamp converts the Timestamp into Time
func FromTimestamp(ts *timestamp.Timestamp) time.Time {
	return time.Unix(ts.Seconds, 0)
}
