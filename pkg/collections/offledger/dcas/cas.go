/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dcas

import (
	"fmt"

	"github.com/ipfs/go-cid"
	dshelp "github.com/ipfs/go-ipfs-ds-help"
)

const (
	// The go-ipfs-blockstore library that is being used for DCAS
	// always prefixes the key with "/blocks"
	dsPrefix = "/blocks"
)

// GetCID returns the content ID (CID) of the value using the given CID version, codec, and multi-hash type.
func GetCID(content []byte, version CIDVersion, codec, mhType uint64) (string, error) {
	cID, err := getCID(content, version, codec, mhType)
	if err != nil {
		return "", err
	}

	return cID.String(), nil
}

// GetCASKey returns the key which will be used in the data store to store the value.
func GetCASKey(content []byte, version CIDVersion, codec, mhType uint64) (string, error) {
	cID, err := getCID(content, version, codec, mhType)
	if err != nil {
		return "", err
	}

	return dsPrefix + dshelp.NewKeyFromBinary(cID.Bytes()).String(), nil
}

// ValidateCID validates the given content ID
func ValidateCID(id string) error {
	_, err := cid.Parse(id)
	return err
}

// CIDVersion specifies the version of the content ID (CID)
type CIDVersion = uint64

const (
	// CIDV0 content ID (CID) version 0
	CIDV0 CIDVersion = 0
	// CIDV1 content ID (CID) version 1
	CIDV1 CIDVersion = 1
)

func getCID(content []byte, version CIDVersion, codec, mhType uint64) (cid.Cid, error) {
	var b cid.Builder

	switch version {
	case CIDV1:
		b = cid.V1Builder{
			Codec:  codec,
			MhType: mhType,
		}
	case CIDV0:
		b = cid.V0Builder{}
	default:
		return cid.Cid{}, fmt.Errorf("unsupported CID version [%d]", version)
	}

	c, err := b.Sum(content)
	if err != nil {
		return cid.Cid{}, err
	}

	return c, nil
}
