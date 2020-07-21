/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvledger

import "path/filepath"

// StateDBPath returns the absolute path of state level DB
func StateDBPath(rootFSPath string) string {
	return filepath.Join(rootFSPath, "stateLeveldb")
}

// HistoryDBPath returns the absolute path of history DB
func HistoryDBPath(rootFSPath string) string {
	return filepath.Join(rootFSPath, "historyLeveldb")
}

// ConfigHistoryDBPath returns the absolute path of configHistory DB
func ConfigHistoryDBPath(rootFSPath string) string {
	return filepath.Join(rootFSPath, "configHistory")
}

// BookkeeperDBPath return the absolute path of bookkeeper DB
func BookkeeperDBPath(rootFSPath string) string {
	return filepath.Join(rootFSPath, "bookkeeper")
}

func fileLockPath(rootFSPath string) string {
	return filepath.Join(rootFSPath, "fileLock")
}
