/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"crypto"
	"time"

	"github.com/golang/protobuf/proto"
	mp "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric/msp"
)

// Identity implements identity
type Identity struct {
	mspID      string
	identifier *msp.IdentityIdentifier
	err        error
}

// NewIdentity creates new mock identity
func NewIdentity() *Identity {
	return &Identity{}
}

// WithMSPID sets the identifier
func (id *Identity) WithMSPID(mspID string) *Identity {
	id.mspID = mspID
	id.identifier = &msp.IdentityIdentifier{Mspid: mspID}
	return id
}

// WithError injects an error
func (id *Identity) WithError(err error) *Identity {
	id.err = err
	return id
}

// ExpiresAt returns the time at which the Identity expires.
func (id *Identity) ExpiresAt() time.Time {
	return time.Time{}
}

// SatisfiesPrincipal returns null if this instance matches the supplied principal or an error otherwise
func (id *Identity) SatisfiesPrincipal(principal *mp.MSPPrincipal) error {
	return id.err
}

// GetIdentifier returns the identifier (MSPID/IDID) for this instance
func (id *Identity) GetIdentifier() *msp.IdentityIdentifier {
	return id.identifier
}

// GetMSPIdentifier returns the MSP identifier for this instance
func (id *Identity) GetMSPIdentifier() string {
	return id.mspID
}

// Validate returns nil if this instance is a valid identity or an error otherwise
func (id *Identity) Validate() error {
	return id.err
}

// GetOrganizationalUnits returns the OU for this instance
func (id *Identity) GetOrganizationalUnits() []*msp.OUIdentifier {
	return nil
}

// Verify checks against a signature and a message
// to determine whether this identity produced the
// signature; it returns nil if so or an error otherwise
func (id *Identity) Verify(msg []byte, sig []byte) error {
	return id.err
}

// Serialize returns a byte array representation of this identity
func (id *Identity) Serialize() ([]byte, error) {
	serializedIdentity := &mp.SerializedIdentity{
		Mspid:   id.mspID,
		IdBytes: []byte(id.mspID),
	}
	return proto.Marshal(serializedIdentity)
}

// Anonymous ...
func (id *Identity) Anonymous() bool {
	return false
}

// MockSigningIdentity ...
type MockSigningIdentity struct {
	// we embed everything from a base identity
	Identity

	// signer corresponds to the object that can produce signatures from this identity
	Signer crypto.Signer
}

// NewMockSigningIdentity ...
func NewMockSigningIdentity() (msp.SigningIdentity, error) {
	return &MockSigningIdentity{}, nil
}

// Sign produces a signature over msg, signed by this instance
func (id *MockSigningIdentity) Sign(msg []byte) ([]byte, error) {
	return []byte(""), nil
}

// GetPublicVersion ...
func (id *MockSigningIdentity) GetPublicVersion() msp.Identity {
	return nil
}
