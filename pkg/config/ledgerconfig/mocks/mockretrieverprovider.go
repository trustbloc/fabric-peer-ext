/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"github.com/trustbloc/fabric-peer-ext/pkg/config/ledgerconfig/state/api"
)

// StateRetrieverProvider is a mock implementation of RetrieverProvider
type StateRetrieverProvider struct {
	r   api.StateRetriever
	err error
}

// NewStateRetrieverProvider returns a mock StateRetrieverProvider
func NewStateRetrieverProvider() *StateRetrieverProvider {
	return &StateRetrieverProvider{
		r: NewStateRetriever(),
	}
}

// WithStateRetriever sets the StateRetriever
func (m *StateRetrieverProvider) WithStateRetriever(r api.StateRetriever) *StateRetrieverProvider {
	m.r = r
	return m
}

// WithError injects the provider with an error
func (m *StateRetrieverProvider) WithError(err error) *StateRetrieverProvider {
	m.err = err
	return m
}

// GetStateRetriever returns a mock StateRetriever
func (m *StateRetrieverProvider) GetStateRetriever() (api.StateRetriever, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.r, nil
}
