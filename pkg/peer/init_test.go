/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package peer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitialize(t *testing.T) {
	require.NotPanics(t, Initialize)
}
