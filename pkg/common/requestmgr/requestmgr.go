/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package requestmgr

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("ext_requestmgr")

// Element contains transient data for a single key
type Element struct {
	Namespace  string
	Collection string
	Key        string
	Value      []byte
	Expiry     time.Time
}

// Elements is a slice of Element
type Elements []*Element

// Get returns the Element matching the given namespace, collection, and key.
func (e Elements) Get(ns, coll, key string) (*Element, bool) {
	for _, element := range e {
		if element.Namespace == ns && element.Collection == coll && element.Key == key {
			return element, true
		}
	}
	return nil, false
}

// Response is the response from a remote peer for a collection of transient data keys
type Response struct {
	Endpoint  string   // The endpoint of the peer that sent the response
	MSPID     string   // The MSP ID of the peer that sent the response
	Signature []byte   // The signature of the peer that provided the data
	Identity  []byte   // The identity of the peer that sent the response
	Data      Elements // The transient data
}

// Request is an interface to get the response
type Request interface {
	ID() uint64
	GetResponse(context context.Context) (*Response, error)
	Cancel()
}

// RequestMgr is an interface to create a new request and to respond to the request
type RequestMgr interface {
	Respond(reqID uint64, response *Response)
	NewRequest() Request
}

type requestMgr struct {
	mutex sync.RWMutex
	mgrs  map[string]*channelMgr
}

var mgr = newRequestMgr()

// Get returns the RequestMgr for the given channel
func Get(channelID string) RequestMgr {
	return mgr.forChannel(channelID)
}

func newRequestMgr() *requestMgr {
	return &requestMgr{
		mgrs: make(map[string]*channelMgr),
	}
}

func (m *requestMgr) forChannel(channelID string) RequestMgr {
	m.mutex.RLock()
	cm, ok := m.mgrs[channelID]
	m.mutex.RUnlock()

	if ok {
		return cm
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	cm = newChannelMgr(channelID)
	m.mgrs[channelID] = cm
	return cm
}

type channelMgr struct {
	mutex         sync.RWMutex
	channelID     string
	requests      map[uint64]*request
	nextRequestID uint64
}

func newChannelMgr(channelID string) *channelMgr {
	logger.Debugf("[%s] Creating new channel request manager", channelID)
	return &channelMgr{
		channelID:     channelID,
		requests:      make(map[uint64]*request),
		nextRequestID: 1000000000,
	}
}

func (c *channelMgr) NewRequest() Request {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	reqID := c.newRequestID()

	logger.Debugf("[%s] Subscribing to transient data request %d", c.channelID, reqID)

	s := newRequest(reqID, c.channelID, c.remove)
	c.requests[reqID] = s

	return s
}

func (c *channelMgr) Respond(reqID uint64, response *Response) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	s, ok := c.requests[reqID]
	if !ok {
		logger.Debugf("[%s] No transient data requests for %d", c.channelID, reqID)
		return
	}

	logger.Debugf("[%s] Publishing transient data response %d", c.channelID, reqID)
	s.respond(response)
}

func (c *channelMgr) remove(reqID uint64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, ok := c.requests[reqID]; !ok {
		return
	}

	delete(c.requests, reqID)
	logger.Debugf("[%s] Unsubscribed from transient data request %d", c.channelID, reqID)
}

func (c *channelMgr) newRequestID() uint64 {
	return atomic.AddUint64(&c.nextRequestID, 1)
}

type request struct {
	reqID     uint64
	channelID string
	remove    func(reqID uint64)
	respChan  chan *Response
	done      bool
}

func newRequest(reqID uint64, channelID string, remove func(reqID uint64)) *request {
	return &request{
		reqID:     reqID,
		channelID: channelID,
		respChan:  make(chan *Response, 1),
		remove:    remove,
	}
}

func (r *request) ID() uint64 {
	return r.reqID
}

func (r *request) GetResponse(ctxt context.Context) (*Response, error) {
	if r.done {
		return nil, errors.New("request has already completed")
	}

	logger.Debugf("[%s] Waiting for response to request %d", r.channelID, r.ID())

	defer r.Cancel()

	select {
	case <-ctxt.Done():
		logger.Debugf("[%s] Request %d was timed out or cancelled", r.channelID, r.ID())
		return nil, ctxt.Err()
	case item := <-r.respChan:
		logger.Debugf("[%s] Got response for request %d", r.channelID, r.ID())
		return item, nil
	}
}

func (r *request) Cancel() {
	r.remove(r.reqID)
	r.done = true
}

func (r *request) respond(res *Response) {
	r.respChan <- res
}
