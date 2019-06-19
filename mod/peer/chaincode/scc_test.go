/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chaincode

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

func TestCreateSCC(t *testing.T) {
	require.Empty(t, CreateSCC(mocks.NewQueryExecutorProvider()))
}
