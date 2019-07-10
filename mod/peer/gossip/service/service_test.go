/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/trustbloc/fabric-peer-ext/pkg/roles"

	"github.com/hyperledger/fabric/protos/gossip"
)

func TestHandleGossipByCommitter(t *testing.T) {

	//make sure roles is committer not endorser
	if roles.IsEndorser() {
		rolesValue := make(map[roles.Role]struct{})
		rolesValue[roles.CommitterRole] = struct{}{}
		roles.SetRoles(rolesValue)
		defer func() { roles.SetRoles(nil) }()
	}

	doneCh := make(chan bool, 1)
	handle := func(msg *gossip.GossipMessage) {
		doneCh <- true
	}
	HandleGossip(handle)(nil)

	select {
	case <-doneCh:
		t.Fatal("handler not supposed to be executed for committer")
	default:
		//do nothing
	}
}

func TestHandleGossipByEndorser(t *testing.T) {

	//make sure roles is endorser not committer
	if roles.IsCommitter() {
		rolesValue := make(map[roles.Role]struct{})
		rolesValue[roles.EndorserRole] = struct{}{}
		roles.SetRoles(rolesValue)
		defer func() { roles.SetRoles(nil) }()
	}

	doneCh := make(chan bool, 1)
	handle := func(msg *gossip.GossipMessage) {
		doneCh <- true
	}
	HandleGossip(handle)(nil)

	select {
	case ok := <-doneCh:
		require.True(t, ok)
	default:
		t.Fatal("handler supposed to be executed for endorser")
	}
}

func TestIsPvtDataReconcilerEnabledByCommitter(t *testing.T) {

	//make sure roles is committer not endorser
	if roles.IsEndorser() {
		rolesValue := make(map[roles.Role]struct{})
		rolesValue[roles.CommitterRole] = struct{}{}
		roles.SetRoles(rolesValue)
		defer func() { roles.SetRoles(nil) }()
	}

	require.True(t, IsPvtDataReconcilerEnabled(true))
	require.False(t, IsPvtDataReconcilerEnabled(false))
}

func TestIsPvtDataReconcilerEnabledByEndorser(t *testing.T) {

	//make sure roles is endorser not committer
	if roles.IsCommitter() {
		rolesValue := make(map[roles.Role]struct{})
		rolesValue[roles.EndorserRole] = struct{}{}
		roles.SetRoles(rolesValue)
		defer func() { roles.SetRoles(nil) }()
	}

	require.False(t, IsPvtDataReconcilerEnabled(true))
	require.False(t, IsPvtDataReconcilerEnabled(false))
}
