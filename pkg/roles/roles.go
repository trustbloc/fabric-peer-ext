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
type Role string

// Roles is a set of peer roles
type Roles []Role

// New creates Roles from the given slice of roles
func New(r ...Role) Roles {
	return Roles(r)
}

// FromStrings creates Roles from the given slice of strings
func FromStrings(r ...string) Roles {
	rls := make(Roles, len(r))
	for i, s := range r {
		rls[i] = Role(s)
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
	initOnce.Do(func() {
		roles = getRoles()
	})

	if len(roles) == 0 {
		// No roles were explicitly set, therefore the peer is assumed to have all roles.
		return true
	}

	_, ok := roles[role]
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

// GetRoles returns the roles for the peer
func GetRoles() []Role {
	var ret []Role
	for role := range roles {
		ret = append(ret, role)
	}
	return ret
}

// AsString returns the roles for the peer
func AsString() []string {
	var ret []string
	for role := range roles {
		ret = append(ret, string(role))
	}
	return ret
}

func getRoles() map[Role]struct{} {
	exists := struct{}{}
	strRoles := config.GetRoles()
	if strRoles == "" {
		// The peer has all roles by default
		return map[Role]struct{}{}
	}
	rolesMap := make(map[Role]struct{})
	for _, r := range strings.Split(strRoles, ",") {
		r = strings.ToLower(strings.TrimSpace(r))
		rolesMap[Role(r)] = exists
	}
	return rolesMap
}
