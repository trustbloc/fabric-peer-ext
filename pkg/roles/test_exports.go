// +build testing

/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package roles

//SetRoles used for unit test
func SetRoles(rolesValue map[Role]struct{}) {
	if rolesValue == nil {
		roles = initRoles()
	} else {
		roles = rolesValue
	}
}

// SetRole sets one or more roles and returns a cancel function which, when invoked,
// resets the roles to the previous state.
func SetRole(role ...Role) (reset func()) {
	existingRoles := roles

	rolesValue := make(map[Role]struct{})

	for _, r := range role {
		rolesValue[r] = struct{}{}
	}

	SetRoles(rolesValue)

	return func() { SetRoles(existingRoles) }
}
