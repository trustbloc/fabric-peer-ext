/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package multirequest

import (
	"context"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/trustbloc/fabric-peer-ext/pkg/common"
)

var logger = flogging.MustGetLogger("ext_multirequest")

// Request is the request to execute
type Request func(ctxt context.Context) (common.Values, error)

type req struct {
	id      string
	execute Request
}

type res struct {
	id     string
	values common.Values
	err    error
}

// Response contains the response for a given request ID
type Response struct {
	RequestID string
	Values    common.Values
}

// MultiRequest executes multiple requests and returns the first, non-error response
type MultiRequest struct {
	requests []*req
}

// New returns a new MultiRequest
func New() *MultiRequest {
	return &MultiRequest{}
}

// Add adds a request function
func (r *MultiRequest) Add(id string, execute Request) {
	r.requests = append(r.requests, &req{id: id, execute: execute})
}

// Execute executes the requests concurrently and returns the responses.
func (r *MultiRequest) Execute(ctxt context.Context) *Response {
	respChan := make(chan *res, len(r.requests)+1)

	cctxt, cancel := context.WithCancel(ctxt)
	defer cancel()

	for _, request := range r.requests {
		go func(r *req) {
			ccctxt, cancelReq := context.WithCancel(cctxt)
			defer cancelReq()
			values, err := r.execute(ccctxt)
			respChan <- &res{id: r.id, values: values, err: err}
		}(request)
	}

	resp := &Response{}
	done := false

	// Wait for all responses
	for range r.requests {
		response := <-respChan

		if done {
			continue
		}

		if handleResponse(response, resp) {
			done = true
			cancel()
		}
	}

	return resp
}

func handleResponse(response *res, resp *Response) bool {
	if response.err != nil {
		logger.Debugf("Error response was received from [%s]: %s", response.id, response.err)
		return false
	}

	resp.RequestID = response.id
	resp.Values = resp.Values.Merge(response.values)

	if response.values.AllSet() {
		logger.Debugf("All values were received from [%s]", response.id)
		return true
	}

	logger.Debugf("One or more values are missing in response from [%s]", response.id)
	return false
}
