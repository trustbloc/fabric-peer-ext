/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package resource

// Clear all resources. (Used by unit tests.)
func (b *Manager) Clear() {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.resources = nil
}
