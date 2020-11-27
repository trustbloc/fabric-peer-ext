// +build testing

/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package appdata

// NewRetrieverForTest creates a retriever used for unit tests
func NewRetrieverForTest(channelID string, gossip gossipService, gossipMaxAttempts, gossipMaxPeers int, reqCreator requestCreator) *Retriever {
	return newRetriever(channelID, gossip, gossipMaxAttempts, gossipMaxPeers, reqCreator)
}
