// +build testing

/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package storeprovider

import (
	tdapi "github.com/trustbloc/fabric-peer-ext/pkg/collections/transientdata/api"
)

// SetNewTransientDataProvider sets the transient data provider for unit tests
func SetNewTransientDataProvider(provider func() tdapi.StoreProvider) {
	newTransientDataProvider = provider
}
