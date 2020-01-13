/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"hash"
	"reflect"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	coreApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/factory/defcore"
	"github.com/hyperledger/fabric/bccsp"
	bccspfactory "github.com/hyperledger/fabric/bccsp/factory"
	"github.com/pkg/errors"
)

const (
	// Import key option types
	aes256ImportKeyOptsType          = "*bccsp.AES256ImportKeyOpts"
	hmacImportKeyOptsType            = "*bccsp.HMACImportKeyOpts"
	ecdsaPKIXPublicKeyImportOptsType = "*bccsp.ECDSAPKIXPublicKeyImportOpts"
	ecdsaPrivateKeyImportOptsType    = "*bccsp.ECDSAPrivateKeyImportOpts"
	ecdsaGoPublicKeyImportOptsType   = "*bccsp.ECDSAGoPublicKeyImportOpts"
	x509PublicKeyImportOptsType      = "*bccsp.X509PublicKeyImportOpts"

	// Hash option types
	shaHashOptsType     = "*bccsp.SHAOpts"
	sha256HashOptsType  = "*bccsp.SHA256Opts"
	sha384HashOptsType  = "*bccsp.SHA384Opts"
	sha3256HashOptsType = "*bccsp.SHA3_256Opts"
	sha3384HashOptsType = "*bccsp.SHA3_384Opts"

	// Key gen option types
	ecdsaKeyGenOptsType     = "*bccsp.ECDSAKeyGenOpts"
	ecdsaP256KeyGenOptsType = "*bccsp.ECDSAP256KeyGenOpts"
	ecdsaP384KeyGenOptsType = "*bccsp.ECDSAP384KeyGenOpts"
	aesKeyGenOptsType       = "*bccsp.AESKeyGenOpts"
	aes256KeyGenOptsType    = "*bccsp.AES256KeyGenOpts"
	aes192KeyGenOptsType    = "*bccsp.AES192KeyGenOpts"
	aes128KeyGenOptsType    = "*bccsp.AES128KeyGenOpts"
)

// corePkg is will provide custom sdk core pkg
type corePkg struct {
	defcore.ProviderFactory
}

func newCorePkg() *corePkg {
	return &corePkg{}
}

// CreateCryptoSuiteProvider returns an implementation of factory default bccsp cryptosuite
func (f *corePkg) CreateCryptoSuiteProvider(_ core.CryptoSuiteConfig) (core.CryptoSuite, error) {
	return newCryptoSuite(bccspfactory.GetDefault()), nil
}

type cryptoSuite struct {
	bccsp bccsp.BCCSP
}

var newCryptoSuite = func(bccsp bccsp.BCCSP) *cryptoSuite {
	return &cryptoSuite{bccsp: bccsp}
}

// KeyGen generates a key using opts.
func (c *cryptoSuite) KeyGen(opts coreApi.KeyGenOpts) (coreApi.Key, error) {
	o, err := getBCCSPKeyGenOpts(opts)
	if err != nil {
		return nil, err
	}

	k, err := c.bccsp.KeyGen(o)
	if err != nil {
		return nil, err
	}

	return &key{key: k}, nil
}

// KeyImport imports a key from its raw representation using opts.
// The opts argument should be appropriate for the primitive used.
func (c *cryptoSuite) KeyImport(raw interface{}, opts coreApi.KeyImportOpts) (coreApi.Key, error) {
	o, err := getBCCSPKeyImportOpts(opts)
	if err != nil {
		return nil, err
	}

	k, err := c.bccsp.KeyImport(raw, o)
	if err != nil {
		return nil, err
	}

	return &key{key: k}, nil
}

// GetKey returns the key this CSP associates to
// the Subject Key Identifier ski.
func (c *cryptoSuite) GetKey(ski []byte) (coreApi.Key, error) {
	k, err := c.bccsp.GetKey(ski)
	if err != nil {
		return nil, err
	}

	return &key{key: k}, nil
}

// Hash hashes messages msg using options opts.
// If opts is nil, the default hash function will be used.
func (c *cryptoSuite) Hash(msg []byte, opts coreApi.HashOpts) ([]byte, error) {
	o, err := getBCCSPHashOpts(opts)
	if err != nil {
		return nil, errors.WithMessage(err, "unable to hash the message")
	}

	return c.bccsp.Hash(msg, o)
}

// GetHash returns and instance of hash.Hash using options opts.
// If opts is nil, the default hash function will be returned.
func (c *cryptoSuite) GetHash(opts coreApi.HashOpts) (hash.Hash, error) {
	o, err := getBCCSPHashOpts(opts)
	if err != nil {
		return nil, errors.WithMessage(err, "unable to get hash")
	}

	return c.bccsp.GetHash(o)
}

// Sign signs digest using key k.
// The opts argument should be appropriate for the algorithm used.
//
// Note that when a signature of a hash of a larger message is needed,
// the caller is responsible for hashing the larger message and passing
// the hash (as digest).
func (c *cryptoSuite) Sign(k coreApi.Key, digest []byte, opts coreApi.SignerOpts) ([]byte, error) {
	return c.bccsp.Sign(k.(*key).key, digest, opts)
}

