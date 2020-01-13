/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"errors"
	"testing"

	"github.com/hyperledger/fabric/bccsp"
	"github.com/stretchr/testify/require"
	"github.com/trustbloc/fabric-peer-ext/pkg/txn/client/mocks"
)

//go:generate counterfeiter -o ./mocks/bccsp.gen.go --fake-name BCCSP github.com/hyperledger/fabric/bccsp.BCCSP
//go:generate counterfeiter -o ./mocks/bccspkey.gen.go --fake-name BCCSPKey github.com/hyperledger/fabric/bccsp.Key
//go:generate counterfeiter -o ./mocks/hash.gen.go --fake-name Hash hash.Hash

func TestCorePkg_CreateCryptoSuiteProvider(t *testing.T) {
	pkg := newCorePkg()
	require.NotNil(t, pkg)

	p, err := pkg.CreateCryptoSuiteProvider(nil)
	require.NoError(t, err)
	require.NotNil(t, p)
}

func TestCryptoSuite_KeyGen(t *testing.T) {
	b := &mocks.BCCSP{}

	s := newCryptoSuite(b)
	require.NotNil(t, s)

	t.Run("success", func(t *testing.T) {
		key, err := s.KeyGen(&bccsp.ECDSAKeyGenOpts{})
		require.NoError(t, err)
		require.NotNil(t, key)
	})

	t.Run("Invalid opts", func(t *testing.T) {
		key, err := s.KeyGen(&unsupportedOptsType{})
		require.Error(t, err)
		require.Nil(t, key)
	})

	t.Run("KeyGen error", func(t *testing.T) {
		errExpected := errors.New("injected key gen error")
		b.KeyGenReturns(nil, errExpected)

		key, err := s.KeyGen(&bccsp.ECDSAKeyGenOpts{})
		require.EqualError(t, err, errExpected.Error())
		require.Nil(t, key)
	})
}

func TestCryptoSuite_KeyImport(t *testing.T) {
	b := &mocks.BCCSP{}

	s := newCryptoSuite(b)
	require.NotNil(t, s)

	t.Run("success", func(t *testing.T) {
		key, err := s.KeyImport(nil, &bccsp.AES256ImportKeyOpts{})
		require.NoError(t, err)
		require.NotNil(t, key)
	})

	t.Run("Invalid opts", func(t *testing.T) {
		key, err := s.KeyImport(nil, &unsupportedOptsType{})
		require.Error(t, err)
		require.Nil(t, key)
	})

	t.Run("KeyImport error", func(t *testing.T) {
		errExpected := errors.New("injected key import error")
		b.KeyImportReturns(nil, errExpected)

		key, err := s.KeyImport(nil, &bccsp.AES256ImportKeyOpts{})
		require.EqualError(t, err, errExpected.Error())
		require.Nil(t, key)
	})
}

func TestCryptoSuite_GetKey(t *testing.T) {
	b := &mocks.BCCSP{}

	s := newCryptoSuite(b)
	require.NotNil(t, s)

	t.Run("success", func(t *testing.T) {
		key, err := s.GetKey(nil)
		require.NoError(t, err)
		require.NotNil(t, key)
	})

	t.Run("GetKey error", func(t *testing.T) {
		errExpected := errors.New("injected get key error")
		b.GetKeyReturns(nil, errExpected)

		key, err := s.GetKey(nil)
		require.EqualError(t, err, errExpected.Error())
		require.Nil(t, key)
	})
}

func TestCryptoSuite_GetHash(t *testing.T) {
	b := &mocks.BCCSP{}
	b.GetHashReturns(&mocks.Hash{}, nil)

	s := newCryptoSuite(b)
	require.NotNil(t, s)

	t.Run("success", func(t *testing.T) {
		o, err := s.GetHash(&bccsp.SHA256Opts{})
		require.NoError(t, err)
		require.NotNil(t, o)
	})

	t.Run("Invalid opts", func(t *testing.T) {
		o, err := s.GetHash(&unsupportedHashOptsType{})
		require.Error(t, err)
		require.Nil(t, o)
	})
}

func TestCryptoSuite_Hash(t *testing.T) {
	b := &mocks.BCCSP{}
	h := []byte("hash")
	b.HashReturns(h, nil)

	s := newCryptoSuite(b)
	require.NotNil(t, s)

	t.Run("success", func(t *testing.T) {
		bytes, err := s.Hash([]byte(""), &bccsp.SHA256Opts{})
		require.NoError(t, err)
		require.Equal(t, h, bytes)
	})

	t.Run("Invalid opts", func(t *testing.T) {
		bytes, err := s.Hash([]byte(""), &unsupportedHashOptsType{})
		require.Error(t, err)
		require.Empty(t, bytes)
	})
}

