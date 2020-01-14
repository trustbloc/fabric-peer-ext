/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"errors"
	"testing"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/client/mocks"
)

//go:generate counterfeiter -o ./mocks/cryptosuite.gen.go --fake-name CryptoSuite github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core.CryptoSuite

func TestNewIdentityManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		epCfg := &mocks.EndpointConfig{}
		epCfg.NetworkConfigReturns(&fab.NetworkConfig{
			Organizations: map[string]fab.OrganizationConfig{"org1msp": {}},
		})

		m, err := newIdentityManager("org1MSP", &mocks.CryptoSuite{}, epCfg, "./msp")
		require.NoError(t, err)
		require.NotNil(t, m)
	})

	t.Run("No endpoint config -> error", func(t *testing.T) {
		m, err := newIdentityManager("org1MSP", &mocks.CryptoSuite{}, nil, "./msp")
		require.EqualError(t, err, "endpoint config is required")
		require.Nil(t, m)
	})

	t.Run("No org name -> error", func(t *testing.T) {
		m, err := newIdentityManager("", nil, &mocks.EndpointConfig{}, "./msp")
		require.EqualError(t, err, "orgName is required")
		require.Nil(t, m)
	})

	t.Run("No crypto suite -> error", func(t *testing.T) {
		m, err := newIdentityManager("org1MSP", nil, &mocks.EndpointConfig{}, "./msp")
		require.EqualError(t, err, "cryptoProvider is required")
		require.Nil(t, m)
	})

	t.Run("Org not found -> error", func(t *testing.T) {
		epCfg := &mocks.EndpointConfig{}
		epCfg.NetworkConfigReturns(&fab.NetworkConfig{})

		m, err := newIdentityManager("org1MSP", &mocks.CryptoSuite{}, epCfg, "./msp")
		require.EqualError(t, err, "org config retrieval failed")
		require.Nil(t, m)
	})

	t.Run("MSP config path not provided -> error", func(t *testing.T) {
		epCfg := &mocks.EndpointConfig{}
		epCfg.NetworkConfigReturns(&fab.NetworkConfig{
			Organizations: map[string]fab.OrganizationConfig{"org1msp": {}},
		})

		m, err := newIdentityManager("org1MSP", &mocks.CryptoSuite{}, epCfg, "")
		require.EqualError(t, err, "either mspConfigPath or an embedded list of users is required")
		require.Nil(t, m)
	})
}

func TestIdentityManager_GetSigningIdentity(t *testing.T) {
	epCfg := &mocks.EndpointConfig{}
	netCfg := &fab.NetworkConfig{
		Organizations: map[string]fab.OrganizationConfig{
			"org1msp": {
				Users: make(map[string]fab.CertKeyPair),
			},
		},
	}
	epCfg.NetworkConfigReturns(netCfg)

	cp := &mocks.CryptoSuite{}
	bkey := &mocks.BCCSPKey{}
	bkey.PrivateReturns(true)

	pBytes := []byte("priv key")
	bkey.BytesReturns(pBytes, nil)

	key := &key{key: bkey}
	cp.KeyImportReturns(key, nil)
	cp.GetKeyReturns(key, nil)

	m, err := newIdentityManager("org1MSP", cp, epCfg, "./testdata")
	require.NoError(t, err)
	require.NotNil(t, m)

	t.Run("user in file -> success", func(t *testing.T) {
		id, err := m.GetSigningIdentity("User1")
		require.NoError(t, err)
		require.NotNil(t, id)

		bytes, err := id.Serialize()
		require.NoError(t, err)
		require.NotEmpty(t, bytes)

		privKey := id.PrivateKey()
		require.NotNil(t, privKey)
		require.True(t, privKey.Private())
		bytes, err = privKey.Bytes()
		require.NoError(t, err)
		require.Equal(t, pBytes, bytes)

		eCert := id.EnrollmentCertificate()
		require.NotEmpty(t, eCert)

		identifier := id.Identifier()
		require.NotNil(t, identifier)

		pubVersion := id.PublicVersion()
		require.Equal(t, id, pubVersion)

		require.EqualError(t, id.Verify(nil, nil), "not implemented")

		bytes, err = id.Sign(nil)
		require.EqualError(t, err, "not implemented")
		require.Empty(t, bytes)
	})

	t.Run("embedded user invalid cert -> error", func(t *testing.T) {
		netCfg.Organizations["org1msp"].Users["user1"] = fab.CertKeyPair{
			Cert: []byte("cert"),
			Key:  []byte("key"),
		}
		defer func() {
			delete(netCfg.Organizations["org1msp"].Users, "user1")
		}()

		id, err := m.GetSigningIdentity("User1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "could not decode pem bytes")
		require.Nil(t, id)
	})

	t.Run("No user -> error", func(t *testing.T) {
		id, err := m.GetSigningIdentity("")
		require.EqualError(t, err, "username is required")
		require.Nil(t, id)
	})

	t.Run("Key import -> error", func(t *testing.T) {
		errExpected := errors.New("injected key import error")
		cp.KeyImportReturns(nil, errExpected)
		defer func() {
			cp.KeyImportReturns(key, nil)
		}()

		id, err := m.GetSigningIdentity("User1")
		require.Error(t, err)
		require.Contains(t, err.Error(), errExpected.Error())
		require.Nil(t, id)
	})
}
