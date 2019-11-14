/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package retriever

import (
	"testing"

	"github.com/stretchr/testify/require"
	olmocks "github.com/trustbloc/fabric-peer-ext/pkg/collections/offledger/mocks"
	tdatamocks "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/mocks"
)

func TestNewProvider(t *testing.T) {
	p := NewProvider().Initialize(&tdatamocks.TransientDataProvider{}, &olmocks.Provider{})
	require.NotNil(t, p)
	require.NotNil(t, p.RetrieverForChannel("testchannel"))
}
