/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import "github.com/spf13/viper"

const confRoles = "ledger.roles"

// GetRoles returns the roles of the peer. Empty return value indicates that the peer has all roles.
func GetRoles() string {
	return viper.GetString(confRoles)
}
