/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/proto"
	pb_msp "github.com/hyperledger/fabric-protos-go/msp"
	coreApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	mspApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/msp"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/pkg/errors"
)

type identityManager struct {
	*msp.IdentityManager
	orgName        string
	mspID          string
	embeddedUsers  map[string]fabApi.CertKeyPair
	keyDir         string
	certDir        string
	config         fabApi.EndpointConfig
	cryptoProvider coreApi.CryptoSuite
}

type user struct {
	mspID                 string
	id                    string
	enrollmentCertificate []byte
	privateKey            coreApi.Key
}

// newIdentityManager Constructor for a custom identity manager.
func newIdentityManager(orgName string, cryptoProvider coreApi.CryptoSuite, config fabApi.EndpointConfig, mspConfigPath string) (*identityManager, error) {
	if orgName == "" {
		return nil, errors.New("orgName is required")
	}

	if cryptoProvider == nil {
		return nil, errors.New("cryptoProvider is required")
	}

	if config == nil {
		return nil, errors.New("endpoint config is required")
	}

	netwkConfig := config.NetworkConfig()

	// viper keys are case insensitive
	orgConfig, ok := netwkConfig.Organizations[strings.ToLower(orgName)]
	if !ok {
		return nil, errors.New("org config retrieval failed")
	}

	if mspConfigPath == "" && len(orgConfig.Users) == 0 {
		return nil, errors.New("either mspConfigPath or an embedded list of users is required")
	}

	mspConfigPath = filepath.Join(orgConfig.CryptoPath, mspConfigPath)

	return &identityManager{
		orgName:        orgName,
		mspID:          orgConfig.MSPID,
		config:         config,
		embeddedUsers:  orgConfig.Users,
		keyDir:         mspConfigPath + "/keystore",
		certDir:        mspConfigPath + "/signcerts",
		cryptoProvider: cryptoProvider,
	}, nil
}

// GetSigningIdentity will sign the given object with provided key,
func (m *identityManager) GetSigningIdentity(userName string) (mspApi.SigningIdentity, error) {
	user, err := m.getUser(userName)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// getUser returns a user for the given user name
func (m *identityManager) getUser(userName string) (*user, error) {
	if userName == "" {
		return nil, errors.New("username is required")
	}

	enrollmentCert, err := m.getEnrollmentCert(userName)
	if err != nil {
		return nil, err
	}

	//Get Key from Pem bytes
	key, err := getCryptoSuiteKeyFromPem(enrollmentCert, m.cryptoProvider)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get cryptosuite key from enrollment cert")
	}

	//Get private key using SKI
	privateKey, err := m.cryptoProvider.GetKey(key.SKI())
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get private key")
	}

	// make sure the key is private for the signingIdentity
	if !privateKey.Private() {
		return nil, errors.New("failed to get private key, found a public key instead")
	}

	return &user{
		mspID:                 m.mspID,
		id:                    userName,
		enrollmentCertificate: enrollmentCert,
		privateKey:            privateKey,
	}, nil
}

func (m *identityManager) getEnrollmentCert(userName string) ([]byte, error) {
	enrollmentCertBytes := m.embeddedUsers[strings.ToLower(userName)].Cert
	if len(enrollmentCertBytes) > 0 {
		logger.Debugf("Found embedded enrollment cert for user [%s]", userName)
		return enrollmentCertBytes, nil
	}

	enrollmentCertDir := strings.Replace(m.certDir, "{userName}", userName, -1)
	enrollmentCertPath, err := getFirstPathFromDir(enrollmentCertDir)
	if err != nil {
		return nil, errors.WithMessagef(err, "find enrollment cert path failed for path [%s]", enrollmentCertDir)
	}

	enrollmentCertBytes, err = ioutil.ReadFile(filepath.Clean(enrollmentCertPath))
	if err != nil {
		return nil, errors.WithMessage(err, "reading enrollment cert path failed")
	}

	logger.Debugf("Found enrollment cert for user [%s] in file [%s]", userName, enrollmentCertPath)

	return enrollmentCertBytes, nil
}

func getFirstPathFromDir(dir string) (string, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return "", errors.Wrap(err, "read directory failed")
	}

	for _, f := range files {
		if !f.IsDir() {
			return filepath.Join(dir, string(filepath.Separator), f.Name()), nil
		}
	}

	return "", errors.New("no paths found")
}

func getCryptoSuiteKeyFromPem(idBytes []byte, cryptoSuite coreApi.CryptoSuite) (coreApi.Key, error) {
	// Decode the pem bytes
	pemCert, _ := pem.Decode(idBytes)
	if pemCert == nil {
		return nil, errors.Errorf("getCryptoSuiteKeyFromPem error: could not decode pem bytes [%v]", idBytes)
	}

	// get a cert
	cert, err := x509.ParseCertificate(pemCert.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "getCryptoSuiteKeyFromPem error: failed to parse x509 cert")
	}

	// get the public key in the right format
	certPubK, err := cryptoSuite.KeyImport(cert, &bccsp.X509PublicKeyImportOpts{Temporary: true})
	if err != nil {
		return nil, errors.Wrap(err, "getCryptoSuiteKeyFromPem error: failed to get public key in right format ")
	}
	return certPubK, nil
}

// Serialize returns a serialized identity
func (u *user) Serialize() ([]byte, error) {
	serializedIdentity := &pb_msp.SerializedIdentity{
		Mspid:   u.mspID,
		IdBytes: u.EnrollmentCertificate(),
	}

	identity, err := proto.Marshal(serializedIdentity)
	if err != nil {
		return nil, errors.Wrap(err, "marshal serializedIdentity failed")
	}

	return identity, nil
}

// PrivateKey return private key
func (u *user) PrivateKey() coreApi.Key {
	return u.privateKey
}

// EnrollmentCertificate return enrollment certificate
func (u *user) EnrollmentCertificate() []byte {
	return u.enrollmentCertificate
}

// Identifier returns user identifier
func (u *user) Identifier() *mspApi.IdentityIdentifier {
	return &mspApi.IdentityIdentifier{MSPID: u.mspID, ID: u.id}
}

// Verify a signature over some message using this identity as reference
func (u *user) Verify(msg []byte, sig []byte) error {
	return errors.New("not implemented")
}

// PublicVersion returns the public parts of this identity
func (u *user) PublicVersion() mspApi.Identity {
	return u
}

// Sign the message
func (u *user) Sign(msg []byte) ([]byte, error) {
	return nil, errors.New("not implemented")
}
