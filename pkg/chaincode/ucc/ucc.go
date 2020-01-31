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
	registry map[chaincodeID]api.UserCC
}

func newRegistry() *Registry {
	r := &Registry{
		registry: make(map[chaincodeID]api.UserCC),
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
func Get(name, version string) (api.UserCC, bool) {
	return instance.Get(name, version)
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
			logger.Infof("Notifying in-process user chaincode [%s:%s] that channel [%s] was joined", cc.Name(), cc.Version(), channelID)
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
	if err := getVersion(cc).Validate(); err != nil {
		return errors.WithMessagef(err, "validation error for in-process user chaincode [%s:%s]", cc.Name(), cc.Version())
	}

	ccID := newChaincodeID(cc.Name(), cc.Version())

	logger.Infof("Registering in-process user chaincode [%s]", ccID)

	_, exists := r.registry[ccID]
	if exists {
		return errors.Errorf("chaincode already registered: [%s]", ccID)
	}

	r.registry[ccID] = cc

	return nil
}

// Get returns the in-process chaincode for the given ID
func (r *Registry) Get(name, version string) (api.UserCC, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	ccID := newChaincodeID(name, version)

	cc, ok := r.registry[ccID]
	if ok {
		logger.Debugf("Found exact match for in-process user chaincode [%s]", ccID)
		return cc, true
	}

	for _, cc := range r.registry {
		if cc.Name() != name {
			continue
		}

		if getVersion(cc).Matches(version) {
			logger.Debugf("Found in-process user chaincode that matches the desired version [%s:%s]: [%s]", name, version, cc.Name(), cc.Version())
			return cc, true
		}
	}

	logger.Debugf("Did NOT find any matching in-process user chaincode for [%s]", ccID)

	return nil, false
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

type chaincodeID struct {
	name    string
	version string
}

func newChaincodeID(name, version string) chaincodeID {
	return chaincodeID{
		name:    name,
		version: version,
	}
}

// String returns the string representation of the chaincode ID
func (c chaincodeID) String() string {
	return c.name + ":" + c.version
}

func getVersion(cc api.UserCC) Version {
	return Version(cc.Version())
}
