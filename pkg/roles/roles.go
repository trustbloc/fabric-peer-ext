/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package roles

import (
	"strings"
	"sync"

	"github.com/trustbloc/fabric-peer-ext/pkg/config"
)

const (
	// CommitterRole indicates that the peer commits data to the ledger
	CommitterRole Role = "committer"
	// EndorserRole indicates that the peer endorses transaction proposals
	EndorserRole Role = "endorser"
	// ValidatorRole indicates that the peer validates the block
	ValidatorRole Role = "validator"
)

// Role is the role of the peer
type Role = string

// Roles is a set of peer roles
type Roles []Role

// FromStrings creates Roles from the given slice of strings
func FromStrings(r ...string) Roles {
	rls := make(Roles, len(r))
	for i, s := range r {
		rls[i] = strings.ToLower(s)
	}
	return rls
}

// Contains return true if the given role is included in the set
func (r Roles) Contains(role Role) bool {
	if len(r) == 0 {
		// Return true by default in order to be backward compatible
		return true
	}
	for _, r := range r {
		if r == role {
			return true
		}
	}
	return false
}

var initOnce sync.Once
var roles map[Role]struct{}

// HasRole returns true if the peer has the given role
func HasRole(role Role) bool {
	_, ok := getRoles()[role]
	return ok
}

// IsCommitter returns true if the peer is a committer, otherwise the peer does not commit to the DB
func IsCommitter() bool {
	return HasRole(CommitterRole)
}

// IsEndorser returns true if the peer is an endorser
func IsEndorser() bool {
	return HasRole(EndorserRole)
}

// IsValidator returns true if the peer is a validator
func IsValidator() bool {
	return HasRole(ValidatorRole)
}

// IsClustered returns true if we're running in clustered mode
func IsClustered() bool {
	return (IsCommitter() && !IsEndorser()) || (IsEndorser() && !IsCommitter())
}

// GetRoles returns the roles for the peer
func GetRoles() []Role {
	var ret []Role
	for role := range getRoles() {
		ret = append(ret, role)
	}
	return ret
}

// AsString returns the roles for the peer
func AsString() []string {
	var ret []string
	for role := range getRoles() {
		ret = append(ret, role)
	}
	return ret
}

func getRoles() map[Role]struct{} {
	initOnce.Do(func() {
		roles = initRoles()
	})
	return roles
}

func initRoles() map[Role]struct{} {
	rolesMap := make(map[Role]struct{})
	exists := struct{}{}
	strRoles := config.GetRoles()

	if strRoles == "" {
		// The peer has all roles by default
		rolesMap[CommitterRole] = exists
		rolesMap[EndorserRole] = exists
		rolesMap[ValidatorRole] = exists
		return rolesMap
	}

	for _, r := range strings.Split(strRoles, ",") {
		r = strings.ToLower(strings.TrimSpace(r))
		rolesMap[r] = exists
	}

	return rolesMap
}