// Verify verifies signature against key k and digest
// The opts argument should be appropriate for the algorithm used.
func (c *cryptoSuite) Verify(k coreApi.Key, signature, digest []byte, opts coreApi.SignerOpts) (bool, error) {
	return c.bccsp.Verify(k.(*key).key, signature, digest, opts)
}

type key struct {
	key bccsp.Key
}

func (k *key) Bytes() ([]byte, error) {
	return k.key.Bytes()
}

func (k *key) SKI() []byte {
	return k.key.SKI()
}

func (k *key) Symmetric() bool {
	return k.key.Symmetric()
}

func (k *key) Private() bool {
	return k.key.Private()
}

func (k *key) PublicKey() (coreApi.Key, error) {
	pubKey, err := k.key.PublicKey()
	if err != nil {
		return nil, err
	}

	return &key{key: pubKey}, nil
}

// getBCCSPKeyImportOpts converts the SDK's implementation of KeyImportOpts to fabric bccsp KeyImportTypes.
// Reason: Reflect check on opts type in bccsp implementation - so they must be the local Fabric types.
func getBCCSPKeyImportOpts(opts coreApi.KeyImportOpts) (bccsp.KeyImportOpts, error) {
	// Note: Since the SDK hides the opts in the internal package, we need to convert the type to a string
	optType := reflect.TypeOf(opts).String()

	switch optType {
	case aes256ImportKeyOptsType:
		return &bccsp.AES256ImportKeyOpts{Temporary: opts.Ephemeral()}, nil
	case hmacImportKeyOptsType:
		return &bccsp.HMACImportKeyOpts{Temporary: opts.Ephemeral()}, nil
	case ecdsaPKIXPublicKeyImportOptsType:
		return &bccsp.ECDSAPKIXPublicKeyImportOpts{Temporary: opts.Ephemeral()}, nil
	case ecdsaPrivateKeyImportOptsType:
		return &bccsp.ECDSAPrivateKeyImportOpts{Temporary: opts.Ephemeral()}, nil
	case ecdsaGoPublicKeyImportOptsType:
		return &bccsp.ECDSAGoPublicKeyImportOpts{Temporary: opts.Ephemeral()}, nil
	case x509PublicKeyImportOptsType:
		return &bccsp.X509PublicKeyImportOpts{Temporary: opts.Ephemeral()}, nil
	default:
		return nil, errors.Errorf("unsupported KeyImportOpts type: [%s]", optType)
	}
}

// getBCCSPHashOpts converts the SDK's implementation of HashOpts to fabric bccsp HashOpts
// Reason: Reflect check on opts type in bccsp implementation - so they must be the local Fabric types.
func getBCCSPHashOpts(opts coreApi.HashOpts) (bccsp.HashOpts, error) {
	// Note: Since the SDK hides the opts in the internal package, we need to convert the type to a string
	optType := reflect.TypeOf(opts).String()

	switch optType {
	case shaHashOptsType:
		return &bccsp.SHAOpts{}, nil
	case sha256HashOptsType:
		return &bccsp.SHA256Opts{}, nil
	case sha384HashOptsType:
		return &bccsp.SHA384Opts{}, nil
	case sha3256HashOptsType:
		return &bccsp.SHA3_256Opts{}, nil
	case sha3384HashOptsType:
		return &bccsp.SHA3_384Opts{}, nil
	default:
		return nil, errors.Errorf("unsupported HashOpts type: [%s]", optType)
	}
}

// getBCCSPKeyGenOpts converts the SDK's implementation of KeyGenOpts to fabric bccsp KeyGenOpts
// Reason: Reflect check on opts type in bccsp implementation - so they must be the local Fabric types.
func getBCCSPKeyGenOpts(opts coreApi.KeyGenOpts) (bccsp.KeyGenOpts, error) {
	// Note: Since the SDK hides the opts in the internal package, we need to convert the type to a string
	optType := reflect.TypeOf(opts).String()

	switch optType {
	case ecdsaKeyGenOptsType:
		return &bccsp.ECDSAKeyGenOpts{Temporary: opts.Ephemeral()}, nil
	case ecdsaP256KeyGenOptsType:
		return &bccsp.ECDSAP256KeyGenOpts{Temporary: opts.Ephemeral()}, nil
	case ecdsaP384KeyGenOptsType:
		return &bccsp.ECDSAP384KeyGenOpts{Temporary: opts.Ephemeral()}, nil
	case aesKeyGenOptsType:
		return &bccsp.AESKeyGenOpts{Temporary: opts.Ephemeral()}, nil
	case aes256KeyGenOptsType:
		return &bccsp.AES256KeyGenOpts{Temporary: opts.Ephemeral()}, nil
	case aes192KeyGenOptsType:
		return &bccsp.AES192KeyGenOpts{Temporary: opts.Ephemeral()}, nil
	case aes128KeyGenOptsType:
		return &bccsp.AES128KeyGenOpts{Temporary: opts.Ephemeral()}, nil
	default:
		return nil, errors.Errorf("unsupported KeyGenOpts type: [%s]", optType)
	}
}
