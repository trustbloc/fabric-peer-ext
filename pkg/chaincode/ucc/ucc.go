/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ucc

import (
	"sync"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/extensions/chaincode/api"
	"github.com/pkg/errors"
	"github.com/trustbloc/fabric-peer-ext/pkg/chaincode/builder"
	"github.com/trustbloc/fabric-peer-ext/pkg/resource"
)

var logger = flogging.MustGetLogger("ext_ucc")

var instance = newRegistry()

// Registry maintains a registry of in-process user chaincode
type Registry struct {
	mutex    sync.RWMutex
	creators []interface{}
	registry map[string]api.UserCC
}

func newRegistry() *Registry {
	r := &Registry{
		registry: make(map[string]api.UserCC),
	}
	resource.Register(r.Initialize)
	return r
}

// Register registers an in-process user chaincode creator function. The user chaincode
// will be initialized during peer startup with all of its declared dependencies.
func Register(ccCreator interface{}) {
	instance.addCreator(ccCreator)
}

// Get returns the in-process chaincode for the given ID
func Get(ccID string) (api.UserCC, bool) {
	return instance.Get(ccID)
}

// Chaincodes returns all registered in-process chaincodes
func Chaincodes() []api.UserCC {
	return instance.Chaincodes()
}

// WaitForReady blocks until the chaincodes are all registered
func WaitForReady() {
	instance.WaitForReady()
}

// Initialize is called on peer startup
func (r *Registry) Initialize() *Registry {
	logger.Info("Initializing in-process user chaincode registry")

	// Acquire a write lock. The lock will be released once
	//the chaincodes have all registered.
	r.mutex.Lock()

	go r.registerChaincodes()

	return r
}

type channelListener interface {
	ChannelJoined(channelID string)
}

// ChannelJoined is called when the peer joins a channel
func (r *Registry) ChannelJoined(channelID string) {
	logger.Infof("Channel joined [%s]", channelID)

	for _, cc := range r.Chaincodes() {
		l, ok := cc.(channelListener)
		if ok {
			logger.Infof("Notifying in-process user chaincode [%s] that channel [%s] was joined", cc.Name(), channelID)
			l.ChannelJoined(channelID)
		}
	}
}

func (r *Registry) registerChaincodes() {
	// mutex.Lock was called by the Initialize function and this function
	// will call mutex.Unlock after all chaincodes are registered.
	defer r.mutex.Unlock()

	b := builder.New()
	for _, c := range r.creators {
		b.Add(c)
	}

	descs, err := b.Build(resource.Mgr.Resources()...)
	if err != nil {
		panic(err)
	}

	logger.Infof("Registering [%d] in-process user chaincodes", len(descs))

	for _, desc := range descs {
		err = r.register(desc.(api.UserCC))
		if err != nil {
			panic(err)
		}
	}
}

func (r *Registry) addCreator(c interface{}) {
	r.creators = append(r.creators, c)
}

func (r *Registry) register(cc api.UserCC) error {
	logger.Infof("Registering in-process user chaincode [%s]", cc.Name())

	_, exists := r.registry[cc.Name()]
	if exists {
		return errors.Errorf("Chaincode already registered: [%s]", cc.Name())
	}

	r.registry[cc.Name()] = cc

	return nil
}

// Get returns the in-process chaincode for the given ID
func (r *Registry) Get(ccID string) (api.UserCC, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	cc, ok := r.registry[ccID]
	return cc, ok
}

// Chaincodes returns all registered in-process chaincodes
func (r *Registry) Chaincodes() []api.UserCC {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var ccs []api.UserCC

	for _, cc := range r.registry {
		ccs = append(ccs, cc)
	}

	return ccs
}

// WaitForReady blocks until the chaincodes are all registered
func (r *Registry) WaitForReady() {
	logger.Debugf("Waiting for chaincodes to be registered...")

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	logger.Debugf("... done registering chaincodes.")
}
