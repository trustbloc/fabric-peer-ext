/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dbname

import (
	extdbname "github.com/trustbloc/fabric-peer-ext/pkg/common/dbname"
)

// Resolve resolves the database name according to the peer's configuration
func Resolve(name string) string {
	return extdbname.ResolverInstance.Resolve(name)
}

// IsRelevant returns true if the given database is relevant to this peer. If the database is shared
// by multiple peers then it may not be relevant to this peer.
func IsRelevant(dbName string) bool {
	return extdbname.ResolverInstance.IsRelevant(dbName)
}
