/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	"fmt"
	"time"
)

// Key is a transient data key
type Key struct {
	Namespace  string
	Collection string
	Key        string
}

func (k Key) String() string {
	return fmt.Sprintf("%s:%s:%s", k.Namespace, k.Collection, k.Key)
}

// Value is a transient data value
type Value struct {
	Value      []byte
	TxID       string
	ExpiryTime time.Time
}
