/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"sync"

	validatorv20 "github.com/hyperledger/fabric/core/committer/txvalidator/v20"
)

// TxValidator implements a mock transaction validator
type TxValidator struct {
	results map[int][]*validatorv20.BlockValidationResult
	mutex   sync.Mutex
}

// NewTxValidator returns a mock transaction validator
func NewTxValidator() *TxValidator {
	return &TxValidator{
		results: make(map[int][]*validatorv20.BlockValidationResult),
	}
}

// WithValidationResult adds mock validation results that will be submitted when Validate is invoked
func (m *TxValidator) WithValidationResult(r *validatorv20.BlockValidationResult) *TxValidator {
	m.results[r.TIdx] = append(m.results[r.TIdx], r)

	return m
}

// ValidateTx submits any mock validation results to the given results channel
func (m *TxValidator) ValidateTx(req *validatorv20.BlockValidationRequest, resultsChan chan<- *validatorv20.BlockValidationResult) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	results := m.results[req.TIdx]

	if len(results) == 0 {
		return
	}

	r := results[0]

	m.results[req.TIdx] = results[1:]

	resultsChan <- r
}