func TestCryptoSuite_Sign(t *testing.T) {
	b := &mocks.BCCSP{}

	s := newCryptoSuite(b)
	require.NotNil(t, s)

	t.Run("success", func(t *testing.T) {
		sig := []byte("signature")
		b.SignReturns(sig, nil)

		bytes, err := s.Sign(&key{}, nil, nil)
		require.NoError(t, err)
		require.Equal(t, sig, bytes)
	})

	t.Run("error", func(t *testing.T) {
		errExpected := errors.New("injected sign error")
		b.SignReturns(nil, errExpected)

		bytes, err := s.Sign(&key{}, nil, nil)
		require.EqualError(t, err, errExpected.Error())
		require.Empty(t, bytes)
	})
}

func TestCryptoSuite_Verify(t *testing.T) {
	b := &mocks.BCCSP{}

	s := newCryptoSuite(b)
	require.NotNil(t, s)

	t.Run("success", func(t *testing.T) {
		b.VerifyReturns(true, nil)

		ok, err := s.Verify(&key{}, nil, nil, nil)
		require.NoError(t, err)
		require.True(t, ok)
	})

	t.Run("error", func(t *testing.T) {
		errExpected := errors.New("injected sign error")
		b.VerifyReturns(false, errExpected)

		ok, err := s.Verify(&key{}, nil, nil, nil)
		require.EqualError(t, err, errExpected.Error())
		require.False(t, ok)
	})
}

func TestGetBCCSPHashOpts(t *testing.T) {
	t.Run(shaHashOptsType, func(t *testing.T) {
		o, err := getBCCSPHashOpts(&bccsp.SHAOpts{})
		require.NoError(t, err)
		require.IsType(t, &bccsp.SHAOpts{}, o)
	})

	t.Run(sha256HashOptsType, func(t *testing.T) {
		o, err := getBCCSPHashOpts(&bccsp.SHA256Opts{})
		require.NoError(t, err)
		require.IsType(t, &bccsp.SHA256Opts{}, o)
	})

	t.Run(sha384HashOptsType, func(t *testing.T) {
		o, err := getBCCSPHashOpts(&bccsp.SHA384Opts{})
		require.NoError(t, err)
		require.IsType(t, &bccsp.SHA384Opts{}, o)
	})

	t.Run(sha3256HashOptsType, func(t *testing.T) {
		o, err := getBCCSPHashOpts(&bccsp.SHA3_256Opts{})
		require.NoError(t, err)
		require.IsType(t, &bccsp.SHA3_256Opts{}, o)
	})

	t.Run(sha3384HashOptsType, func(t *testing.T) {
		o, err := getBCCSPHashOpts(&bccsp.SHA3_384Opts{})
		require.NoError(t, err)
		require.IsType(t, &bccsp.SHA3_384Opts{}, o)
	})

	t.Run("unsupported", func(t *testing.T) {
		o, err := getBCCSPHashOpts(&unsupportedHashOptsType{})
		require.Error(t, err)
		require.Nil(t, o)
	})
}

func TestGetBCCSPKeyImportOpts(t *testing.T) {
	t.Run(aes256ImportKeyOptsType, func(t *testing.T) {
		o, err := getBCCSPKeyImportOpts(&bccsp.AES256ImportKeyOpts{Temporary: true})
		require.NoError(t, err)
		require.IsType(t, &bccsp.AES256ImportKeyOpts{}, o)
		require.True(t, o.Ephemeral())
	})

	t.Run(hmacImportKeyOptsType, func(t *testing.T) {
		o, err := getBCCSPKeyImportOpts(&bccsp.HMACImportKeyOpts{Temporary: true})
		require.NoError(t, err)
		require.IsType(t, &bccsp.HMACImportKeyOpts{}, o)
		require.True(t, o.Ephemeral())
	})

	t.Run(ecdsaPKIXPublicKeyImportOptsType, func(t *testing.T) {
		o, err := getBCCSPKeyImportOpts(&bccsp.ECDSAPKIXPublicKeyImportOpts{Temporary: true})
		require.NoError(t, err)
		require.IsType(t, &bccsp.ECDSAPKIXPublicKeyImportOpts{}, o)
		require.True(t, o.Ephemeral())
	})

	t.Run(ecdsaPrivateKeyImportOptsType, func(t *testing.T) {
		o, err := getBCCSPKeyImportOpts(&bccsp.ECDSAPrivateKeyImportOpts{Temporary: true})
		require.NoError(t, err)
		require.IsType(t, &bccsp.ECDSAPrivateKeyImportOpts{}, o)
		require.True(t, o.Ephemeral())
	})

	t.Run(ecdsaGoPublicKeyImportOptsType, func(t *testing.T) {
		o, err := getBCCSPKeyImportOpts(&bccsp.ECDSAGoPublicKeyImportOpts{Temporary: true})
		require.NoError(t, err)
		require.IsType(t, &bccsp.ECDSAGoPublicKeyImportOpts{}, o)
		require.True(t, o.Ephemeral())
	})

	t.Run(x509PublicKeyImportOptsType, func(t *testing.T) {
		o, err := getBCCSPKeyImportOpts(&bccsp.X509PublicKeyImportOpts{Temporary: true})
		require.NoError(t, err)
		require.IsType(t, &bccsp.X509PublicKeyImportOpts{}, o)
		require.True(t, o.Ephemeral())
	})

	t.Run("unsupported", func(t *testing.T) {
		o, err := getBCCSPKeyImportOpts(&unsupportedOptsType{})
		require.Error(t, err)
		require.Nil(t, o)
	})
}

