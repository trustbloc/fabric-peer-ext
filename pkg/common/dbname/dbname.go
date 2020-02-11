/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dbname

import (
	"regexp"
	"strings"

	"github.com/hyperledger/fabric/common/flogging"
)

var logger = flogging.MustGetLogger("ext_common")

// PartitionType defines the type of DB partition
type PartitionType string

const (
	// PartitionNone indicates that the DB shouldn't be partitioned
	PartitionNone PartitionType = ""
	// PartitionPeer indicates that the DB should be partitioned by peer ID
	PartitionPeer PartitionType = "PEER"
	// PartitionMSP indicates that the DB should be partitioned by MSP ID
	PartitionMSP PartitionType = "MSP"
)

const allowableChars = "[0-9]|[a-z]|[\\_\\$\\(\\)+\\-\\/]"

type peerConfig interface {
	PeerID() string
	MSPID() string
	DBPartitionType() string
}

// Resolver resolves database names according to the configured partition type.
type Resolver struct {
	prefix string
}

// ResolverInstance is the singleton instance of a DB name resolver
var ResolverInstance = &Resolver{}

// Initialize is called on peer startup to initialize the Resolver
func (r *Resolver) Initialize(peerConfig peerConfig) *Resolver {
	logger.Infof("Initializing peer with DB partition type: [%s]", peerConfig.DBPartitionType())

	switch PartitionType(peerConfig.DBPartitionType()) {
	case PartitionPeer:
		r.prefix = prefix(peerConfig.PeerID())
	case PartitionMSP:
		r.prefix = prefix(peerConfig.MSPID())
	}

	return r
}

// IsRelevant returns true if the given database name is one of the local peer's databases.
// Otherwise, false if the database is not relevant to the peer (i.e. it may be a database
// belonging to another peer/cluster).
func (r *Resolver) IsRelevant(dbName string) bool {
	if dbName[0:1] == "_" {
		// System database
		return true
	}

	if r.prefix == "" {
		return true
	}

	return strings.HasPrefix(dbName, r.prefix)
}

// Resolve resolves the database name according to partition type.
// If partition type is not specified then the database name should not be changed.
// If partition type is PER_PEER then the peer ID is used as a prefix to the DB name.
// If partition type is PER_MSP then the MSP ID is used as a prefix to the DB name.
func (r *Resolver) Resolve(name string) string {
	if name[0:1] == "_" {
		// System database
		return name
	}

	return r.prefix + name
}

// replace replaces all illegal characters in the given string with the given character.
// Allowed characters are: a-z, 0-9, and any of the characters: _$()+-/
// All other characters are replaced with the given char.
func replace(pattern *regexp.Regexp, str string, rc rune) string {
	rar := make([]rune, len(str))
	for i, c := range strings.ToLower(str) {
		if pattern.Match([]byte{byte(c)}) {
			rar[i] = c
		} else {
			rar[i] = rc
		}
	}

	return string(rar)
}

func prefix(ns string) string {
	return replace(pattern(), ns, '-') + "_"
}

func pattern() *regexp.Regexp {
	pattern, err := regexp.Compile(allowableChars)
	if err != nil {
		panic(err)
	}
	return pattern
}
