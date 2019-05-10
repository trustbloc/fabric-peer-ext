// +build testing

/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package roles

//SetRoles used for unit test
func SetRoles(rolesValue map[Role]struct{}) {
	roles = rolesValue
}