func TestGetBCCSPKeyGenOpts(t *testing.T) {
	t.Run(ecdsaKeyGenOptsType, func(t *testing.T) {
		o, err := getBCCSPKeyGenOpts(&bccsp.ECDSAKeyGenOpts{Temporary: true})
		require.NoError(t, err)
		require.IsType(t, &bccsp.ECDSAKeyGenOpts{}, o)
		require.True(t, o.Ephemeral())
	})

	t.Run(ecdsaP256KeyGenOptsType, func(t *testing.T) {
		o, err := getBCCSPKeyGenOpts(&bccsp.ECDSAP256KeyGenOpts{Temporary: true})
		require.NoError(t, err)
		require.IsType(t, &bccsp.ECDSAP256KeyGenOpts{}, o)
		require.True(t, o.Ephemeral())
	})

	t.Run(ecdsaP384KeyGenOptsType, func(t *testing.T) {
		o, err := getBCCSPKeyGenOpts(&bccsp.ECDSAP384KeyGenOpts{Temporary: true})
		require.NoError(t, err)
		require.IsType(t, &bccsp.ECDSAP384KeyGenOpts{}, o)
		require.True(t, o.Ephemeral())
	})

	t.Run(aesKeyGenOptsType, func(t *testing.T) {
		o, err := getBCCSPKeyGenOpts(&bccsp.AESKeyGenOpts{Temporary: true})
		require.NoError(t, err)
		require.IsType(t, &bccsp.AESKeyGenOpts{}, o)
		require.True(t, o.Ephemeral())
	})

	t.Run(aes256KeyGenOptsType, func(t *testing.T) {
		o, err := getBCCSPKeyGenOpts(&bccsp.AES256KeyGenOpts{Temporary: true})
		require.NoError(t, err)
		require.IsType(t, &bccsp.AES256KeyGenOpts{}, o)
		require.True(t, o.Ephemeral())
	})

	t.Run(aes192KeyGenOptsType, func(t *testing.T) {
		o, err := getBCCSPKeyGenOpts(&bccsp.AES192KeyGenOpts{Temporary: true})
		require.NoError(t, err)
		require.IsType(t, &bccsp.AES192KeyGenOpts{}, o)
		require.True(t, o.Ephemeral())
	})

	t.Run(aes128KeyGenOptsType, func(t *testing.T) {
		o, err := getBCCSPKeyGenOpts(&bccsp.AES128KeyGenOpts{Temporary: true})
		require.NoError(t, err)
		require.IsType(t, &bccsp.AES128KeyGenOpts{}, o)
		require.True(t, o.Ephemeral())
	})

	t.Run("unsupported", func(t *testing.T) {
		o, err := getBCCSPKeyGenOpts(&unsupportedOptsType{})
		require.Error(t, err)
		require.Nil(t, o)
	})
}

func TestKey(t *testing.T) {
	bk := &mocks.BCCSPKey{}
	k := &key{key: bk}

	t.Run("Bytes", func(t *testing.T) {
		b := []byte("bytes")
		bk.BytesReturns(b, nil)
		bytes, err := k.Bytes()
		require.NoError(t, err)
		require.Equal(t, b, bytes)
	})

	t.Run("Private", func(t *testing.T) {
		bk.PrivateReturns(true)
		require.True(t, k.Private())
	})

	t.Run("PublicKey", func(t *testing.T) {
		pk := &mocks.BCCSPKey{}
		pkBytes := []byte("public key")
		pk.BytesReturns(pkBytes, nil)

		bk.PublicKeyReturns(pk, nil)
		pubkey, err := k.PublicKey()
		require.NoError(t, err)

		pubKeyBytes, err := pubkey.Bytes()
		require.NoError(t, err)
		require.Equal(t, pkBytes, pubKeyBytes)

		errExpected := errors.New("public key error")
		bk.PublicKeyReturns(nil, errExpected)
		pubkey, err = k.PublicKey()
		require.EqualError(t, err, errExpected.Error())
	})

	t.Run("SKI", func(t *testing.T) {
		b := []byte("bytes")
		bk.SKIReturns(b)
		require.Equal(t, b, k.SKI())
	})

	t.Run("SKI", func(t *testing.T) {
		bk.SymmetricReturns(true)
		require.True(t, k.Symmetric())
	})
}

type unsupportedHashOptsType struct {
}

func (o *unsupportedHashOptsType) Algorithm() string {
	return ""
}

type unsupportedOptsType struct {
}

func (o *unsupportedOptsType) Algorithm() string {
	return ""
}

func (o *unsupportedOptsType) Ephemeral() bool {
	return false
}
