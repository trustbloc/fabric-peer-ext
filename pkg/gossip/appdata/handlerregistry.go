/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package appdata

import (
	"sync"

	gproto "github.com/hyperledger/fabric-protos-go/gossip"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("ext_dispatcher")

// Handler handles an application data request
type Handler func(channelID string, request *gproto.AppDataRequest) ([]byte, error)

// HandlerRegistry manages handlers for application-specific Gossip messages
type HandlerRegistry struct {
	mutex    sync.RWMutex
	handlers map[string]Handler
}

// NewHandlerRegistry returns a new HandlerRegistry
func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		handlers: make(map[string]Handler),
	}
}

// Register registers a handler for the given data type. Only one handler may be registered per data type,
// otherwise an error results
func (p *HandlerRegistry) Register(dataType string, handler Handler) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	_, exists := p.handlers[dataType]
	if exists {
		return errors.Errorf("handler for data type [%s] already registered", dataType)
	}

	p.handlers[dataType] = handler

	return nil
}

// HandlerForType returns the handler for the given data type
func (p *HandlerRegistry) HandlerForType(dataType string) (Handler, bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	handler, ok := p.handlers[dataType]
	return handler, ok
}
