/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dissemination

import (
	"testing"

	"github.com/trustbloc/fabric-peer-ext/pkg/mocks"
)

func TestLocalMSPProvider(t *testing.T) {
	ip := &mocks.IdentifierProvider{}

	LocalMSPProvider.Initialize(ip)
}
