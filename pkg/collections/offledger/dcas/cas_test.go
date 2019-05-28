/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dcas

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCASKey(t *testing.T) {
	k := GetCASKey([]byte("value1"))
	assert.NotEmpty(t, k)
}
