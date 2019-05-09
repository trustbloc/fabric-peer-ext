/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package roles

import "github.com/trustbloc/fabric-peer-ext/pkg/roles"

// IsCommitter returns true if the peer is a committer, otherwise the peer does not commit to the DB
func IsCommitter() bool {
	return roles.IsCommitter()
}

// IsEndorser returns true if the peer is an endorser
func IsEndorser() bool {
	return roles.IsEndorser()
}

// IsValidator returns true if the peer is a validator
func IsValidator() bool {
	return roles.IsValidator()
}

// RolesAsString returns the roles for the peer
// nolint - this is an exported function (Renaming function name will break in other projects)
func RolesAsString() []string {
	return roles.AsString()
}
